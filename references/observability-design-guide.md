# Observability 设计指南

## 1. 三类信号

| 信号 | 用途 | 不适合 |
|---|---|---|
| Logs | 解释单个事件发生了什么 | 聚合趋势 |
| Metrics | 趋势、告警、SLO | 单请求细节 |
| Traces | 跨组件调用链和耗时分布 | 高基数全量字段 |

## 2. RED / USE

HTTP/API 用 RED：

```text
Rate: request/sec
Errors: error rate
Duration: latency histogram
```

资源用 USE：

```text
Utilization: 使用率
Saturation: 排队/等待
Errors: 错误数
```

## 3. Production Job Runner 指标

```text
http_request_duration_seconds{route,method,status}
job_queue_depth{queue}
job_claim_duration_seconds{status}
job_execution_duration_seconds{kind,status}
job_retry_total{kind,reason}
job_dead_letter_total{kind,reason}
worker_active{group}
worker_blocked_total{reason}
db_query_duration_seconds{operation,status}
```

不要把 `job_id` / `request_id` 放进 Prometheus label。

## 4. 日志字段

最低字段：

```text
ts, level, service, version, component, operation,
request_id, trace_id, job_id, worker_id,
duration_ms, status, error_kind, error_message
```

错误分类：

```text
validation, conflict, timeout, dependency, canceled, rate_limited, internal
```

## 5. Trace Span 设计

```text
POST /jobs
  └── SubmitJob
      ├── ValidateJob
      ├── InsertJob
      └── PublishJobCreated

worker.loop
  └── ClaimJob
  └── ExecuteJob
  └── PersistResult
  └── PublishJobFinished
```

Span attribute：

```text
job.id, job.kind, job.attempt, worker.id, db.operation, queue.name
```

## 6. 告警原则

告警应该指向用户影响或明确行动：

- P95/P99 超过 SLO 持续 N 分钟。
- error rate 超过阈值。
- queue depth 持续增长。
- worker active 为 0 但 queue depth > 0。
- DLQ 增长。

不要对每个低级指标都告警。

## 7. Dashboard 布局

```text
一屏：请求量、错误率、延迟、队列深度
二屏：worker active、job duration、retry/DLQ
三屏：DB query、DB pool、Redis/MQ latency
四屏：runtime memory、goroutine、GC、CPU
```
