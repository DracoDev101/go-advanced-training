# Go 深度进阶与生产级系统设计训练营：课程大纲

## 模块一：Go 设计哲学与工程观

### Lesson 1：Go 的设计哲学与工程化取舍

- Go 解决的真实问题是什么
- 简单性与工程协作成本
- 显式错误处理的系统设计意义
- 组合优于继承
- 隐式 interface 与小接口
- goroutine/channel 背后的并发哲学
- idiomatic Go vs Java-style Go

## 模块二：基础概念深度理解

### Lesson 2：值、指针与逃逸分析

- value semantics vs pointer semantics
- stack vs heap
- escape analysis
- interface、closure、goroutine 导致的逃逸
- struct 值传递与指针传递的取舍
- benchmark 验证而不是凭直觉优化

### Lesson 3：Slice、Map、String 底层结构

- slice header、len、cap、底层数组共享
- append 扩容与 subslice 内存保留问题
- map bucket、overflow bucket、扩容与随机遍历
- string immutability、byte/rune、字符串拼接性能

### Lesson 4：Interface 深入

- iface 与 eface
- nil interface 陷阱
- dynamic dispatch 成本
- 小接口设计
- accept interfaces, return concrete types
- interface 应该定义在使用方还是实现方

### Lesson 5：Error Handling 生产实践

- error as value
- sentinel error
- custom error type
- wrapping、errors.Is、errors.As、errors.Join
- repository/domain/transport 错误分层
- panic/recover 边界

## 模块三：Runtime 与并发原理

### Lesson 6：Goroutine 与 GMP 调度模型

- G/M/P
- work stealing
- syscall blocking
- netpoller
- async preemption
- stack growth
- GOMAXPROCS

### Lesson 7：Channel 原理与使用边界

- channel 内部结构
- sendq/recvq
- buffered/unbuffered channel
- select 与 close semantics
- channel vs mutex
- 背压、泄漏与死锁

### Lesson 8：Context 与取消传播

- cancellation tree
- deadline/timeout
- context value 使用边界
- errgroup
- graceful shutdown
- HTTP/gRPC/DB 中的 context 传播

### Lesson 9：Sync、Atomic 与 Memory Model

- happens-before
- mutex/rwmutex/once/waitgroup
- atomic
- race detector
- safe publication
- loop variable capture 与数据竞争

### Lesson 10：并发故障排查

- goroutine leak 模式
- pprof goroutine/block/mutex profile
- go tool trace
- scheduler trace
- 生产排查路径

## 模块四：GC、内存与性能

### Lesson 11：Go GC 原理

- concurrent mark-sweep
- tri-color marking
- write barrier
- GOGC
- memory limit
- STW
- allocation rate

### Lesson 12：Benchmark 与 pprof

- benchmark 设计
- benchmem
- CPU profile
- heap profile
- mutex/block profile
- trace
- benchmark-driven optimization

### Lesson 13：编译、链接与部署

- go build flags
- static binary
- cross compile
- ldflags
- trimpath
- container image size
- race build

## 模块五：生产级服务架构

### Lesson 14：项目结构与分层设计

- cmd/internal/pkg
- transport/application/domain/infrastructure
- service/repository
- dependency injection
- config 与 logger 初始化

### Lesson 15：HTTP API 生产实践

- chi 与 net/http
- middleware chain
- timeout
- graceful shutdown
- validation
- pagination
- idempotency
- rate limit
- error response schema

### Lesson 16：gRPC 生产实践

- protobuf 设计
- backward compatibility
- interceptor
- metadata
- deadline
- status code
- health/reflection
- streaming backpressure

## 模块六：数据与异步组件

### Lesson 17：PostgreSQL 与事务设计

- pgx
- connection pool
- migration
- transaction boundary
- isolation level
- lock/deadlock/retry
- outbox pattern
- sqlc 取舍

### Lesson 18：Redis 使用边界

- cache aside
- TTL jitter
- penetration/breakdown/avalanche
- distributed rate limit
- idempotency key
- lock 使用边界

### Lesson 19：消息队列与事件驱动

- NATS/Kafka/RabbitMQ/Redis Stream 选型
- at-least-once
- idempotent consumer
- retry 与 DLQ
- schema evolution
- outbox pattern

## 模块七：可靠性与运维设计

### Lesson 20：Observability

- structured logging
- Prometheus metrics
- OpenTelemetry tracing
- pprof
- correlation id
- health/readiness

### Lesson 21：可靠性模式

- timeout
- retry with backoff
- circuit breaker
- bulkhead
- rate limit
- backpressure
- graceful degradation
- idempotency

### Lesson 22：部署与运行

- Docker
- Docker Compose
- Kubernetes 基础
- config/secrets
- health/readiness/liveness
- zero-downtime shutdown

### Lesson 23：系统设计案例

- 高并发 API 服务
- 任务调度系统
- 事件驱动系统
- 实时数据处理服务
- 内部 RPC 服务

### Lesson 24：综合项目 Review

- 架构 review
- 代码 review
- 性能 review
- observability review
- 故障演练
