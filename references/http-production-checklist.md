# HTTP 生产实践与排查 Checklist

## 1. Server 必配项

```go
srv := &http.Server{
    Addr:              ":8080",
    Handler:           router,
    ReadHeaderTimeout: 2 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

不要使用零值 `http.Server` 直接上线。尤其是公网服务，`ReadHeaderTimeout` 能缓解 slowloris。

## 2. Middleware 顺序

推荐顺序：

```text
request_id
→ recover
→ body_limit
→ timeout
→ auth
→ logging
→ metrics
→ tracing
→ handler
```

原则：

- recover 要靠前，保证 panic 被记录。
- body limit 要在 decode 前。
- timeout 要包住业务 handler。
- logging/metrics 要能看到最终 status 和 duration。

## 3. Handler 边界

Handler 只做：

```text
decode → validate → call application service → map error → encode response
```

不要在 handler 里直接写 SQL、发 MQ、做长任务。

## 4. Error Response Schema

```json
{
  "error": {
    "code": "invalid_argument",
    "message": "invalid job kind",
    "request_id": "req_123"
  }
}
```

Domain error 映射：

| Domain error | HTTP |
|---|---|
| validation | 400 |
| unauthenticated | 401 |
| permission denied | 403 |
| not found | 404 |
| conflict / invalid state | 409 |
| rate limited | 429 |
| dependency timeout | 504 |
| internal | 500 |

## 5. Client Cancel

handler 必须尊重：

```go
select {
case <-r.Context().Done():
    return r.Context().Err()
case result := <-done:
    return nil
}
```

如果 client 断开后服务仍继续执行昂贵工作，会造成资源泄漏。

## 6. Graceful Shutdown

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    _ = srv.Close()
}
```

关闭顺序：停止接新请求 → 等待 in-flight → 停 worker → flush logs/metrics。

## 7. 排查路径

### 慢请求

```text
看 http_request_duration_seconds{route,status}
→ trace 找慢 span
→ CPU profile / block profile
→ DB pool/downstream latency
```

### 连接耗尽

检查：

- client 是否设置 timeout。
- `resp.Body.Close()` 是否总是调用。
- `Transport.MaxIdleConnsPerHost` 是否合理。

### 大 body / OOM

使用：

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
```

不要无上限 `io.ReadAll(r.Body)`。
