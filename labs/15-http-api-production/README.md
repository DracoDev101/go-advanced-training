# Lab 15：HTTP Production Debugging

本实验实现一个最小生产级 HTTP API，重点不是路由库，而是 timeout、body limit、错误映射、request id 和 graceful shutdown。

## 运行

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 观察点

- `NewServer` 必须显式配置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout`。
- `BodyLimit` 防止无上限读取 request body。
- `Timeout` 将 request context 传入 service。
- domain error 映射成稳定 HTTP error schema。
- `X-Request-Id` 贯穿 response 和 error body。

## 排查练习

慢请求：先看 route/status 维度 latency，再查 trace 或 CPU/block profile。

大 body：确认是否使用 `http.MaxBytesReader`。

client cancel：确认 handler/service 是否监听 `r.Context().Done()`。

## 生产启发

Handler 边界应保持：decode → validate → application service → map error → encode。不要在 handler 中直接写 SQL 或做长任务。
