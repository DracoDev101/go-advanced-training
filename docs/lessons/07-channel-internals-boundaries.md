# Lesson 7：Channel 原理与使用边界

## 学习目标

完成本课后，学习者应该能够：

- 解释 channel 的核心语义：发送、接收、阻塞、关闭、range。
- 区分 unbuffered channel 与 buffered channel 的同步语义。
- 理解 channel 底层包含缓冲区、send queue、receive queue 和锁。
- 判断 channel 适合传递 ownership / event，mutex 适合保护共享状态。
- 正确使用 `select` 实现 timeout、取消、非阻塞尝试和多路复用。
- 解释 `close` 的语义：谁关闭、何时关闭、关闭后接收什么、为什么不能重复关闭。
- 识别 channel 导致 goroutine leak 的常见模式。
- 在生产系统中用 channel 表达背压，而不是用 buffered channel 掩盖下游问题。

---

## 关键问题

1. unbuffered channel 和 buffered channel 的区别是什么？
2. channel 是不是比 mutex 更“Go 风格”？
3. 什么情况下应该用 channel，什么情况下应该用 mutex？
4. 谁应该负责 close channel？
5. 关闭 channel 后还能接收吗？还能发送吗？
6. buffered channel 是否能解决慢消费者问题？
7. `select` 中如何处理 context cancellation？
8. channel 阻塞什么时候是健康背压，什么时候是 goroutine leak？

---

## 核心结论

- channel 是 goroutine 之间通信和同步的工具，不是所有共享状态问题的默认解法。
- unbuffered channel 同时传递值和同步发送/接收双方。
- buffered channel 在缓冲未满/未空时会弱化同步关系，但容量只是临时缓冲，不是无限队列。
- 通常由发送方关闭 channel；接收方不应关闭自己不拥有的 channel。
- 关闭 channel 是广播“不会再有新值”的信号，不是取消机制的通用替代品。
- `select` + `ctx.Done()` 是避免阻塞泄漏的重要模式。
- channel 适合传递 ownership、事件、任务队列、结果流；mutex 适合保护内存中的共享状态。
- 生产代码中 channel 必须有容量设计、退出协议和观测指标。

---

## 设计哲学

Go 有一句经典表达：

```text
Do not communicate by sharing memory; instead, share memory by communicating.
```

这句话强调用通信表达所有权转移，降低共享可变状态的复杂度。但它不是说“不要使用 mutex”。

更实用的判断是：

```text
channel 用于传递 ownership 或事件；mutex 用于保护共享状态。
```

例如：

- worker pool 的任务分发：channel 很自然。
- HTTP server 的 in-memory cache：map + RWMutex 通常更自然。
- 一个 goroutine 专属维护状态，其他 goroutine 通过消息请求它：channel 可以。
- 多个 goroutine 高频读取少量状态：mutex/atomic 更简单。

Go 的设计不是鼓励“为了 Go 风格强行 channel”，而是给你多种同步原语，让代码语义清晰。

---

## 1. Channel 基础语义

创建 channel：

```go
ch := make(chan int)
```

发送：

```go
ch <- 1
```

接收：

```go
v := <-ch
```

关闭：

```go
close(ch)
```

range 接收直到 channel 关闭：

```go
for v := range ch {
    fmt.Println(v)
}
```

---

## 2. Unbuffered Channel

```go
ch := make(chan int)
```

unbuffered channel 没有容量。发送方和接收方必须同时 rendezvous。

这意味着：

- 发送会阻塞，直到有人接收。
- 接收会阻塞，直到有人发送。
- 成功发送/接收建立 happens-before 关系。

适合：

- 强同步交接。
- 明确 ownership transfer。
- 测试中等待 goroutine 到达某个点。

不适合：

- 高吞吐异步缓冲。
- 下游可能长期慢的队列。

---

## 3. Buffered Channel

```go
ch := make(chan int, 10)
```

buffered channel 有固定容量。

- 缓冲未满时，发送不阻塞。
- 缓冲非空时，接收不阻塞。
- 缓冲满时，发送阻塞。
- 缓冲空时，接收阻塞。

buffer 不是越大越好。它只是吸收短期抖动。

如果消费者长期慢，buffer 最终会满，然后生产者仍会阻塞。盲目加大 buffer 会：

- 增加内存占用。
- 推迟暴露背压。
- 增加排队延迟。
- 让故障恢复更慢。

---

## 4. close 语义

### 4.1 关闭后接收

```go
close(ch)
v, ok := <-ch
```

当 channel 已关闭且缓冲已空：

```text
v = 零值
ok = false
```

### 4.2 关闭后发送

向已关闭 channel 发送会 panic：

```text
panic: send on closed channel
```

### 4.3 重复关闭

重复 close 会 panic：

```text
panic: close of closed channel
```

### 4.4 谁负责 close？

经验规则：

```text
由发送方关闭 channel，表示不会再发送新值。
```

接收方通常不应该关闭 channel，因为它不知道其他发送方是否还会发送。

如果有多个发送方，通常需要一个协调者在所有发送方退出后关闭 channel，例如 `sync.WaitGroup`。

---

## 5. select

`select` 让 goroutine 等待多个 channel 操作。

### 5.1 cancellation

```go
select {
case v := <-ch:
    handle(v)
case <-ctx.Done():
    return ctx.Err()
}
```

### 5.2 timeout

```go
select {
case result := <-resultCh:
    return result, nil
case <-time.After(time.Second):
    return Result{}, context.DeadlineExceeded
}
```

