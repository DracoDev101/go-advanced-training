# Lesson 8：Context 与取消传播

## 学习目标

完成本课后，学习者应该能够：

- 解释 `context.Context` 的三个核心职责：取消、截止时间、请求作用域值。
- 区分 cancellation、timeout、deadline 的语义。
- 正确使用 `context.WithCancel`、`WithTimeout`、`WithDeadline` 和 `WithValue`。
- 理解 context cancellation tree：父 context 取消会传播到子 context。
- 在 HTTP、gRPC、DB、worker、pipeline 中正确传递 context。
- 识别 context misuse：存入 struct、传 nil、滥用 Value、忘记 cancel、后台任务误用 request context。
- 用 `select { case <-ctx.Done(): ... }` 让 goroutine 可退出。
- 设计 graceful shutdown：停止接收新请求、取消后台任务、等待 in-flight work、设置 shutdown timeout。

---

## 关键问题

1. context 是干什么的？它不是什么？
2. 为什么函数参数里通常把 `ctx context.Context` 放第一个？
3. `WithTimeout` 返回的 cancel 为什么即使超时也要调用？
4. request context 能不能传给后台异步任务？
5. `context.Value` 能不能当成通用参数传递机制？
6. 父 context 取消后，子 context 会怎样？子 context 取消会影响父 context 吗？
7. DB query、HTTP client、worker loop 如何响应 context？
8. graceful shutdown 应该如何组织 context？

---

## 核心结论

- context 是跨 API 边界传递取消信号、deadline 和请求作用域元数据的标准机制。
- context 应作为函数第一个参数显式传递，不应存入 struct 作为长期状态。
- `WithCancel/WithTimeout/WithDeadline` 返回的 cancel 必须调用，释放 timer 和子节点资源。
- 父 context 取消会取消所有子 context；子 context 取消不会取消父 context。
- `context.Value` 只适合 request-scoped metadata，例如 request id、trace id、auth principal；不应用于业务参数。
- 长时间运行的 goroutine、channel send/receive、IO 操作都应该监听 `ctx.Done()`。
- request context 生命周期与请求绑定；后台任务通常应复制必要数据并使用独立 context。
- graceful shutdown 不是 `os.Exit`，而是有阶段、有超时、有等待、有观测的停止流程。

---

## 设计哲学

Go 没有给 goroutine 提供强制 kill API。这是有意为之：强制终止会破坏资源清理、锁状态和一致性。

Go 的取消模型是协作式的：

```text
调用方发出取消信号，被调用方定期检查并自行退出。
```

context 正是这种协作式取消的标准载体。

这也意味着：

```text
context 只能发出取消信号，不能保证代码立刻停止。
```

如果一个函数从不检查 `ctx.Done()`，也不把 context 传给可取消的 IO，它就不会及时响应取消。

---

## 1. Context 的三个职责

### 1.1 Cancellation

```go
ctx, cancel := context.WithCancel(parent)
defer cancel()
```

调用 cancel 后：

```go
<-ctx.Done()
ctx.Err() == context.Canceled
```

### 1.2 Deadline / Timeout

```go
ctx, cancel := context.WithTimeout(parent, time.Second)
defer cancel()
```

超时后：

```go
ctx.Err() == context.DeadlineExceeded
```

`WithDeadline` 使用绝对时间，`WithTimeout` 使用相对时间。

### 1.3 Request-scoped values

```go
type requestIDKey struct{}
ctx = context.WithValue(ctx, requestIDKey{}, "req-123")
```

只适合跨 API 边界传递请求作用域元数据。

不适合：

```go
ctx = context.WithValue(ctx, "userID", userID) // string key 易冲突
ctx = context.WithValue(ctx, "limit", 100)     // 业务参数不应藏在 ctx
```

---

## 2. Cancellation Tree

context 是一棵树。

```text
root
 ├── request ctx
 │    ├── db query ctx
 │    └── http client ctx
 └── background worker ctx
```

父 context 取消：所有子 context 都会取消。

子 context 取消：不会影响父 context，也不会影响兄弟 context。

这让我们可以表达：

- 请求结束，取消该请求下的 DB/HTTP 调用。
- 服务 shutdown，取消所有后台 worker。
- 单个子操作 timeout，不影响整个请求其他部分。

---

## 3. 为什么必须调用 cancel

即使 timeout 会自动触发，也应该调用 cancel：

```go
ctx, cancel := context.WithTimeout(parent, time.Second)
defer cancel()
```

原因：

- 释放 timer。
- 从父 context 的子节点集合中移除。
- 更早释放相关资源。

在循环中尤其重要：

```go
for _, item := range items {
    ctx, cancel := context.WithTimeout(parent, time.Second)
    err := process(ctx, item)
    cancel()
    if err != nil { ... }
}
```

不要在循环里简单 `defer cancel()`，否则 cancel 会推迟到函数结束才执行。

---

## 4. Context 在 API 中的位置

约定：

```go
func Do(ctx context.Context, id string) error
```

原因：

- 明确这个操作可取消。
- 便于静态阅读。
- 与标准库、生态保持一致。

不要：

```go
type Service struct {
    ctx context.Context
}
```

context 是调用作用域，不是对象生命周期配置。长期对象应该接收 context 作为方法参数。

---

## 5. Context 与 IO

### 5.1 HTTP server

handler 中的 request context 会在以下情况取消：

