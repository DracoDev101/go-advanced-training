# 综合项目：Production Job Runner

## 项目目标

构建一个生产级任务调度与执行系统，用于贯穿整套 Go 深度进阶课程。

项目覆盖：

- HTTP API
- gRPC API
- PostgreSQL 持久化
- Redis 缓存、限流与幂等
- NATS/Kafka 事件发布
- worker pool
- context cancellation
- graceful shutdown
- Prometheus metrics
- OpenTelemetry tracing
- pprof
- Docker Compose
- 故障注入与性能优化

## 核心 API

```text
POST   /jobs
GET    /jobs/{id}
GET    /jobs
POST   /jobs/{id}/cancel
GET    /jobs/{id}/logs
GET    /healthz
GET    /readyz
GET    /metrics
```

## Job 状态机

```text
PENDING -> RUNNING -> SUCCEEDED
PENDING -> RUNNING -> FAILED
PENDING -> RUNNING -> TIMEOUT
PENDING -> CANCELLED
RUNNING -> CANCELLED
```

## 推荐技术栈

```text
Go
chi
pgx
PostgreSQL
Redis
NATS 或 Kafka
slog
Prometheus
OpenTelemetry
pprof
Docker Compose
```

## 目录草案

```text
production-job-runner/
  cmd/
    api/
      main.go
    worker/
      main.go
  internal/
    config/
    logging/
    httpapi/
    grpcapi/
    service/
    repository/
    worker/
    queue/
    metrics/
    tracing/
  migrations/
  proto/
  deploy/
    docker-compose.yml
  tests/
  Makefile
  README.md
```

## 评估维度

- 架构是否清晰
- 并发控制是否可靠
- context 取消是否完整
- 任务状态机是否正确
- 数据库事务边界是否明确
- 消费者是否幂等
- timeout/retry/backoff 是否合理
- logs/metrics/traces 是否足够排障
- pprof 是否能定位性能问题
- 测试是否覆盖核心路径与故障路径
