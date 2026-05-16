# Lesson 6：Goroutine 与 GMP 调度模型

## 学习目标

完成本课后，学习者应该能够：

- 解释 goroutine 与 OS thread 的区别。
- 解释 Go runtime 的 GMP 调度模型：G、M、P 分别代表什么。
- 理解 work stealing、local run queue、global run queue、netpoller 的基本作用。
- 理解 goroutine stack 从小栈开始并可增长的原因。
- 判断 `GOMAXPROCS` 影响什么、不影响什么。
- 使用 `GODEBUG=schedtrace=...,scheddetail=1` 观察调度状态。
- 识别 goroutine 泄漏、无界 goroutine、阻塞 syscall 等生产风险。
- 在生产服务中为 goroutine 设计明确生命周期、取消路径和并发上限。

---

## 关键问题

1. goroutine 为什么比 OS thread 轻？
2. goroutine 很轻，是否意味着可以无限创建？
3. G、M、P 分别是什么？
4. `GOMAXPROCS` 控制的是 goroutine 数量还是并行执行 Go 代码的 P 数量？
5. 阻塞网络 IO 为什么不会轻易卡死所有 goroutine？
6. syscall、cgo、长时间 CPU loop 对调度有什么影响？
7. goroutine 泄漏为什么比普通内存泄漏更隐蔽？
8. 生产代码如何管理 goroutine 生命周期？

---

## 核心结论

- goroutine 是 Go runtime 管理的轻量级并发执行单元，不等同于 OS thread。
- Go 使用 M:N 调度：多个 goroutine 映射到多个 OS thread。
- G 表示 goroutine，M 表示 machine/OS thread，P 表示 processor/执行 Go 代码所需的调度资源。
- `GOMAXPROCS` 控制可同时执行 Go 代码的 P 的数量，通常默认等于可用 CPU 数。
- goroutine 初始栈很小，并可按需增长，因此创建成本远低于 OS thread。
- 网络 IO 通常通过 runtime netpoller 集成，不会为每个阻塞连接长期占用一个 OS thread。
- goroutine 轻量但不是免费；无界创建会带来内存、调度、GC 和下游压力。
- 生产级 goroutine 必须有 owner、退出条件、取消信号、错误处理和观测指标。

---

## 设计哲学

Go 把并发作为语言与 runtime 的一等能力。它并不是简单包装 OS thread，而是通过 goroutine + channel/context/sync 让开发者以较低成本表达并发任务。

Go 的并发哲学可以概括为：

```text
让并发写起来便宜，但让正确性依然显式。
```

也就是说，`go f()` 很容易，但这不意味着 goroutine 生命周期可以不设计。

生产系统里，真正困难的不是“启动 goroutine”，而是：

```text
它什么时候退出？
谁取消它？
错误怎么返回？
是否有并发上限？
是否会阻塞发送？
是否能被观测？
```

---

## 1. Goroutine 与 OS Thread

OS thread 通常由内核调度，栈较大，创建和上下文切换成本较高。

goroutine 由 Go runtime 调度，初始栈较小，可增长，可被 runtime multiplex 到较少的 OS thread 上。

简化对比：

| 维度 | goroutine | OS thread |
|---|---|---|
| 调度者 | Go runtime | OS kernel |
| 初始栈 | 小，可增长 | 通常较大 |
| 创建成本 | 低 | 高 |
| 数量级 | 可轻松成千上万 | 通常较少 |
| 阻塞 IO | 可与 netpoller 配合 | thread 阻塞 |

但 goroutine 不是免费：

- 每个 goroutine 仍有栈、调度元数据。
- goroutine 持有引用会影响 GC。
- 大量 runnable goroutine 会增加调度开销。
- 大量阻塞 goroutine 可能表示下游背压失效。

---

## 2. GMP 模型

### 2.1 G：Goroutine

G 保存 goroutine 的执行状态：

- 栈信息。
- 当前 PC/SP。
- 调度状态。
- 等待原因。
- 与 panic/defer 等相关的数据。