- 客户端断开。
- 请求被 server 取消。
- handler 返回后生命周期结束。

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    _ = service.Do(ctx)
}
```

### 5.2 HTTP client

```go
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := http.DefaultClient.Do(req)
```

### 5.3 database/sql

```go
row := db.QueryRowContext(ctx, query, args...)
```

如果 context 超时或取消，driver 会尽力取消底层操作。

---

## 6. Context 与 goroutine

长生命周期 goroutine 应该监听 context：

```go
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case job := <-jobs:
        handle(job)
    }
}
```

channel send 也应可取消：

```go
select {
case out <- v:
case <-ctx.Done():
    return ctx.Err()
}
```

否则下游退出时，上游可能永久阻塞。

---

## 7. 后台任务与 request context

反例：

```go
func handler(w http.ResponseWriter, r *http.Request) {
    go sendEmail(r.Context(), userID)
    w.WriteHeader(http.StatusAccepted)
}
```

handler 返回后，request context 可能取消，导致后台邮件任务立即失败。

更好的方式：

- 将任务写入队列。
- 后台 worker 使用服务级 context。
- 从 request context 中复制必要 metadata，例如 request id、user id。
- 为后台任务设置自己的 timeout。

---

## 8. Graceful Shutdown

典型流程：

```text
收到 SIGTERM
  ↓
停止接收新请求
  ↓
给 server.Shutdown 一个 timeout context
  ↓
取消后台 worker root context
  ↓
等待 worker / queue drain
  ↓
flush logs / metrics
  ↓
退出进程
```

HTTP server：

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```

注意：

- shutdown context 是“最多等待多久关闭”的 context。
- worker root context 是“通知 worker 退出”的 context。
- 两者相关但语义不同。

---

## 9. Context misuse

### 9.1 传 nil context

不要传 nil。使用：

```go
context.Background()
context.TODO()
```

### 9.2 滥用 context.Value

不要把业务参数塞进 context。

错误：

```go
ctx = context.WithValue(ctx, "page_size", 100)
```

正确：

```go
ListUsers(ctx, ListUsersParams{PageSize: 100})
```

### 9.3 忘记 cancel

尤其在循环中会导致 timer 和子 context 资源滞留。

### 9.4 context 存 struct

长期服务对象不应持有请求 context。

### 9.5 忽略 ctx.Err

取消后应返回 `ctx.Err()`，让调用方知道是 canceled 还是 deadline exceeded。

---

## 代码实验

配套实验目录：

```text
labs/08-context-cancellation/
```

实验覆盖：

1. parent cancel 传播到 child。
2. child cancel 不影响 parent。
3. timeout 后返回 `context.DeadlineExceeded`。
4. worker loop 响应 context cancellation。
5. send with context 避免 channel 阻塞泄漏。
6. typed key 安全使用 context value。
7. request context 与 background context 的区别。
8. graceful shutdown demo。

运行：

```bash
cd labs/08-context-cancellation
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

---

## 常见误区

### 误区 1：context 会自动杀死 goroutine

不会。它只是关闭 Done channel。goroutine 必须主动检查。

### 误区 2：WithTimeout 超时后不用 cancel

仍应调用 cancel 释放资源。

### 误区 3：context.Value 可以当参数包用

这会隐藏依赖、破坏类型安全、降低可测试性。

### 误区 4：后台任务直接使用 request context

请求结束后 context 取消，后台任务可能意外终止。

### 误区 5：所有函数都必须接收 context

只有可能阻塞、IO、耗时、跨边界、需要取消/追踪的函数才需要 context。纯计算小函数不必强行加。

---

## 故障案例

### 案例 1：客户端断开后 DB 查询继续运行

原因：repository 使用 `db.Query` 而不是 `db.QueryContext`。

修复：全链路传递 request context。

### 案例 2：请求返回 202 后后台任务马上失败

原因：后台任务使用 request context，handler 返回后 context 被取消。

修复：任务入队；worker 使用服务级 context，并设置任务 timeout。

### 案例 3：循环中 defer cancel 导致 timer 堆积

原因：循环创建 `WithTimeout`，但 `defer cancel()` 推迟到函数结束。

修复：每次迭代结束立即 `cancel()`。

---

## 作业

1. 在 Lab 8 中新增一个 HTTP client demo，用 `httptest.Server` 模拟慢响应，验证 timeout。
2. 写一个 worker，收到 context cancel 后停止接收新 job，但完成当前 job。
3. 用 typed key 在 context 中传 request id，并在日志函数中读取。
4. 写一个错误示例：把业务参数存在 context.Value，再改成显式参数结构体。
5. 设计 Production Job Runner 的 shutdown 流程图。

---

## 评估标准

- 能解释 context 的三个职责。
- 能正确使用 WithCancel/WithTimeout/WithDeadline。
- 能说明 cancel 为什么必须调用。
- 能设计 cancellation tree。
- 能识别 request context 与 background worker context 的边界。
- 能写出响应 context 的 worker/channel/IO 代码。
- 能设计 graceful shutdown 流程。

---

## 本节不展开

- OpenTelemetry baggage 与 context propagation 的完整规范。
- gRPC metadata 与 context 的高级用法。
- Kubernetes termination lifecycle 的完整细节。
- errgroup 的深入实现。

---

## 延伸阅读

- Go Blog: Context
- Go Blog: Pipelines and cancellation
- `context` package documentation
- `net/http` Server.Shutdown documentation
- 100 Go Mistakes and How to Avoid Them
