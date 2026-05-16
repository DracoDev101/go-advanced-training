# Lesson 5：Error Handling 生产实践

## 学习目标

完成本课后，学习者应该能够：

- 解释 Go 为什么选择 `error as value`，而不是异常机制。
- 区分 sentinel error、custom error type、wrapped error、joined error 的适用场景。
- 正确使用 `fmt.Errorf("...: %w", err)` 保留错误链。
- 使用 `errors.Is` 判断错误类别，使用 `errors.As` 提取错误类型。
- 设计 repository → service/domain → transport 的错误分层与映射。
- 判断何时应该 panic，何时应该返回 error。
- 在生产系统中让错误同时具备：可判断、可观测、可排障、不会泄露敏感信息。

---

## 关键问题

1. Go 为什么不用 exception？
2. `error` 只是一个 interface，它为什么足够表达复杂失败？
3. 什么时候应该使用 sentinel error？
4. 什么时候应该使用 custom error type？
5. 为什么 `fmt.Errorf("failed: %v", err)` 会破坏错误链？
6. repository 层的 `sql.ErrNoRows` 应不应该直接暴露给 HTTP handler？
7. panic/recover 应该放在哪里？
8. 如何设计统一错误响应而不丢失内部排障信息？

---

## 核心结论

- Go 把错误作为普通值，是为了让失败路径显式出现在代码中。
- error handling 的目标不是“少写 if err”，而是让失败路径可控、可判断、可观测。
- 包装错误时应使用 `%w`，否则 `errors.Is/As` 无法穿透错误链。
- sentinel error 适合稳定、少量、可比较的错误类别。
- custom error type 适合携带结构化上下文，例如字段名、状态码、远端响应码。
- transport 层不应该直接暴露底层错误；应将 domain error 映射到 HTTP/gRPC 状态。
- panic 只应用于不可恢复的程序员错误或启动期配置错误，不应用于普通业务失败。
- 日志里记录内部细节，对外响应只暴露安全、稳定、可理解的信息。

---

## 设计哲学

Go 的错误处理常被批评“啰嗦”：

```go
if err != nil {
    return err
}
```

但这正是 Go 的设计取舍：

```text
失败路径是系统复杂性的一部分，不能被异常机制隐藏在控制流之外。
```

异常让 happy path 更短，但也让调用边界的失败行为更隐式。Go 选择让错误成为普通值，迫使 API 明确告诉调用方：这个操作可能失败，你必须决定如何处理。

在生产系统中，错误处理不是语法问题，而是系统设计问题：

```text
底层错误如何转换成业务语义？
哪些错误可以重试？
哪些错误应该返回 404/409/429/500？
哪些错误要报警？
哪些错误不能暴露给用户？
```

---

## 1. error 是什么

Go 的 `error` 是一个普通 interface：

```go
type error interface {
    Error() string
}
```

任何实现了 `Error() string` 的类型都可以作为 error。

最简单的错误：

```go
return errors.New("not found")
```

带格式的错误：

```go
return fmt.Errorf("open config %s: %w", path, err)
```

注意 `%w`：它表示包装一个底层错误，并让错误链可以被 `errors.Is/As` 遍历。

---

## 2. Sentinel error

Sentinel error 是包级变量：

```go
var ErrUserNotFound = errors.New("user not found")
```

调用方可以判断：

```go
if errors.Is(err, ErrUserNotFound) {
    // handle not found
}
```

适合 sentinel error 的场景：

- 错误类别稳定。
- 不需要携带太多结构化字段。
- 调用方需要明确分支处理。
- 包边界内外都能接受这个错误语义。

不适合：

- 错误需要携带字段名、资源 ID、远端状态码。
- 错误类别可能频繁变化。
- 想把内部实现细节暴露给上层。

生产建议：sentinel error 的数量要少，语义要稳定。

---

## 3. Custom error type

当错误需要结构化上下文时，使用自定义类型：

```go
type ValidationError struct {
    Field string
    Rule  string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("invalid %s: %s", e.Field, e.Rule)
}
```

调用方使用 `errors.As`：

```go
var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Println(ve.Field)
}
```

适合：

- validation error。
- remote API error。
- rate limit error。
- conflict/version error。
- 需要携带 retry-after、field、code、operation 的错误。

注意：custom error type 的字段也会成为 API 的一部分。如果暴露在公共 package，要谨慎设计。

---

## 4. Wrapping：保留上下文

错误从底层向上传播时，应该不断增加上下文，而不是丢失原始原因。

错误链示例：

```text
create user: get user by email: sql: no rows in result set
```

代码：

```go
if err != nil {
    return fmt.Errorf("get user by email: %w", err)
}
```

反例：

```go
return fmt.Errorf("get user by email: %v", err)
```

`%v` 只拼接字符串，会破坏错误链。之后：

```go
errors.Is(err, sql.ErrNoRows)
```

将无法工作。

---

## 5. errors.Is 与 errors.As

### errors.Is

用于判断错误链中是否包含某个目标错误：

```go
if errors.Is(err, ErrUserNotFound) {
    return http.StatusNotFound
}
```

适合判断错误类别。

### errors.As

用于从错误链中提取某个具体错误类型：

```go
var ve *ValidationError
if errors.As(err, &ve) {
    return http.StatusBadRequest
}
```

适合读取结构化上下文。

### errors.Join

Go 1.20+ 支持把多个错误组合起来：

```go
return errors.Join(err1, err2)
```

`errors.Is/As` 可以遍历 joined errors。

适合：

- 批量任务中多个子任务失败。
- shutdown 时多个资源关闭失败。
- validation 中多个字段错误。

但不要滥用；调用方是否能理解多个错误也很重要。

---

## 6. 错误分层设计

