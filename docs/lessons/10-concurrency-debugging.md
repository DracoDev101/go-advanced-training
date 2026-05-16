# Lesson 10：并发 Debugging 与 Race Detector

## 学习目标

完成本课后，学习者应该能够：

- 区分 **data race、race condition、deadlock、goroutine leak、lock contention、channel blocking**。
- 用 `go test -race`、goroutine dump、block profile、mutex profile、`go tool trace` 建立并发故障证据链。
- 读懂 race detector 报告中的读写栈、goroutine 创建栈和冲突变量。
- 为 worker、channel、mutex、context 设计可验证的生命周期边界。
- 在 Production Job Runner 中定位“任务不消费、CPU 不高但延迟高、goroutine 越跑越多”等问题。

---

## 关键问题

1. 为什么没有 data race 的程序仍然可能出现业务竞态？
2. `go test -race` 能发现什么，不能发现什么？
3. goroutine leak 的本质是“谁在等待谁”？
4. deadlock、长期阻塞、锁竞争在 profile 上分别长什么样？
5. 线上出现 queue depth 上升但 CPU 很低时，优先看哪些证据？
6. 如何把并发 bug 从“偶现”变成稳定可复现的测试？

---

## 核心结论

- **并发 Debugging 的第一步不是猜锁，而是分类症状**：错误结果、卡死、变慢、资源增长，对应不同工具。
- **race detector 只证明发生了未同步的并发访问**；它不证明业务逻辑正确，也不能覆盖没有执行到的路径。
- **goroutine leak 通常是生命周期协议错误**：发送方没人接、接收方没人关、context 没传、worker 没退出。
- **block profile 看等待 channel / select / cond 的时间；mutex profile 看锁竞争导致的等待时间**。
- 生产排查要把 **goroutine dump + profile + 指标 + 日志字段** 串起来，而不是只看单个截图。

---


## 0. 系统化排查流程：不要从修复开始

并发和性能问题最容易陷入“猜一个原因然后改代码”。本课要求按证据推进：

```text
1. 现象分型：错误结果 / 卡死 / 变慢 / 泄漏 / 队列堆积
2. 影响面：单实例、单 endpoint、单 job kind，还是全局
3. 时间线：是否和发布、配置、流量、依赖故障相关
4. 证据包：日志 + 指标 + goroutine dump + 对应 profile
5. 缩小组件：API / worker / DB / queue / downstream / runtime
6. 单一假设：说明“我认为 X 是根因，因为证据 Y”
7. 最小验证：定向测试、benchmark、profile 对比或故障复现
8. 修复根因：补回归测试和观测信号
```

首轮证据采集命令：

```bash
# goroutine dump：卡死、泄漏、CPU 低但延迟高
curl -s http://127.0.0.1:6060/debug/pprof/goroutine?debug=2 > goroutine.txt

# CPU profile：CPU 高或整体慢
curl -s 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30' > cpu.out

# heap / allocs：内存上涨或 GC 压力
curl -s http://127.0.0.1:6060/debug/pprof/heap > heap.out
curl -s http://127.0.0.1:6060/debug/pprof/allocs > allocs.out

# mutex / block：锁竞争或同步阻塞，需要程序开启采样率
curl -s http://127.0.0.1:6060/debug/pprof/mutex > mutex.out
curl -s http://127.0.0.1:6060/debug/pprof/block > block.out
```

分析入口：

```bash
go tool pprof -http=:0 cpu.out
go tool pprof -http=:0 -alloc_space allocs.out
go tool pprof -http=:0 -inuse_space heap.out
go tool pprof -http=:0 mutex.out
go tool pprof -http=:0 block.out
```

完整手册见：`references/troubleshooting-playbook.md`。

---

## 1. 问题分类：同样是“并发出问题”，其实是四类问题

| 症状 | 常见根因 | 主要工具 |
|---|---|---|
| 结果偶尔不对 | data race、业务竞态、循环变量捕获 | `go test -race`、定向单测 |
| 程序卡住 | channel 双方不匹配、锁顺序反转、WaitGroup 计数错误 | goroutine dump、block profile |
| 延迟升高但 CPU 不高 | 锁竞争、channel 阻塞、连接池耗尽 | mutex/block profile、runtime trace |
| 内存/goroutine 持续增长 | goroutine leak、timer leak、未关闭 response body | goroutine/heap profile、指标趋势 |

生产里更常见的不是全局 deadlock，而是 **部分 worker 卡住、吞吐下降、队列堆积、泄漏缓慢增长**。

---

## 2. Data race vs Race condition

### Data race

两个 goroutine 并发访问同一内存地址，至少一个是写，并且没有 happens-before 关系。

```go
var n int

go func() { n++ }()
go func() { n++ }()
```

这类问题由 Go race detector 直接定位。

### Race condition

程序没有违反 memory model，但业务结果依赖不稳定时序。

