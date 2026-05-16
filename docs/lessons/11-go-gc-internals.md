# Lesson 11：Go GC 原理与调优边界

## 学习目标

完成本课后，学习者应该能够：

- 解释 Go GC 为什么是并发、三色标记、非分代、非移动的 GC。
- 读懂 `GODEBUG=gctrace=1` 中 heap、goal、STW、mark assist 等关键信号。
- 区分 **内存泄漏、分配速率过高、短期尖峰、GC CPU 过高、RSS 不下降**。
- 判断应优先减少分配、复用对象、调 `GOGC`，还是设置 `GOMEMLIMIT`。
- 在 Production Job Runner 中控制 job payload、日志字段、批处理、缓存带来的 GC 压力。

---

## 关键问题

1. Go GC 到底在回收什么？栈、堆、全局变量分别如何参与？
2. 为什么“减少分配速率”通常比“调大 GOGC”更重要？
3. `GOGC=100` 表示什么？它不是“每 100ms GC 一次”。
4. STW 是否还重要？现代 Go 的主要 GC 成本在哪里？
5. `GOMEMLIMIT` 解决什么问题，又可能带来什么副作用？
6. 为什么 RSS 不下降不一定代表 Go 对象泄漏？

---

## 核心结论

- Go GC 的核心优化目标不是“零暂停”，而是 **在低暂停和可控 CPU 成本之间平衡**。
- 生产中最常见的问题不是 GC 算法不行，而是代码制造了过高的 **allocation rate**。
- `GOGC` 控制下一轮 GC 目标堆大小：活跃堆越大，允许增长越多；调大它通常用内存换 CPU。
- `GOMEMLIMIT` 是软内存限制，适合容器环境，但限制太紧会导致频繁 GC 和吞吐下降。
- 优化顺序：**profile 证明分配热点 → 减少分配/缩短对象生命周期 → 再调 GOGC/GOMEMLIMIT**。

---

## 1. Go GC 回收的对象：先理解堆

Go 变量不等于堆对象。变量可能在栈上，也可能逃逸到堆上。

```go
func f() *User {
    u := User{Name: "a"}
    return &u // u 逃逸到堆
}
```

GC 主要管理堆对象。栈会随着 goroutine 生命周期增长/收缩，栈上的指针会作为 root 被扫描。

GC roots 包括：goroutine 栈上的指针、全局变量中的指针、runtime 内部结构中的指针、finalizer/cgo 等特殊 root。

---

## 2. 三色标记和 write barrier

三色抽象：

```text
白色：尚未发现，最终可能被回收
灰色：已发现，但它指向的对象还没扫描完
黑色：已发现，并且它指向的对象也扫描完
```

Go GC 是并发标记：应用 goroutine 和 GC 同时运行。应用在 GC 标记期间仍然会修改指针：

```go
obj.child = other
```

如果没有屏障，可能出现黑对象指向白对象，但白对象没有被扫描到，导致活对象被错误回收。write barrier 的作用是让指针写入在 GC 期间被 runtime 记录。

生产理解：大量指针对象、复杂对象图、频繁指针写入会增加扫描和屏障成本；`[]byte` 这类无指针数据比 `[]*T` 对 GC 更友好。

---

## 3. GC pacing、GOGC 与 allocation rate

`GOGC=100` 的含义：下一轮 GC 目标堆大小大约是：

```text
goal = live_heap * (1 + GOGC/100)
```

如果上一轮 GC 后 live heap 是 200MB，`GOGC=100`，下一轮目标约 400MB。

| 指标 | 含义 | 优化方向 |
|---|---|---|
| live heap | GC 后仍然存活的对象 | 减少长生命周期对象、缓存上限 |
| allocation rate | 单位时间分配量 | 减少临时对象、复用 buffer |
| GC CPU fraction | GC 消耗 CPU 比例 | 降低分配或调高 GOGC |
| heap goal | 下一轮 GC 目标 | 由 live heap 和 GOGC 决定 |

如果 allocation rate 很高，GC 会被迫更频繁地工作。调大 GOGC 只能降低频率，但会增加内存占用。

---

## 4. Mark assist：为什么业务 goroutine 会被迫帮 GC

当程序分配速度超过 GC 进度时，runtime 会让正在分配的 goroutine 做一部分标记工作，这叫 mark assist。

表现：请求 tail latency 升高；CPU profile 中出现 GC 相关栈；gctrace 中 GC CPU 压力变大。含义不是“GC 卡住了程序”，而是程序分配太快，欠了 GC 的账。

---

## 5. 读懂 gctrace

```bash
GODEBUG=gctrace=1 go test -run TestAllocPressure ./...
GODEBUG=gctrace=1 go run ./cmd/server
```

典型输出：

```text
gc 12 @4.232s 3%: 0.08+12+0.05 ms clock, 0.6+4.1/20/0+0.4 ms cpu, 64->80->40 MB, 82 MB goal, 8 P
```

| 片段 | 关注点 |
|---|---|
| `gc 12` | 第几次 GC，频率是否异常 |
| `3%` | GC CPU 占比 |
| `0.08+12+0.05 ms` | STW + concurrent mark + STW |
| `64->80->40 MB` | GC 前、GC 峰值、GC 后 live heap |
| `82 MB goal` | 下一轮目标堆 |

判断：`after GC` 持续上升可能是真实存活对象增长或泄漏；`before GC` 很高但 `after GC` 稳定说明分配速率高，但不一定泄漏。

---

## 6. GOMEMLIMIT：容器环境下的软限制

```bash
GOMEMLIMIT=512MiB ./server
```

适用：Kubernetes/container 有明确内存 limit，希望 Go runtime 在接近 limit 前更积极 GC。

风险：limit 设得过低会导致 GC 频繁运行，吞吐下降；它限制的是 Go runtime 管理的内存目标，不等于进程 RSS 的硬上限；cgo、mmap、文件缓存等不完全受它控制。

---

## 7. Production Job Runner 落地

GC 压力来源：job payload 过大；worker 每次执行构造大量临时 JSON/map/log fields；无上限缓存保存 job result；批处理一次拉太多 job。

设计原则：payload 外置化；batch 有上限；日志字段固定；缓存有容量；保留受控 `/debug/pprof`。

---

## 代码实验

配套目录：`labs/11-go-gc-internals/`

建议升级命令：

```bash
go test -bench=. -benchmem ./...
GODEBUG=gctrace=1 go test -bench=BenchmarkAllocHeavy -run=^$ ./...
go test -run=^$ -bench=BenchmarkAllocHeavy -memprofile mem.out ./...
go tool pprof -http=:0 mem.out
```

---

## 常见误区与故障案例

误区：看到 RSS 不下降就断言泄漏；把 `GOGC` 当成时间间隔；优先调 GC 参数而不是减少分配；滥用 `sync.Pool` 保存业务状态。

案例：任务系统 P99 周期性抖动，gctrace 显示 GC 频率升高，heap profile 指向 job result JSON 序列化。修复：固定日志字段结构，限制 result 摘要长度，批量处理加上限，优化后 `allocs/op` 和 GC 频率下降。

---

## 作业与评估

作业：用 benchmark 对比 `map[string]any` 日志字段和固定 struct 字段；解释一次 gctrace 输出；写 Production Job Runner 内存预算。

评估：能解释 GOGC、live heap、allocation rate 的关系；能用 gctrace 判断泄漏还是分配速率高；能提出减少 GC 压力的代码级和架构级方案。
