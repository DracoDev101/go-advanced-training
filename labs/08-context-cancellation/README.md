# Lab 8：Context 与取消传播

本实验配套 Lesson 8，观察 context cancellation tree、timeout、worker loop、channel send 取消、typed key、request/background context 边界和 graceful shutdown。

## 运行命令

```bash
cd labs/08-context-cancellation

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## 观察目标

1. parent cancel 会传播到 child。
2. child cancel 不影响 parent。
3. timeout 后返回 `context.DeadlineExceeded`。
4. worker loop 监听 `ctx.Done()` 后能退出。
5. channel send 使用 `select + ctx.Done()` 避免永久阻塞。
6. typed key 避免 context value key 冲突。
7. background task 不应直接继承 request context 的取消。
8. shutdown group 用 root context 通知 worker 退出，并用 shutdown context 限制等待时间。

## 生产启发

- context 是调用作用域，不要长期存进 struct。
- `WithTimeout/WithCancel` 后要调用 cancel。
- `context.Value` 只放 request-scoped metadata，不放业务参数。
- graceful shutdown 需要停止入口、取消后台、等待完成和超时保护。
