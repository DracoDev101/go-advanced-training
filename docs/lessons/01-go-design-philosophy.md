# Lesson 1：Go 的设计哲学与工程化取舍

## 学习目标

完成本课后，学习者应该能够：

- 解释 Go 诞生时试图解决的工程问题。
- 理解 Go 为什么强调简单、明确、可读与统一工具链。
- 理解 Go 为什么没有继承、没有异常、长期没有泛型。
- 理解 error as value、composition over inheritance、implicit interface、小接口的设计意义。
- 能识别 Java-style Go、过度抽象 Go、idiomatic Go 之间的区别。
- 能对一段过度设计的 Go 代码做工程化重构。

---

## 1. 问题引入

Go 不是为了让单个程序员写出最“优雅”的抽象而设计的。

Go 更关注这些问题：

- 大规模工程里代码能否被普通工程师快速读懂？
- 构建、测试、部署是否足够简单稳定？
- 新人能否快速进入项目？
- 依赖关系是否清晰？
- 错误路径是否显式？
- 并发程序是否能以较低认知成本表达？
- 工具链是否统一，减少团队争论？

Go 的很多设计看起来“少”，本质上是主动限制复杂度。

> Go 的核心哲学不是表达力最大化，而是工程协作成本最小化。

---

## 2. Go 解决的真实问题

Go 诞生于 Google 内部的大规模软件工程环境。它面对的问题不是“如何设计一门理论上最完美的语言”，而是：

- C++ 编译慢、复杂度高。
- Java 运行时较重，部署与性能有固定成本。
- Python 开发快但性能、部署、类型约束不足。
- 大型团队协作中，过度抽象会迅速恶化可维护性。

Go 的选择是：

```text
快速编译
静态类型
单二进制部署
统一格式化
内置测试工具
轻量并发
简单语言特性
显式错误处理
```

这也是为什么 Go 代码常常显得“朴素”。这种朴素是刻意设计的。

---

## 3. 简单性不是简陋

Go 的简单性主要来自几类约束。

### 3.1 语法约束

Go 故意减少语法糖：

- 没有隐式类型转换。
- 没有宏。
- 没有操作符重载。
- 没有复杂继承层次。
- 泛型出现很晚，并且设计克制。

这些限制降低了阅读成本。

在 Go 里，代码通常应该让读者直接看到控制流、数据流和错误流。

### 3.2 格式约束

`gofmt` 消灭了大量无意义争论。

团队不需要讨论：

- 大括号风格。
- 缩进风格。
- import 排序。
- 对齐风格。

统一格式本身就是工程效率工具。

### 3.3 抽象约束

Go 支持抽象，但不鼓励过度抽象。

典型 Go 风格：

```go
type Store interface {
    Get(ctx context.Context, id string) (User, error)
}
```

典型过度抽象：

```go
type AbstractUserRepositoryFactoryProvider interface {
    CreateUserRepositoryFactory() UserRepositoryFactory
}
```

Go 的接口更适合描述最小行为，而不是搭建复杂类型体系。

---

## 4. 为什么没有继承

Go 用组合替代继承。

继承的问题：

- 父类变更影响子类。
- 层级越深，行为越难预测。
- 代码复用和多态被绑定在一起。
- 大型工程里容易形成脆弱基类问题。

Go 的组合方式：

```go
type Logger struct{}

func (l Logger) Info(msg string) {
    // ...
}

type UserService struct {
    logger Logger
}
```

或者通过嵌入：

```go
type AuditedService struct {
    Logger
}
```

但生产代码中也要谨慎使用 embedding，因为它可能隐藏 API surface。

### 实践原则

```text
优先显式字段组合。
需要暴露被嵌入类型方法时，再考虑 embedding。
不要为了模拟继承而使用 embedding。
```

---

## 5. 为什么没有异常

Go 使用显式 error 返回值。

典型代码：

```go
user, err := repo.GetUser(ctx, id)
if err != nil {
    return fmt.Errorf("get user %s: %w", id, err)
}
```

这看起来啰嗦，但有几个重要优点：

- 错误路径可见。
- 调用方必须面对失败。
- 错误可以带上下文逐层包装。
- 系统边界处可以明确做错误映射。

异常的问题是：

- 控制流被隐藏。
- 函数签名不一定表达失败类型。
- 大型系统中容易出现过宽的 catch。
- 失败路径容易被当成非主路径忽略。

Go 的选择是让失败成为普通控制流的一部分。

### 生产错误分层示例

```text
sql.ErrNoRows
    ↓ repository wraps
repository: find user by id: sql: no rows in result set
    ↓ service maps
ErrUserNotFound
    ↓ transport maps
HTTP 404 / gRPC NotFound
```

---

## 6. Interface：隐式实现与小接口

Go 的 interface 是隐式实现的。

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

任何类型只要有 `Read` 方法，就自动实现 `Reader`。

这带来几个结果：

- 实现方不需要依赖接口定义方。
- 接口可以在使用方定义。
- 更容易形成小接口。
- 测试替身更容易构造。

### 接口应该小

好接口：

```go
type Clock interface {
    Now() time.Time
}
```

坏接口：

```go
type UserRepository interface {
    Create(ctx context.Context, u User) error
    Update(ctx context.Context, u User) error
    Delete(ctx context.Context, id string) error
    Find(ctx context.Context, id string) (User, error)
    List(ctx context.Context, filter UserFilter) ([]User, error)
    Count(ctx context.Context, filter UserFilter) (int, error)
    BulkImport(ctx context.Context, users []User) error
    RebuildIndex(ctx context.Context) error
}
```

