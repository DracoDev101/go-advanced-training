# Lab 13：Go Build、Link 与部署形态

本实验配套 Lesson 13，用统一的最小事件处理模型练习：context 取消、错误处理、状态记录、并发安全、benchmark 和生产 checklist。

## 关键词

build, link, ldflags, container, version, release

## 运行命令

```bash
cd labs/13-build-link-deploy

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 观察目标

1. 正常路径会记录事件并更新计数。
2. 无效输入返回稳定错误。
3. 已取消 context 会中断处理。
4. benchmark 提供后续优化基线。
5. 通过最小模型讨论本节主题在 Production Job Runner 中的落地。

## 生产启发

- 每个生产组件都要有 context、错误语义、观测字段和测试。
- 不要只实现 happy path，失败路径必须可验证。
- benchmark 不是最终答案，但能防止凭感觉优化。