```go
if repo.Status(jobID) == "pending" {
    repo.MarkRunning(jobID)
}
```

每个 DB 调用本身都线程安全，但两个 worker 可能同时看到 `pending`，导致重复执行。修复方式不是 Go mutex，而是 DB 条件更新：

```sql
UPDATE jobs
SET status = 'running'
WHERE id = $1 AND status = 'pending';
```

**重点**：race detector 解决不了分布式/数据库层面的竞态，它只看进程内内存访问。

---

## 3. Race Detector：怎么用、怎么看、有什么边界

```bash
go test -race ./...
go test -race -run TestName -count=100 ./...
go test -race -run TestName -shuffle=on ./...
```

典型报告包含三段：

```text
WARNING: DATA RACE
Read at 0x... by goroutine 8:
  package.(*Cache).Get()

Previous write at 0x... by goroutine 7:
  package.(*Cache).Set()

Goroutine 8 created at:
  package.TestCache()
```

读报告顺序：

1. 找冲突地址对应的变量。
2. 看 read/write 栈，确认哪个路径没有锁或 atomic。
3. 看 goroutine created at，确认生命周期来源。
4. 修复后用 `-race -count=100` 验证。

边界：只检查运行到的代码路径；不能发现业务竞态、死锁、泄漏；有明显性能开销，不适合长期线上开启。

---

## 4. Goroutine leak：定位“谁在等谁”

典型泄漏：发送方被卡住。

```go
func start(out chan<- Result) {
    go func() {
        out <- slowCall() // 如果没人接收，这个 goroutine 永远不退出
    }()
}
```

修复原则：

```go
select {
case out <- result:
case <-ctx.Done():
    return
}
```

生产排查：

```bash
curl http://localhost:6060/debug/pprof/goroutine?debug=2 > goroutine.txt
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

看 dump 时先按栈签名聚类：

```text
2000 goroutines blocked on chan send at worker.go:83
500 goroutines blocked on database/sql.(*DB).conn
```

---

## 5. Deadlock、block profile、mutex profile

全局 deadlock 才会触发：

```text
fatal error: all goroutines are asleep - deadlock!
```

但生产里更常见的是 **局部 deadlock**：部分 goroutine 永远等待，进程仍然活着。

Block profile 观察 goroutine 在同步原语上等待的时间：

```bash
go test -run TestX -blockprofile block.out ./...
go tool pprof -http=:0 block.out
```

Mutex profile 观察锁等待：

```bash
go test -run TestX -mutexprofile mutex.out ./...
go tool pprof -http=:0 mutex.out
```

判断原则：CPU 不高、延迟高、mutex profile 高 → 锁竞争；block profile 高 → channel/cond/wait 设计问题。

---

## 6. go tool trace 看什么

```bash
go test -run TestX -trace trace.out ./...
go tool trace trace.out
```

重点看：Goroutine analysis、Network blocking、Synchronization blocking、Scheduler latency。`trace` 不是第一工具，优先顺序通常是：指标 → goroutine dump → pprof → trace。

---

## 7. Production Job Runner 落地

worker 生命周期协议：

```text
start(ctx) -> claim job -> execute with timeout -> persist result -> publish event -> exit when ctx canceled
```

必须有的观测字段：`job_id, worker_id, attempt, status, queue_depth, duration_ms, error_kind`。

必须有的测试：context cancel 后 worker 退出；job channel close 后 worker 退出；下游阻塞时不会泄漏 goroutine；同一个 job 不会被两个 worker 同时 claim。

---

## 代码实验

配套目录：`labs/10-concurrency-debugging/`

建议升级为专项实验：

```bash
go test -race -run TestUnsafeCounter ./...
go test -run TestGoroutineLeak -count=1 ./...
go test -run TestBlockedChannel -blockprofile block.out ./...
go test -run TestMutexContention -mutexprofile mutex.out ./...
go tool pprof -http=:0 block.out
```

---

## 常见误区

1. 以为 `-race` 通过就代表并发逻辑正确。
2. 用 `time.Sleep` 等 goroutine 结束，导致测试偶现。
3. 只在发送方关闭 channel，忘记接收方退出协议。
4. 锁竞争时盲目换 atomic，反而破坏不变量。

---

## 故障案例

现象：`/healthz` 正常，CPU 低，queue depth 持续升高。goroutine dump 显示大量 worker 卡在 `results <- r`；block profile 指向同一行。根因是结果 channel 没有消费者，发送路径没有 `ctx.Done()` 分支。修复：发送结果时使用 `select`，关闭 worker 时先停止生产，再 drain 或丢弃结果，并记录 drop 指标。

---

## 作业与评估

作业：增加一个泄漏 goroutine 的版本和修复版本；用 block profile 证明阻塞点变化；写一个 DB claim job 的并发测试。

评估：能读懂 race detector 报告；能从 goroutine dump 聚类定位阻塞点；能区分 block profile 和 mutex profile。
