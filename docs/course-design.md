# 课程编排设计

本文档固化《Go 深度进阶与生产级系统设计训练营》的课程设计决策，用于指导后续讲义、实验和综合项目的编写。

---

## 1. 课程定位

本课程不是 Go 入门课，而是面向已有初步 Go 编程经验的学习者的进阶训练营。

课程核心目标是帮助学习者建立三种能力：

1. **解释能力**：能解释 Go 的设计哲学、底层机制和工程取舍。
2. **实现能力**：能写出可测试、可观测、可维护、可演进的 Go 后端代码。
3. **判断能力**：能在生产环境中做合理的组件选择、架构权衡和故障排查。

课程应避免成为松散的 Go 知识百科。每个主题都必须回到一个问题：

> 这个机制如何影响生产代码的设计、性能、可靠性和可维护性？

---

## 2. 学习者画像

### 2.1 默认已掌握

课程默认学习者已经具备：

- Go 基本语法。
- `struct`、method、interface 的基础使用。
- slice、map 的基础使用。
- goroutine、channel 的基础使用。
- `go test` 的基础使用。
- 能写简单 HTTP handler。
- 能读懂常见 Go 项目结构。

### 2.2 不默认掌握

课程不默认学习者理解：

- Go 的设计哲学与工程约束。
- value semantics 与 pointer semantics 的系统性取舍。
- escape analysis。
- slice 底层数组共享和内存保留问题。
- map bucket、扩容、并发安全边界。
- interface 的 `iface`/`eface` 表示与 nil interface 陷阱。
- error wrapping 与错误分层。
- GMP 调度模型。
- channel 内部队列、close semantics、泄漏模式。
- context cancellation tree。
- Go memory model 与 happens-before。
- GC、GOGC、allocation rate。
- pprof、trace、race detector 的生产排查路径。
- HTTP/gRPC 生产级 timeout、graceful shutdown、observability。
- DB 事务边界、缓存策略、消息队列幂等消费。

---

## 3. 总体编排策略

采用 **混合式编排**：前半部分体系化打底，后半部分项目驱动落地。

```text
Lesson 1–8：语言设计、基础概念、runtime 与并发原理，以小实验为主。
Lesson 9–13：memory model、GC、性能诊断与工程工具，以故障实验为主。
Lesson 14–24：生产服务架构与 Production Job Runner 综合项目，以项目演进为主。
```

### 3.1 内容比例

建议比例：

```text
Go 原理与语言机制：40%
生产工程实践：40%
综合项目与故障演练：20%
```

### 3.2 每个主题的闭环

每个主题都按如下闭环组织：

```text
问题引入
→ Go 的设计选择
→ 底层机制
→ 代码实验
→ 生产实践
→ 常见误区
→ 故障案例
→ 作业与评估
```

示例：讲 `context` 时，不只讲 API，而要覆盖：

```text
为什么需要 context
→ cancellation tree 如何传播
→ HTTP/gRPC/DB 如何接入
→ worker graceful shutdown 如何设计
→ context misuse 有哪些
→ 如何用实验验证取消生效
```

---

## 4. 课程周期与节奏

### 4.1 标准周期

推荐采用 **12 周标准版**。

```text
总 lessons：24
节奏：每周 2 lessons
每周投入：3–6 小时
```

每周建议结构：

```text
Lesson A：原理、机制、设计哲学
Lesson B：实验、生产实践、故障案例
课后：作业或项目增量
```

### 4.2 12 周路线图

| 周次 | 主题 | 重点产出 |
|---|---|---|
| Week 1 | Go 设计哲学、值/指针/逃逸 | 设计取舍笔记、escape lab |
| Week 2 | slice/map/string、interface | 基础概念底层实验 |
| Week 3 | error handling、GMP | 错误分层实验、scheduler 观察 |
| Week 4 | channel、context | worker/cancel 小实验 |
| Week 5 | sync/atomic/memory model、并发故障 | race/leak/block profile 实验 |
| Week 6 | GC、benchmark、pprof、编译部署 | GC/pprof 性能诊断实验 |
| Week 7 | 项目结构、HTTP API | Production Job Runner API 骨架 |
| Week 8 | gRPC、PostgreSQL | gRPC API、DB 持久化 |
| Week 9 | Redis、消息队列 | cache/idempotency/event flow |
| Week 10 | Observability、Reliability | logs/metrics/traces/retry/backpressure |
| Week 11 | Deployment、系统设计案例 | Docker Compose、故障演练 |
| Week 12 | 综合项目 Review | 架构、代码、性能、可观测性 review |

---

## 5. Lesson 编写模板

每节课固定使用如下结构：

```text
# Lesson N：标题

## 学习目标
## 关键问题
## 核心结论
## 设计哲学
## 底层机制
## 代码实验
## 生产实践
## 常见误区
## 故障案例
## 作业
## 评估标准
## 延伸阅读
```

### 5.1 学习目标

学习目标要描述学习者完成后能做什么，而不是只列知识点。

好的写法：

```text
能用 escape analysis 输出解释某段代码为什么发生堆分配。
```

不好的写法：

```text
了解 escape analysis。
```

### 5.2 关键问题

每课开头必须列出 5–8 个关键问题。

例如 interface 课：

```text
interface 的零值是什么？
nil interface 为什么不等于 nil？
接口应该定义在实现方还是使用方？
构造函数为什么通常不返回 interface？
```

### 5.3 核心结论

每课必须给出可以直接用于生产判断的结论。

例如：

```text
不要为每个实现提前定义 interface；interface 应该从使用方需要的最小行为中长出来。
```

### 5.4 代码实验

