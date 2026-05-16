# Lab 6：Goroutine 与 GMP 调度模型

本实验配套 Lesson 6，观察有界 worker pool、context 取消、goroutine 生命周期、`GOMAXPROCS` 和 scheduler trace。

## 运行命令

```bash
cd labs/06-goroutine-scheduler

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
GODEBUG=schedtrace=1000,scheddetail=1 go test -run TestWorkerPoolProcessesJobs ./...
```

也可以改变 P 数量：

```bash
GOMAXPROCS=1 go test -bench=BenchmarkBusyWork -benchmem ./...
GOMAXPROCS=4 go test -bench=BenchmarkBusyWork -benchmem ./...
```

## 观察目标

1. worker pool 用固定 worker 数处理任务，避免无界 goroutine。
2. 某个任务返回错误后，context 取消会传播给其他 worker。
3. sender 如果阻塞在无人接收的 channel 上会泄漏；监听 context 可以退出。
4. `runtime.NumGoroutine()` 可辅助观察泄漏趋势。
5. `GODEBUG=schedtrace=...` 可观察 `gomaxprocs`、`threads`、`runqueue` 等字段。

## 生产启发

- `go f()` 之前先设计 owner、退出条件和错误通道。
- 批量任务不要无界 goroutine，优先 worker pool/semaphore/errgroup limit。
- channel send/receive 可能永久阻塞，长生命周期 goroutine 应监听 context。
- goroutine count、queue depth、active workers 应成为生产指标。