在循环中频繁使用 `time.After` 要谨慎，可能带来额外 timer 分配。更可控的方式是复用 timer 或使用 context timeout。

### 5.3 non-blocking try send/receive

```go
select {
case ch <- v:
    return true
default:
    return false
}
```

这常用于“满了就丢弃”的指标、日志、通知场景。但必须明确丢弃语义。

### 5.4 nil channel trick

nil channel 的发送和接收会永久阻塞。可以在 select 中把某个 channel 设为 nil 来动态禁用 case。

```go
var out chan<- T
if ready {
    out = realOut
}
```

这是高级技巧，能简化状态机，但不应滥用。

---

## 6. Channel vs Mutex

### 适合 channel

- 任务队列。
- pipeline stage。
- fan-in / fan-out。
- 事件通知。
- ownership transfer。
- 一个 goroutine 独占状态，其他 goroutine 通过消息交互。

### 适合 mutex

- 保护 map/cache/counter 等共享内存状态。
- 临界区短、访问频繁。
- 数据结构语义本身就是共享对象。
- 需要读写锁优化读多写少。

反例：用 channel 实现普通计数器通常比 mutex 更复杂、更慢。

---

## 7. Backpressure 与泄漏

channel 阻塞有两种含义：

### 健康背压

消费者处理慢，生产者阻塞，从而限制系统继续堆积。

这是好事，但必须可观测：

- queue length。
- send wait time。
- processing duration。
- dropped count。

### 泄漏或死锁

- 发送方阻塞，但接收方已经退出。
- 接收方等待，但发送方永远不会发送/关闭。
- range channel，但无人 close。
- 错误路径提前 return，未 drain channel。

解决模式：

```go
select {
case ch <- v:
case <-ctx.Done():
    return ctx.Err()
}
```

---

## 8. 生产实践

### Channel 设计 checklist

```text
channel 传递什么？值、指针、事件、ownership？
谁发送？谁接收？是否多发送方/多接收方？
谁关闭？什么时候关闭？
容量是多少？为什么？
满了怎么办：阻塞、丢弃、超时、扩容、返回错误？
取消路径是什么？
错误如何传播？
是否需要 drain？
有哪些指标？
```

### 常见模式

- producer closes outbound channel。
- consumer ranges until channel closed。
- ctx cancels all goroutines。
- errgroup 汇总错误并取消。
- semaphore 限制并发。
- drop channel 用于非关键事件。

---

## 代码实验

配套实验目录：

```text
labs/07-channel-boundaries/
```

实验覆盖：

1. unbuffered channel 的同步交接。
2. buffered channel 容量满时阻塞。
3. close 后接收零值和 ok=false。
4. send on closed channel 会 panic。
5. select + context 避免发送阻塞泄漏。
6. try send 实现满时丢弃。
7. channel counter vs mutex counter benchmark。
8. fan-in 汇总多个输入。

运行：

```bash
cd labs/07-channel-boundaries
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

---

## 常见误区

### 误区 1：channel 比 mutex 更高级

channel 和 mutex 解决不同问题。保护共享 map 通常用 mutex 更清晰。

### 误区 2：接收方可以随手 close channel

close 表示“不会再有新值”。通常只有发送方知道这一点。

### 误区 3：buffered channel 可以解决消费者慢的问题

buffer 只能吸收短期抖动，不能解决长期处理能力不足。

### 误区 4：range channel 会自动结束

range 只有在 channel 被关闭且缓冲 drain 后才结束。

### 误区 5：select default 总是好事

default 会让操作非阻塞，也可能造成 busy loop 或静默丢数据。

---

## 故障案例

### 案例 1：日志 channel 满后业务请求卡住

原因：业务路径同步发送日志到有限 buffer channel，日志消费者异常停止，发送方全部阻塞。

修复：非关键日志使用 try send + drop counter，或保证消费者生命周期和降级策略。

### 案例 2：多个 producer 中某个 producer close channel 导致 panic

原因：其他 producer 仍在发送。

修复：producer 不直接 close 共享 out channel，由协调 goroutine 等所有 producer 完成后 close。

### 案例 3：pipeline 下游提前退出导致上游泄漏

原因：上游阻塞在 send，下游因错误 return。

修复：所有 send/select 都监听 ctx.Done，错误时取消 context。

---

## 作业

1. 在 Lab 7 中新增一个 pipeline：生成数字 → 平方 → 汇总，并支持 context 取消。
2. 写一个测试证明下游提前取消后，上游 goroutine 能退出。
3. 把 channel counter 改成 mutex counter，对比 benchmark。
4. 为 try send drop 模式增加 dropped counter。
5. 设计一个生产队列的指标列表：queue length、enqueue wait、dropped、processed、failed。

---

## 评估标准

- 能解释 unbuffered/buffered channel 的阻塞语义。
- 能说明 close 的正确使用边界。
- 能用 select + context 写出不泄漏的 send/receive。
- 能判断 channel 与 mutex 的适用场景。
- 能解释 buffered channel 与 backpressure 的关系。
- 能通过测试和 benchmark 验证 channel 设计。

---

## 本节不展开

- `runtime.hchan` 源码逐行阅读。
- select pseudo-random fairness 的源码细节。
- lock-free queue 与 channel 的性能对比。
- 复杂 actor model 框架设计。

---

## 延伸阅读

- Go Blog: Share Memory By Communicating
- Go Blog: Pipelines and cancellation
- Go Spec: Channel types
- Effective Go: Channels
- 100 Go Mistakes and How to Avoid Them
