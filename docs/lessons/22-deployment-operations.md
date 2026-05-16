# Lesson 22：部署与运行维护

## 学习目标

完成本课后，学习者应该能够：

- 设计配置加载与校验。
- 实现 liveness/readiness。
- 规划 migration 与 rollback。
- 理解运行时运维 checklist。
- 将本节主题落到 Production Job Runner 的生产设计中。

---

## 关键问题

1. 这个主题解决的生产问题是什么？
2. 相关 Go 标准库或主流生态能力的边界在哪里？
3. 失败模式、超时、取消和回滚如何设计？
4. 如何用测试、benchmark、profile 或故障演练验证？
5. 在 Production Job Runner 中应如何落地？

---

## 核心结论

- 本节关键词：**deployment、config、health、readiness、migration、rollback**。
- 生产级 Go 开发的重点不是堆技术，而是边界清晰、失败可控、可观测、可验证。
- 每个组件都要明确 owner、输入输出、错误语义、超时策略和观测指标。
- 优先用简单直接的实现；复杂模式必须由真实约束或指标驱动。
- 本节 Lab 用最小事件处理模型固化通用工程能力：context、错误、状态、测试、benchmark。

---

## 设计哲学

Go 的工程化实践强调组合胜过继承、显式胜过隐式、可读胜过炫技。对于 **部署与运行维护**，设计时应先问：

```text
边界在哪里？
谁拥有状态？
失败如何传播？
是否支持 context 取消？
是否有容量和超时预算？
如何观测和验证？
```

如果一个设计无法解释失败路径，它还不是生产级设计。

---

## 核心概念

### 1. 边界设计

将协议适配、业务编排、持久化、异步执行、可观测性拆开。每层只承担稳定职责，避免把所有逻辑塞进 handler 或 worker。

### 2. 失败模式

常见失败包括：超时、取消、资源耗尽、重复请求、部分成功、下游不可用、配置错误、部署回滚、观测缺失。

### 3. 验证方法

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

需要时加入 pprof、GODEBUG、故障注入、集成测试和压测。

---

## 生产实践

在 Production Job Runner 中，本节应落到：

- job_id / request_id / trace_id 全链路传播。
- API、scheduler、worker、repository 的依赖方向清晰。
- 超时、重试、幂等、错误映射有明确策略。
- 关键指标包括 queue depth、duration、error code、retry count、worker active count。
- 每个关键失败路径都有测试或演练脚本。

---

## 代码实验

配套实验目录：

```text
labs/22-deployment-operations/
```

运行：

```bash
cd labs/22-deployment-operations
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

---

## 常见误区

1. 只记 API，不设计边界。
2. 只测 happy path，不测失败路径。
3. 没有指标就开始优化。
4. 让单个组件同时承担协议、业务、存储和观测。
5. 把复杂模式当作生产级的证明。

---

## 故障案例

### 案例：缺少观测导致无法定位瓶颈

线上任务延迟升高，但日志没有 job_id，指标没有 queue depth，trace 没有跨 API/worker/repository 传播，导致无法判断瓶颈在 API、DB、队列还是 worker。

修复：补齐结构化日志、RED/USE 指标、trace context、错误分类和失败路径测试。

---

## 作业

1. 运行本节 Lab 的测试、race detector 和 benchmark。
2. 增加一个失败路径测试。
3. 写一段 Production Job Runner 落地设计。
4. 列出 3 个指标、2 个告警、2 个故障演练。
5. 说明本节设计中哪些地方应保持简单，哪些地方值得抽象。

---

## 评估标准

- 能解释本节主题的生产价值和边界。
- 能写出可测试的最小实现。
- 能识别关键失败模式。
- 能定义观测指标和验证方法。
- 能把主题落到综合项目设计中。

---

## 延伸阅读

- Go standard library documentation
- Effective Go
- Go Code Review Comments
- OpenTelemetry / pprof / runtime documentation as applicable
- 100 Go Mistakes and How to Avoid Them