接口越大，实现成本越高，测试越困难，替换越不灵活。

### 常用原则

```text
Accept interfaces, return concrete types.
```

这不是绝对规则，但通常有助于 API 设计：

- 参数接受行为约束。
- 返回具体类型保留扩展能力。

---

## 7. Goroutine 与 Channel 的哲学

Go 的并发哲学常被概括为：

```text
Do not communicate by sharing memory; instead, share memory by communicating.
```

但这句话经常被误解。

它不是说所有并发都必须用 channel。

更准确的实践是：

```text
channel 适合传递 ownership、事件、结果、取消信号。
mutex 适合保护共享状态。
```

错误用法：为了“Go 风格”强行用 channel 管理所有状态。

正确用法：根据数据关系选择工具。

### 示例判断

| 场景 | 更适合 |
|---|---|
| worker pool 分发任务 | channel |
| 保护内存 map | mutex/RWMutex |
| 广播取消 | context |
| 统计计数器 | atomic 或 mutex |
| pipeline 处理流式数据 | channel |
| 共享 LRU cache | mutex |

---

## 8. 工程化优先

Go 的工具链是语言体验的一部分。

常用工具：

```bash
go test ./...
go test -race ./...
go test -bench=. ./...
go vet ./...
gofmt -w .
go mod tidy
go tool pprof
```

Go 的生产工程实践不是只靠语言本身，而是语言 + 工具链 + 约定共同形成的。

---

## 9. 反模式：Java-style Go

下面是一段刻意写得过度抽象的代码。

```go
type UserServiceInterface interface {
    CreateUser(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error)
}

type UserServiceImpl struct {
    userRepository UserRepositoryInterface
    logger         LoggerInterface
}

func NewUserServiceImpl(
    userRepository UserRepositoryInterface,
    logger LoggerInterface,
) UserServiceInterface {
    return &UserServiceImpl{
        userRepository: userRepository,
        logger:         logger,
    }
}

type UserRepositoryInterface interface {
    SaveUser(ctx context.Context, user *UserEntity) error
}

type LoggerInterface interface {
    LogInfo(message string)
    LogError(message string)
}
```

问题：

- 类型名重复表达 Interface/Impl。
- 过早为 service 定义接口。
- 构造函数返回接口，隐藏具体类型扩展能力。
- Logger interface 不是从使用方最小行为出发设计的。
- Entity/Request/Response 命名受 Java 风格影响明显。

更 Go 的版本：

```go
type UserStore interface {
    Save(ctx context.Context, user User) error
}

type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}

type UserService struct {
    store  UserStore
    logger Logger
}

func NewUserService(store UserStore, logger Logger) *UserService {
    return &UserService{store: store, logger: logger}
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (User, error) {
    user := User{
        ID:    req.ID,
        Email: req.Email,
    }

    if err := s.store.Save(ctx, user); err != nil {
        return User{}, fmt.Errorf("save user: %w", err)
    }

    s.logger.Info("user created", "user_id", user.ID)
    return user, nil
}
```

这个版本并不一定完美，但更符合 Go 的常见取舍：

- 接口小而具体。
- 构造函数返回具体类型。
- 错误包装保留上下文。
- 命名减少噪音。
- service 暴露真实行为，而不是框架式抽象。

---

## 10. 代码实验

### 实验 1：识别过度抽象

给定一段 Java-style Go 代码，标记：

- 哪些 interface 没必要？
- 哪些名字有冗余？
- 哪些层级只是搬运数据？
- 哪些错误缺少上下文？
- 哪些地方返回接口不合适？

### 实验 2：重构为 idiomatic Go

要求：

- 去掉无意义 interface。
- 保留真正有测试价值或边界价值的 interface。
- 构造函数返回具体类型。
- 使用 `fmt.Errorf("...: %w", err)` 包装错误。
- 添加 table-driven tests。

### 实验 3：接口大小比较

写两个版本：

1. 一个大 repository interface。
2. 多个按使用方定义的小 interface。

比较测试替身实现复杂度。

---

## 11. 课堂讨论题

1. Go 的简单性是优势还是限制？
2. 显式错误处理是否会导致样板代码太多？
3. 为什么 Go 不鼓励复杂继承层级？
4. interface 定义在实现方和使用方有什么区别？
5. 构造函数返回 interface 有哪些问题？
6. channel 是否应该作为所有并发问题的默认答案？
7. Go 的工程文化对大型团队有什么价值？

---

## 12. 作业

### 作业：重构 Java-style Go

创建一个小模块，实现用户注册逻辑。

初始版本故意包含：

- `UserServiceInterface`
- `UserServiceImpl`
- `UserRepositoryInterface`
- `LoggerInterface`
- 构造函数返回 interface
- 错误没有 `%w`
- 没有 table-driven tests

要求重构后：

- 命名符合 Go 风格。
- 只保留必要 interface。
- service 构造函数返回具体类型。
- 错误链可被 `errors.Is` 或 `errors.As` 检查。
- 增加 table-driven tests。
- `go test ./...` 通过。

### 评估标准

- 代码是否更少但表达更清晰。
- interface 是否按使用方最小行为定义。
- 错误是否保留上下文。
- 测试是否覆盖成功、重复用户、存储失败等场景。
- 是否避免无意义分层。

---

## 13. 延伸阅读

- Effective Go
- Go Proverbs
- Go Blog: Errors are values
- Go Blog: Error handling and Go
- Go Code Review Comments
- 100 Go Mistakes and How to Avoid Them