每课至少包含一个可运行实验，实验必须包括：

- 实验目标。
- 代码路径。
- 运行命令。
- 预期现象。
- 解释。
- 变体实验。
- 生产启发。

### 5.5 生产实践

生产实践部分必须回答：

- 什么情况下应该这样做？
- 什么情况下不应该这样做？
- 规模变大后会出现什么问题？
- 如何观测、验证、排查？

---

## 6. 深度控制原则

每节课分三层：

### 6.1 必讲层

所有学习者必须掌握。正文重点讲。

例：slice 课必讲：

- slice header。
- len/cap。
- append 扩容。
- 底层数组共享。
- nil slice vs empty slice。

### 6.2 深入层

帮助学习者建立底层理解，但不逐行展开源码。

例：slice 课深入层：

- `runtime.growslice`。
- 扩容策略。
- pointer slice 与 GC 扫描压力。
- escape 与 allocation。

### 6.3 拓展层

只放延伸阅读或附录，避免主课失控。

例：slice 课拓展层：

- unsafe 构造 slice header。
- Go 版本间扩容策略差异。
- runtime 源码逐行阅读。

每节课应明确“本节不展开什么”，防止资料无限膨胀。

---

## 7. 实验规范

实验目录建议：

```text
labs/
  01-design-philosophy/
  02-escape-analysis/
  03-slice-map-string/
  04-interface/
```

每个实验目录包含：

```text
README.md
源码文件
测试文件
可选 benchmark 文件
```

实验 README 固定结构：

```text
# Lab N：标题

## 观察目标
## 运行命令
## 预期输出
## 现象解释
## 变体实验
## 生产启发
```

### 7.1 基础验证命令

所有实验尽量支持：

```bash
go test ./...
go test -race ./...
go vet ./...
```

性能相关实验额外支持：

```bash
go test -bench=. -benchmem ./...
```

逃逸分析实验额外支持：

```bash
go build -gcflags="-m -m" ./...
```

调度器实验额外支持：

```bash
GODEBUG=schedtrace=1000,scheddetail=1 go run .
```

GC 实验额外支持：

```bash
GODEBUG=gctrace=1 go run .
```

---

## 8. 综合项目编排

综合项目为 **Production Job Runner**。

项目从 Lesson 14 正式开始，不从第一课开始。这样前半程可以专注底层机制，后半程把机制落到生产项目中。

### 8.1 项目复杂度

采用 **API + Worker 双进程**，暂不做多服务微服务架构。

原因：

- 足够接近生产环境。
- 能覆盖 API、DB、queue、worker、observability、shutdown。
- 不会过早陷入服务治理、部署拓扑和分布式系统复杂度。

### 8.2 项目演进路线

```text
Lesson 14：项目骨架、配置、日志、分层设计
Lesson 15：HTTP API、middleware、error response、graceful shutdown
Lesson 16：gRPC API、protobuf、interceptor、deadline
Lesson 17：PostgreSQL、migration、repository、transaction
Lesson 18：Redis、cache、rate limit、idempotency key
Lesson 19：NATS/Kafka、event publishing、idempotent consumer、DLQ
Lesson 20：logs、metrics、traces、pprof、health/readiness
Lesson 21：timeout、retry、backpressure、circuit breaker、degradation
Lesson 22：Docker Compose、config/secrets、deployment、shutdown
Lesson 23：故障注入：slow DB、duplicate event、worker crash、goroutine leak
Lesson 24：最终 Review：架构、代码、性能、可观测性、可靠性
```

### 8.3 项目边界

第一版不做：

- Kubernetes 深入部署。
- 多租户权限系统。
- 复杂 UI。
- 完整分布式调度算法。
- 高可用 leader election。

这些作为拓展主题保留。

---

## 9. 评估方式

### 9.1 每课评估

每课作业应至少覆盖：

- 一个概念解释题。
- 一个代码实验题。
- 一个生产判断题。

示例：

```text
解释：为什么返回 typed nil error 会导致 err != nil？
实验：写一个最小复现并用测试覆盖。
生产判断：如何设计避免这类错误进入业务代码？
```

### 9.2 阶段评估

每个模块结束后做一次阶段 Review：

- 设计问题讨论。
- 代码 Review。
- 实验结果解释。
- 常见坑复盘。

### 9.3 最终评估

最终以 Production Job Runner 为评估对象：

- 架构清晰度。
- 并发控制正确性。
- context 取消完整性。
- 错误分层合理性。
- 事务边界明确性。
- 消费者幂等性。
- logs/metrics/traces 是否足够排障。
- pprof 是否能定位性能问题。
- 测试是否覆盖核心路径与故障路径。

---

## 10. 资料形态

第一阶段采用：

```text
Markdown 讲义 + 可运行 labs + 综合项目文档
```

后续再补：

```text
instructor-notes/
```

讲师版资料用于记录：

- 每节课时间分配。
- 课堂提问顺序。
- 常见误解。
- 作业点评标准。
- 讲解重点和可跳过内容。

---

## 11. 编写优先级

后续编写顺序：

1. 补齐 Lesson 2–5：基础概念底层机制。
2. 为 Lesson 2–5 配套 labs。
3. 补齐 Lesson 6–10：runtime 与并发。
4. 补齐 Lesson 11–13：GC、pprof、构建部署。
5. 从 Lesson 14 开始建立 Production Job Runner 项目骨架。
6. 每完成一组 lessons，更新 `mkdocs.yml` 导航。

近期优先级：

```text
Lesson 2：值、指针与逃逸分析
Lab 2：escape analysis 实验
Lesson 3：Slice、Map、String 底层结构
Lab 3：slice/map/string 实验
```
