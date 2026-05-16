# Lab 9：sync、atomic 与 Go Memory Model

本实验配套 Lesson 9，观察 Mutex、RWMutex、Once、WaitGroup、atomic、atomic.Value、channel happens-before，以及同步原语 benchmark。

## 运行命令

```bash
cd labs/09-sync-atomic-memory-model

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

如要演示 race detector，可临时写一个对 `UnsafeCounter` 并发 `Inc` 的测试，然后运行：

```bash
go test -race ./...
```

## 观察目标

1. MutexCounter 和 AtomicCounter 都能修复简单计数 data race。
2. RWMutex 适合保护共享 map/cache。
3. sync.Once 保证 lazy init 只执行一次。
4. WaitGroup 的 `Add` 应在启动 goroutine 前完成。
5. atomic.Value 适合发布不可变配置快照。
6. channel close/receive 可以建立 happens-before。
7. Mutex、Atomic、Channel counter benchmark 差异明显。

## 生产启发

- 优先用清晰同步保证正确性，再用 benchmark 判断性能。
- atomic 适合简单变量，不适合复杂不变量。
- 发布配置快照时要 clone 内部 map/slice，避免发布后被修改。
- 所有并发代码都应纳入 `go test -race`。
