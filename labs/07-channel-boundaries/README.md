# Lab 7：Channel 原理与使用边界

本实验配套 Lesson 7，观察 unbuffered/buffered channel、close 语义、select + context、try send、fan-in、pipeline，以及 channel counter vs mutex counter。

## 运行命令

```bash
cd labs/07-channel-boundaries

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## 观察目标

1. unbuffered channel 发送会阻塞直到接收发生。
2. buffered channel 满时发送会阻塞，try send 可以选择丢弃。
3. close 后仍可接收缓冲值；缓冲 drain 后接收零值且 `ok=false`。
4. send on closed channel 会 panic。
5. `select + ctx.Done()` 可以避免发送方永久阻塞。
6. fan-in 需要在所有输入结束后关闭输出。
7. pipeline 每个 stage 都应该监听 context。
8. channel counter 通常比 mutex counter 更复杂、更慢，不要为了“Go 风格”滥用 channel。

## 生产启发

- channel 用于传递 ownership/event，mutex 用于保护共享状态。
- close 通常由发送方负责。
- buffer 只吸收短期抖动，不解决长期下游慢。
- 任何可能阻塞的 send/receive 都要考虑取消路径。
