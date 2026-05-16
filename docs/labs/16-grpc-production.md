# Lab 16：生产级 gRPC

本实验配套 Lesson 16，用统一的最小事件处理模型练习 context 取消、错误处理、状态记录、并发安全、benchmark 和生产 checklist。

## 关键词

grpc, protobuf, interceptor, deadline, status, metadata

## 运行命令

```bash
cd labs/16-grpc-production

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 观察目标

1. 正常路径记录事件并更新计数。
2. 无效输入返回稳定错误。
3. 已取消 context 中断处理。
4. benchmark 提供优化基线。
5. 将本节主题映射到 Production Job Runner。

## 生产启发

- 每个生产组件都要有 context、错误语义、观测字段和测试。
- 不只实现 happy path，失败路径必须可验证。
- benchmark 能防止凭感觉优化。
