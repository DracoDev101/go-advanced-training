# Lab 10：并发 Debugging 与 Race Detector

本实验把 Lesson 10 的并发故障拆成四类可观察现象：data race、goroutine leak、block、mutex contention。

## 运行

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 1. Race detector

安全版本：

```bash
go test -race -run TestSafeCounterConcurrent ./...
```

手动观察 race 报告：打开 `TestUnsafeCounterRaceManual` 中的 `t.Skip`，再运行：

```bash
go test -race -run TestUnsafeCounterRaceManual ./...
```

观察报告里的 read stack、write stack、goroutine created at。

## 2. Goroutine leak

`CancellableSend` 展示生产写法：任何可能阻塞的 channel send 都必须带 `ctx.Done()` 分支。

```bash
go test -run TestCancellableSendDoesNotLeak -count=1 ./...
```

## 3. Block profile

```bash
go test -run TestCancellableSendDoesNotLeak -blockprofile block.out ./...
go tool pprof -http=:0 block.out
```

block profile 用来观察 channel/send/recv/select/cond/wait 等同步阻塞。

## 4. Mutex profile

```bash
go test -run '^$' -bench BenchmarkContendedMutex -mutexprofile mutex.out ./...
go tool pprof -http=:0 mutex.out
```

mutex profile 用来观察锁等待，不要用 CPU profile 代替锁竞争分析。

## 生产启发

Production Job Runner 的 worker 必须满足：

- context cancel 后退出。
- jobs channel close 后退出。
- 发送结果、写事件、提交状态等阻塞点必须有 timeout/cancel 策略。
- race detector 覆盖 in-memory state；DB claim 竞态用条件更新/事务解决。