### 2.2 M：Machine

M 是 OS thread 的抽象。真正执行代码的是 M。

M 可能：

- 执行 Go 代码。
- 进入 syscall。
- 运行 cgo。
- 被阻塞或空闲。

### 2.3 P：Processor

P 是执行 Go 代码所需的调度资源。一个 M 必须绑定一个 P，才能执行 Go goroutine。

`GOMAXPROCS` 控制 P 的数量。

```bash
GOMAXPROCS=2 go test ./...
```

表示最多同时有 2 个 P 执行 Go 代码，不表示只能有 2 个 goroutine，也不表示只能有 2 个 OS thread。

---

## 3. 调度过程简化模型

每个 P 有一个 local run queue。还有一个 global run queue。

简化流程：

```text
G 变为 runnable
    ↓
进入某个 P 的 local run queue 或 global run queue
    ↓
M 绑定 P
    ↓
M 从 P 的队列取 G 执行
```

如果某个 P 没有可运行 G，它可能从其他 P 的 local queue 偷任务，这就是 work stealing。

### 3.1 为什么需要 local queue？

减少全局锁竞争，提高局部性。

### 3.2 为什么需要 work stealing？

让负载更均衡，避免某些 P 忙、某些 P 闲。

---

## 4. Netpoller 与阻塞 IO

Go 的网络库与 runtime netpoller 集成。

当 goroutine 等待网络 IO 时，它通常会被挂起，M 可以去执行其他 goroutine。等 IO 就绪，runtime 再把对应 goroutine 标记为 runnable。

这就是 Go 能高效处理大量连接的重要原因之一。

注意：

- 普通网络 IO 通常由 netpoller 管理。
- 某些阻塞 syscall、文件 IO、cgo 可能占用 OS thread。
- runtime 会尽力补充 M，但这不代表阻塞没有代价。

---

## 5. Preemption：抢占

早期 Go 对长时间 CPU loop 的抢占能力较弱。现代 Go 已支持 async preemption，长时间运行的 goroutine 更容易被抢占。

但仍然建议：

- CPU 密集循环中定期检查 context。
- 不要在单个 goroutine 中做无边界计算。
- 对 CPU heavy job 做并发上限。
- 结合 pprof 观察 CPU hotspot。

---

## 6. Goroutine Stack

goroutine 初始栈很小，并可按需增长。

这让创建大量 goroutine 成为可能，但栈增长和栈扫描仍有成本。

常见问题：

- 深递归可能导致大量栈增长。
- goroutine 持有大对象引用会延长对象生命周期。
- 泄漏 goroutine 会泄漏其栈和引用对象。

---

## 7. 如何观察调度器

使用：

```bash
GODEBUG=schedtrace=1000,scheddetail=1 go run ./cmd/sched-demo
```

常见字段：

```text
SCHED 1000ms: gomaxprocs=4 idleprocs=0 threads=8 spinningthreads=1 needspinning=0 idlethreads=3 runqueue=12
```

关注：

| 字段 | 含义 |
|---|---|
| `gomaxprocs` | P 的数量 |
| `idleprocs` | 空闲 P 数量 |
| `threads` | runtime 管理的 OS thread 数量 |
| `idlethreads` | 空闲 thread 数量 |
| `runqueue` | global run queue 长度 |
| `spinningthreads` | 正在寻找工作的 thread |

如果 runnable goroutine 长期堆积，说明 CPU 或调度资源可能不足，或任务生产速度超过消费速度。

---

## 8. 生产实践

### 8.1 goroutine 必须有生命周期设计

启动 goroutine 前问：

```text
谁拥有它？
它什么时候退出？
它如何收到取消？
错误如何汇报？
是否需要等待它完成？
```

### 8.2 不要无界创建 goroutine

反例：

```go
for _, item := range items {
    go process(item)
}
```

如果 `items` 很大，可能瞬间打爆：

- CPU。
- 内存。
- DB 连接池。
- 下游服务。
- 消息队列 ack 能力。

