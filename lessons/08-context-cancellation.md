# Lesson 8：Context 与取消传播

## 学习目标

完成本课后，学习者应该能够：

- context 是请求生命周期、取消信号、deadline 和请求级元数据的载体。
- context 不是可选参数袋，也不是依赖注入容器。
- 所有外部调用应接收 context 并尊重取消。
- WithCancel/WithTimeout 返回的 cancel 必须调用以释放资源。
- 使用工具和实验验证本课核心机制，而不是只停留在概念理解。
- 将本课机制转化为生产代码中的设计判断、排查路径和 review 标准。

---

## 关键问题

1. context 的职责是什么？
2. cancellation tree 如何工作？
3. deadline 与 timeout 如何传播？
4. context value 应该放什么？
5. 为什么不要把 context 存进 struct？

---

## 核心结论

- context 是请求生命周期、取消信号、deadline 和请求级元数据的载体。
- context 不是可选参数袋，也不是依赖注入容器。
- 所有外部调用应接收 context 并尊重取消。
- WithCancel/WithTimeout 返回的 cancel 必须调用以释放资源。
- graceful shutdown 本质是 context 传播和资源收敛。

---

## 设计哲学

Go 的很多机制都服务于工程协作：让控制流、数据流、错误流和资源生命周期尽量显式。

本课需要持续追问三件事：

1. 这个机制让代码更简单，还是只是让抽象更多？
2. 这个机制在小程序里看起来无所谓，在生产环境下会放大成什么问题？
3. 我们如何用工具验证自己的判断？

对于 `Context 与取消传播`，不要只记 API 或术语，而要理解它对以下方面的影响：

- 可读性
- 可测试性
- 性能
- 并发安全
- 故障隔离
- 可观测性

---

## 底层机制

本课涉及的核心概念：

- `Context interface`
- `cancelCtx`
- `timerCtx`
- `valueCtx`
- `deadline`
- `propagation`
- `errgroup`
- `request scope`

建议讲解顺序：

1. 先用最小代码复现现象。
2. 再解释 runtime、编译器或标准库背后的机制。
3. 最后回到生产代码中应该如何取舍。

### 必讲层

- 机制的基本数据结构或执行模型。
- 常见误区和最小复现。
- 与测试、benchmark、race detector 或 pprof 的验证方式。

### 深入层

- runtime 或编译器层面的实现思路。
- 性能成本和资源生命周期。
- 与生产故障之间的联系。

### 拓展层

- 源码细节、版本差异、极端优化手段只作为延伸阅读，不作为主线要求。

---

## 代码实验

建议实验目录：

```text
labs/08-context-cancellation/
  README.md
  go.mod
  main.go 或 *_test.go
```

实验目标：

- 构造一个最小示例观察本课现象。
- 修改代码触发不同结果。
- 用 Go 工具链验证解释是否正确。

运行命令：

```bash
go test ./...
go test -race ./...
```

实验 README 应包含：

```text
观察目标
运行命令
预期输出
现象解释
变体实验
生产启发
```

---

## 生产实践

- handler 入参使用 r.Context()。
- DB、HTTP client、gRPC 调用都传递 context。
- context value 只放 request id、trace id、auth claims 等请求级数据。
- 不要把 logger/config/db 塞进 context。

生产环境下要额外关注：

- 失败路径是否显式。
- 资源生命周期是否可控。
- 是否存在隐式共享状态。
- 是否能通过日志、指标、trace 或 profile 定位问题。
- 是否有测试覆盖正常路径、边界路径和故障路径。

---

## 常见误区

- 把“能运行”误认为“生产可接受”。
- 在没有 benchmark/profile 证据时做性能判断。
- 用复杂抽象掩盖不清晰的边界。
- 忽略取消、超时、错误包装和资源释放。
- 只测试成功路径，不测试故障和并发路径。

---

## 故障案例

课堂中建议构造一个故障场景：

```text
现象：服务延迟升高、资源持续增长或错误难以定位。
假设：与本课机制相关。
验证：使用测试、race detector、pprof、trace 或日志定位。
修复：调整代码结构、同步策略、错误处理或资源生命周期。
复盘：把经验转化为 review checklist。
```

---

## 作业

1. 写一个最小复现实验，证明本课的一个核心结论。
2. 为实验补充 table-driven tests 或 benchmark。
3. 写一段 300–500 字短文，解释这个机制如何影响生产系统设计。
4. 从现有项目中找一处相关代码，给出 review 建议。

---

## 评估标准

- 能否清楚解释关键问题，而不是背诵术语。
- 能否用命令和实验输出支撑结论。
- 能否识别常见误区并给出替代方案。
- 能否把机制落到生产实践和故障排查。
- 代码是否通过必要的测试、race、benchmark 或构建检查。

---

## 延伸阅读

- Effective Go
- Go Blog
- Go Specification
- Go Memory Model
- Go Code Review Comments
- 100 Go Mistakes and How to Avoid Them
- Go runtime source code（按需阅读，不要求逐行掌握）
