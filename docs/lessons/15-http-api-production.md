# Lesson 15：生产级 HTTP API

## 学习目标

完成本课后，学习者应该能够：

- 设计生产级 HTTP server。
- 实现 middleware 链。
- 处理 timeout、validation 与错误映射。
- 理解 graceful shutdown 在 HTTP 中的落地。
- 将本节主题落到 Production Job Runner 的生产设计中。

---

## 关键问题

1. 这个主题解决的生产问题是什么？
2. Go 标准库或主流生态提供了哪些基础能力？
3. 这个能力的默认行为、边界和失败模式是什么？
4. 如何通过测试、benchmark、profile 或故障演练验证设计？
5. 在 Production Job Runner 中，这个主题应该如何落地？

---

## 核心结论

- 本节关键词：**http、middleware、timeout、validation、shutdown、handler**。
- 生产级 Go 课程不只讲 API 用法，更要讲设计边界、故障模式和观测方式。
- 所有工程决策都应能回答：为什么这样设计、失败时如何表现、如何验证、如何回滚。
- 简单实现优先；只有在指标证明瓶颈存在时，再引入复杂优化。
- 本节配套 Lab 提供最小可运行实验，用于把抽象概念变成可观察现象。

---

## 设计哲学

Go 的工程哲学偏向直接、可读、可组合。对于 **生产级 HTTP API**，不要把问题理解成“选一个库”或“套一个模式”，而要从系统边界出发：

```text
输入是什么？
输出是什么？
谁拥有状态？
失败如何传播？
超时和取消如何生效？
如何观测？
如何测试？
```

好的 Go 代码通常不是抽象层数最多的代码，而是边界清楚、依赖方向稳定、失败路径明确的代码。

---

## 底层机制与核心概念

### 1. 语义边界

先明确本节主题的语义边界。不要让一个组件同时承担过多职责。比如：

- API 层负责协议、校验、错误映射。
- Service 层负责业务编排和事务边界。
- Repository 层负责持久化细节。
- Worker 层负责异步执行和生命周期。
- Observability 层负责日志、指标、追踪，而不是业务决策。

### 2. 失败模式

每个生产组件都要列出失败模式：

```text
超时
取消
并发冲突
资源耗尽
下游错误
部分成功
重复执行
观测缺失
```

### 3. 验证方式

验证不能只靠“手动跑一下”。应至少包含：

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

涉及性能或 runtime 的主题，还应加入 pprof、trace、GODEBUG 或故障注入。

---

## 生产实践

### Production Job Runner 落地点

在综合项目中，本节内容应落到以下问题：

- API/worker/repository 的边界是否清晰？
- context 是否全链路传递？
- timeout、retry、幂等和错误映射是否明确？
- 是否有足够指标判断系统健康？
- 是否有测试证明关键失败路径？
- 是否能在部署时快速回滚？

### 工程 checklist

```text
是否有单元测试？
是否有 race detector 验证？
是否有 benchmark 或 profile？
是否记录 request_id / job_id / trace_id？
是否有健康检查？
是否有容量和超时配置？
是否有故障演练？
```

---

## 代码实验

配套实验目录：

```text
labs/15-http-api-production/
```

运行：

```bash
cd labs/15-http-api-production
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

实验目标：

- 用最小代码复现本节核心机制。
- 用测试固定正确行为。
- 用 benchmark 或 profile 建立优化前后的可观察对比。
- 总结生产启发。

---

## 常见误区

1. 只记住 API，不理解边界。
2. 只写 happy path，不测试失败路径。
3. 只看平均延迟，不看 tail latency 和错误率。
4. 用复杂模式掩盖需求不清。
5. 没有指标就开始优化。

---

## 故障案例

### 案例：设计中缺少边界和观测

症状：线上出现延迟升高或任务堆积，但无法判断是 API、DB、队列、worker 还是下游服务导致。

根因：组件边界不清，日志缺少 job_id/request_id，指标缺少 queue depth、duration、error code。

修复：补齐结构化日志、关键指标、错误分类和 context 传播，并增加失败路径测试。

---

## 作业

1. 阅读本节 Lab，运行所有测试和 benchmark。
2. 为 Lab 增加一个失败路径测试。
3. 写一段设计说明：这个主题在 Production Job Runner 中如何落地。
4. 列出 3 个可观测指标和 2 个故障演练场景。
5. 对比两种实现方案，说明你会选择哪一个以及原因。

---

## 评估标准

- 能解释本节核心概念和设计边界。
- 能写出可测试的最小实现。
- 能识别至少三个生产失败模式。
- 能说明如何观测和验证。
- 能将本节内容映射到综合项目。

---

## 延伸阅读

- Go standard library documentation
- Effective Go
- Go Code Review Comments
- 100 Go Mistakes and How to Avoid Them
- OpenTelemetry / pprof / runtime documentation as applicable
