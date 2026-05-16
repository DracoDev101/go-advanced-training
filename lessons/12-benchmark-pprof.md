# Lesson 12：Benchmark 与 pprof 性能诊断

## 学习目标

完成本课后，学习者应该能够：

- 写出可信 Go benchmark，避免编译器优化、输入不稳定、I/O 干扰和数据规模失真。
- 解释 `ns/op`、`B/op`、`allocs/op`、吞吐、tail latency 的差异。
- 使用 CPU、heap、allocs、block、mutex profile 定位不同类型瓶颈。
- 用 `benchstat` 比较优化前后结果，而不是凭单次 benchmark 下结论。
- 在 Production Job Runner 中建立“指标发现 → profile 定位 → benchmark 验证 → 回归保护”的性能闭环。

---

## 关键问题

1. 什么样的 benchmark 是不可信的？
2. 为什么微基准快，不代表接口整体快？
3. CPU profile 中 flat、cum 分别代表什么？
4. heap profile 和 allocs profile 有什么区别？
5. 什么时候用 benchmark，什么时候用 load test，什么时候用 production profile？
6. 性能优化如何避免改坏可读性和可靠性？

---

## 核心结论

- benchmark 是实验，不是仪式；必须有稳定输入、明确假设、可重复对比。
- pprof 的作用是缩小搜索空间：先找热点，再决定是否优化。
- CPU 热点、分配热点、锁等待、阻塞等待是四种不同问题，不能用同一个 profile 判断。
- 优化必须有 baseline 和 after，并用 `benchstat` 或线上指标证明收益。
- 没有指标的性能优化通常是在制造复杂度。

---

## 1. Benchmark 基本形态和指标含义

```go
func BenchmarkEncodeJob(b *testing.B) {
    job := sampleJob()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        _, err := EncodeJob(job)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

```bash
go test -bench=BenchmarkEncodeJob -benchmem ./...
```

| 指标 | 含义 |
|---|---|
| `ns/op` | 每次操作平均耗时 |
| `B/op` | 每次操作分配字节数 |
| `allocs/op` | 每次操作分配次数 |
| `-8` | GOMAXPROCS / CPU 并行信息，不是 goroutine 数 |

---

## 2. 常见 benchmark 陷阱

编译器把结果优化掉：应保存到包级 `sink`。把准备数据算进 benchmark：应准备后 `b.ResetTimer()`。随机输入导致结果不稳定：benchmark 输入要固定。微基准替代系统压测：`EncodeJob` 很快不代表 HTTP endpoint 快，端到端还有 middleware、DB、队列、日志、网络。

---

## 3. 用 benchstat 做优化前后对比

```bash
go test -bench=. -benchmem -count=10 ./... > old.txt
# 修改代码
go test -bench=. -benchmem -count=10 ./... > new.txt
benchstat old.txt new.txt
```

看三件事：变化是否显著；`ns/op` 是否改善；`B/op`、`allocs/op` 是否下降或可解释。不要用单次运行的 5% 波动做决策。

---

## 4. pprof：不同 profile 回答不同问题

| Profile | 回答的问题 | 典型命令 |
|---|---|---|
| CPU | CPU 时间花在哪里 | `go test -cpuprofile cpu.out` |
| heap | 当前存活对象在哪里 | `-memprofile mem.out` |
| allocs | 历史分配热点在哪里 | `/debug/pprof/allocs` |
| goroutine | goroutine 在哪里 | `/debug/pprof/goroutine?debug=2` |
| block | 同步阻塞在哪里 | `-blockprofile block.out` |
| mutex | 锁等待在哪里 | `-mutexprofile mutex.out` |
| trace | 调度、网络、同步时间线 | `-trace trace.out` |

---

## 5. CPU profile 怎么读

```bash
go test -run=^$ -bench=BenchmarkExecuteJob -cpuprofile cpu.out ./...
go tool pprof cpu.out
```

常用命令：`top`、`list ExecuteJob`、`web`、`peek json`。

| 字段 | 含义 |
|---|---|
| flat | 函数自身消耗的 CPU |
| cum | 函数自身 + 子调用累计 CPU |

`flat` 高说明函数内部计算重；`cum` 高但 `flat` 低说明下游重；`runtime.mallocgc` 高通常表示分配导致 CPU 消耗。

---

## 6. Heap / allocs profile 怎么读

```bash
go test -run=^$ -bench=BenchmarkExecuteJob -benchmem -memprofile mem.out ./...
go tool pprof -alloc_space mem.out
go tool pprof -inuse_space mem.out
```

| 视角 | 含义 | 用途 |
|---|---|---|
| `alloc_space` | 历史累计分配 | 找分配速率热点 |
| `inuse_space` | 当前仍存活 | 找泄漏/长期持有 |

如果 `alloc_space` 高但 `inuse_space` 不高，说明对象很快被回收，问题是 GC 压力而不是泄漏。

---

## 7. 从线上指标到 profile 的排查路径

```text
1. 指标发现：P99 上升 / CPU 高 / RSS 高 / queue depth 高
2. 分类：CPU bound? memory pressure? lock contention? downstream blocking?
3. 采集对应 profile
4. 定位 top hotspot
5. 写 benchmark 或 regression test 重现
6. 优化最小代码路径
7. benchstat + 线上指标验证
```

不要反过来：先打开 pprof，看哪里显眼就改哪里。

---

## 8. Production Job Runner 落地

关键 benchmark：job payload decode/validate、job claim SQL、worker execute wrapper、result serialization、event publish envelope、structured log fields construction。

关键 profile：CPU 看 worker 是否被 JSON/压缩占满；heap 看 result 是否长期持有大对象；block 看 worker 是否卡在队列/DB pool；mutex 看状态缓存或 metrics registry 是否锁竞争。

性能预算示例：SubmitJob handler P95 < 50ms；ClaimJob DB tx P95 < 20ms；Worker wrapper overhead < 1ms/job；Result summary <= 32KB。

---

## 代码实验

配套目录：`labs/12-benchmark-pprof/`

```bash
go test -bench=. -benchmem -count=10 ./... > old.txt
go test -run=^$ -bench=. -cpuprofile cpu.out -memprofile mem.out ./...
go tool pprof -http=:0 cpu.out
go tool pprof -http=:0 -alloc_space mem.out
```

---

## 常见误区与故障案例

误区：只看 `ns/op`；benchmark 输入太小；用 `fmt.Sprintf`、`time.Now`、随机数污染热路径；没有 `-count` 和 `benchstat` 就比较 3% 差异；为微小性能牺牲边界清晰。

案例：接口 P99 高，开发者优化 JSON 编码，benchmark 提升 20%，线上无变化。线上 block profile 显示大量时间等待 DB connection pool。真实问题是 DB pool 太小和事务持有时间过长。

---

## 作业与评估

作业：写 bad benchmark 和 good benchmark；对比 `alloc_space` 与 `inuse_space`；为 Production Job Runner 写 5 个性能预算和对应 profile 方法。

评估：能写可信 benchmark；能用 pprof top/list 定位热点；能解释 CPU、heap、allocs、block、mutex profile 的差异；能用 benchstat 证明优化收益。