生产服务里建议把错误分成三层：

```text
infrastructure/repository error
    ↓
domain/service error
    ↓
transport error
```

### 6.1 Repository 层

底层错误可能来自数据库：

```go
sql.ErrNoRows
context.DeadlineExceeded
pgconn.PgError
```

repository 可以把它转换成更稳定的 domain error：

```go
if errors.Is(err, sql.ErrNoRows) {
    return User{}, ErrUserNotFound
}
```

### 6.2 Service/domain 层

service 层表达业务语义：

```go
ErrUserNotFound
ErrEmailAlreadyExists
ErrInvalidUserState
```

### 6.3 Transport 层

HTTP handler 或 gRPC server 负责映射：

```text
ErrUserNotFound       → 404 Not Found / gRPC NotFound
ValidationError       → 400 Bad Request / InvalidArgument
ErrEmailAlreadyExists → 409 Conflict / AlreadyExists
context deadline      → 504 Gateway Timeout / DeadlineExceeded
unknown error         → 500 Internal Server Error / Internal
```

对外响应不要直接暴露 SQL、连接串、内部拓扑、第三方 API 原始响应中的敏感字段。

---

## 7. panic/recover 边界

panic 不是 Go 的普通错误处理机制。

适合 panic 的场景：

- 程序员错误，例如 impossible branch。
- 启动期配置非法，服务无法继续运行。
- 模板/正则在 init 阶段必须编译成功。

不适合 panic 的场景：

- 用户输入错误。
- 数据库查不到。
- 外部服务超时。
- 业务状态冲突。

HTTP server 中可以用 recover middleware 防止进程崩溃，但 recover 后要：

- 记录 stack trace。
- 返回 500。
- 不把 panic 内容直接暴露给用户。
- 继续保证 request-scoped cleanup。

---

## 8. 生产实践：错误响应与日志

### 8.1 对外错误响应

建议统一结构：

```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "user not found",
    "request_id": "req_123"
  }
}
```

字段建议：

- `code`：稳定、机器可读。
- `message`：安全、人类可读。
- `request_id`：便于排障关联。
- 可选 `details`：validation fields 等安全细节。

### 8.2 内部日志

日志应包含：

- operation。
- request id / trace id。
- domain code。
- wrapped error 全链路。
- 必要上下文，例如 user id、job id。

避免：

- 密码、token、密钥。
- 完整连接串。
- 第三方敏感响应。
- 用户隐私字段。

---

## 代码实验

配套实验目录：

```text
labs/05-error-handling/
```

实验覆盖：

1. `%w` wrapping 保留错误链。
2. `%v` wrapping 破坏 `errors.Is`。
3. sentinel error 映射到 HTTP status。
4. custom validation error 使用 `errors.As` 提取字段。
5. repository error 转换为 domain error。
6. transport 层统一错误响应。
7. `errors.Join` 表达多个 validation error。
8. recover middleware 捕获 panic 并返回 500。

运行：

```bash
cd labs/05-error-handling
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

---

## 常见误区

### 误区 1：用字符串判断错误

```go
if err.Error() == "not found" {}
```

这很脆弱。应该使用 `errors.Is/As`。

### 误区 2：包装错误时用 `%v`

`%v` 会丢失错误链。需要保留链路时使用 `%w`。

### 误区 3：把所有错误都转成 500

这会让客户端无法正确处理 400/404/409/429 等业务语义。

### 误区 4：直接把内部错误返回给用户

可能泄露数据库结构、内部服务名、路径、敏感字段。

### 误区 5：用 panic 处理业务错误

panic 应该是异常边界，不是业务控制流。

---

## 故障案例

### 案例 1：404 被错误返回为 500

原因：repository 用 `%v` 包装 `sql.ErrNoRows`，handler 的 `errors.Is` 失效。

修复：改成 `%w`，或在 repository 层转换成 `ErrUserNotFound`。

### 案例 2：错误响应泄露内部 SQL

响应：

```text
pq: relation internal_user_shadow does not exist
```

修复：内部日志记录原始错误；对外返回稳定错误码和安全 message。

### 案例 3：批量任务失败只返回最后一个错误

多个子任务失败，但代码只保留最后一个 error，导致排障困难。

修复：使用 `errors.Join` 或结构化批量结果，保留多个失败原因。

---

## 作业

1. 在 Lab 5 中新增一个 `ErrPermissionDenied`，映射到 HTTP 403。
2. 写一个 custom `RateLimitError`，包含 `RetryAfter time.Duration`，并用 `errors.As` 提取。
3. 把一个 `%v` 包装错误的函数改为 `%w`，写测试证明 `errors.Is` 生效。
4. 设计一个统一错误响应 JSON，确保不会泄露内部错误。
5. 写一个 recover middleware 测试：handler panic 后返回 500，并记录 request id。

---

## 评估标准

- 能解释 error as value 的工程意义。
- 能正确使用 `%w`、`errors.Is`、`errors.As`。
- 能判断 sentinel error 与 custom error type 的适用场景。
- 能设计 repository/service/transport 的错误分层。
- 能把 domain error 映射到 HTTP/gRPC status。
- 能说明 panic/recover 的边界。
- 能设计安全、稳定、可观测的错误响应。

---

## 本节不展开

- 分布式 tracing 中 error status 的完整语义。
- gRPC status details 的高级用法。
- 多语言 API error schema 规范。
- Sentry/OTel exception event 的集成细节。

---

## 延伸阅读

- Go Blog: Error handling and Go
- Go Blog: Working with Errors in Go 1.13
- Go FAQ: Why does Go not have exceptions?
- Effective Go: Errors
- Go Code Review Comments: Error Strings
- 100 Go Mistakes and How to Avoid Them
