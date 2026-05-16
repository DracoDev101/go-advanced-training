# Lab 5：Error Handling 生产实践

本实验配套 Lesson 5，观察 `%w` wrapping、`errors.Is`、`errors.As`、domain error 映射、统一 API 错误响应、`errors.Join` 和 recover 边界。

## 运行命令

```bash
cd labs/05-error-handling

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## 观察目标

1. `%w` 保留错误链，`%v` 破坏错误链。
2. repository 层错误可以转换为 domain error。
3. domain error 映射到 HTTP status。
4. custom error type 可以通过 `errors.As` 提取结构化字段。
5. 对外错误响应不泄露内部错误细节。
6. `errors.Join` 支持多个错误并可被 `errors.As` 遍历。
7. recover 边界捕获 panic，返回安全 500。

## 生产启发

- 不要用字符串匹配错误。
- 包装底层错误时使用 `%w`。
- 对外响应使用稳定 code，内部日志保留完整错误链。
- panic/recover 不是业务错误处理机制。
