# Lesson 16：gRPC 生产实践

## 学习目标

完成本课后，学习者应该能够掌握：protobuf schema evolution；deadline、metadata、status code、interceptor；unary vs streaming；health/reflection、load balancing、retry policy。

---

## 关键问题

1. 本课主题解决哪个具体生产问题？
2. 它的正确边界是什么，哪些事情不该由它承担？
3. 默认行为中有哪些容易踩坑的地方？
4. 失败时如何观测、定位和恢复？
5. 在 Production Job Runner 中如何落地并验证？

---

## 核心结论

gRPC 的优势是强契约和连接复用；风险是 deadline 缺失、错误码滥用、schema 破坏兼容。

---

## 硬核要点

Status code 规则：`INVALID_ARGUMENT` 请求无效；`NOT_FOUND` 资源不存在；`FAILED_PRECONDITION` 状态不允许；`UNAVAILABLE` 暂时不可用；`DEADLINE_EXCEEDED` 超时。不要用 `UNKNOWN` 包所有错误。

---

## Production Job Runner 落地

本课内容必须映射到综合项目中的一个可验证设计点：明确 API / worker / repository / infrastructure 的责任边界；给出 timeout、retry、幂等、错误映射或一致性策略；定义至少 3 个观测信号；写一个失败路径测试或故障演练步骤。

---

## 代码实验

配套实验目录：`labs/16-grpc-production/`

```bash
cd labs/16-grpc-production
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

建议后续把当前最小 Lab 升级为本课专项 Lab，而不是只复用通用事件模型。

---

## 常见误区

1. 用框架或组件名替代设计边界。
2. 只写 happy path，不验证失败路径。
3. 只讨论“能不能跑”，不讨论容量、超时、回滚和观测。
4. 在没有指标和 profile 的情况下过早优化或过早抽象。

---

## 作业与评估

作业：写一页 Production Job Runner 落地设计；补充一个失败路径测试；定义 3 个指标和 2 个日志字段；说明一个不应该使用本技术/模式的场景。

评估：能说清楚机制和边界；能解释失败模式；能把设计落到代码、测试或 profile；能在综合项目中做出取舍并说明理由。