应使用 worker pool、semaphore 或 errgroup + limit。

### 8.3 context 取消传播

长生命周期 goroutine 应监听：

```go
select {
case <-ctx.Done():
    return ctx.Err()
case item := <-ch:
    // work
}
```

### 8.4 observability

生产指标建议：

- goroutine count。
- worker active count。
- queue depth。
- task duration。
- task error count。
- blocked send/receive 或 timeout count。

---

## 代码实验

配套实验目录：

```text
labs/06-goroutine-scheduler/
```

实验覆盖：

1. 有界 worker pool。
2. context 取消后 worker 退出。
3. 无界 goroutine 与有界 worker 的对比。
4. `GOMAXPROCS` 对 CPU 任务吞吐的影响。
5. 使用 `GODEBUG=schedtrace` 观察调度器。
6. goroutine leak demo 与修复。

运行：

```bash
cd labs/06-goroutine-scheduler
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
GODEBUG=schedtrace=1000,scheddetail=1 go test -run TestWorkerPoolProcessesJobs ./...
```

---

## 常见误区

### 误区 1：goroutine 很轻，所以不用限制数量

goroutine 轻量但不是免费。生产中必须限制并发和下游压力。

### 误区 2：GOMAXPROCS 控制 goroutine 数量

`GOMAXPROCS` 控制 P 的数量，也就是并行执行 Go 代码的调度资源数量。

### 误区 3：channel 阻塞就一定没问题

阻塞可能是正确背压，也可能是 goroutine 泄漏。

### 误区 4：只要函数返回，里面启动的 goroutine 也会结束

goroutine 独立执行。函数返回不会自动取消子 goroutine。

### 误区 5：只看 CPU，不看 goroutine count

goroutine 泄漏早期可能 CPU 不高，但内存、引用对象和下游连接会逐渐异常。

---

## 故障案例

### 案例 1：请求结束后 goroutine 仍在运行

原因：handler 内启动 goroutine，但没有传递 request context，也没有退出条件。

修复：传入 context，或把后台任务交给有生命周期管理的 worker 系统。

### 案例 2：批量任务无界 goroutine 打爆 DB

原因：每个 item 启动一个 goroutine，每个 goroutine 查询数据库，超过连接池和数据库承载能力。

修复：worker pool 或 semaphore 限制并发，并对每个操作设置 timeout。

### 案例 3：发送到无人接收的 channel 导致泄漏

原因：goroutine 阻塞在 `ch <- value`，接收方提前退出。

修复：发送时监听 context 或使用带缓冲/非阻塞策略，但必须明确丢弃/重试语义。

---

## 作业

1. 在 Lab 6 中把 worker pool 的 worker 数从 1、2、4、8 调整，观察 benchmark 变化。
2. 使用 `GODEBUG=schedtrace=1000,scheddetail=1` 运行测试，截取并解释 `gomaxprocs`、`threads`、`runqueue`。
3. 写一个会泄漏 goroutine 的例子，再用 context 修复。
4. 用 `runtime.NumGoroutine()` 写测试，验证取消后 goroutine 数量不会持续增长。
5. 设计一个生产 worker 指标列表。

---

## 评估标准

- 能解释 G/M/P 的职责。
- 能解释 `GOMAXPROCS` 的作用。
- 能说明 netpoller 为什么有助于高并发网络服务。
- 能识别无界 goroutine 和 goroutine leak。
- 能写出带 context 取消和并发上限的 worker pool。
- 能使用 schedtrace 观察调度状态。

---

## 本节不展开

- runtime scheduler 源码逐行阅读。
- cgo 与 thread pinning 的深层细节。
- `runtime.LockOSThread` 的高级场景。
- Go trace 可视化的完整使用。

---

## 延伸阅读

- Go Blog: Go Concurrency Patterns
- Go Blog: Pipelines and cancellation
- Go runtime scheduler design documents
- Go Execution Tracer
- 100 Go Mistakes and How to Avoid Them
