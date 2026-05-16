# Lab 20：Observability for Job Runner

本实验用无外部依赖的最小模型练习 logs、metrics、traces 的职责边界。

## 运行

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 观察点

- `StructuredLog` 固定字段：`request_id`、`trace_id`、`job_id`、`component`、`operation`、`duration_ms`、`error_kind`。
- `Metrics` 用低基数字段：`kind`、`status`、`reason`；不要把 `job_id` 放 label。
- `Tracer` 把 `trace_id` 和 `job.id` 放 span attributes，用于单请求链路。
- `JobRunner.Execute` 演示失败时同时记录 duration、retry counter、span。

## 生产指标建议

```text
http_request_duration_seconds{route,method,status}
job_queue_depth{queue}
job_execution_duration_seconds{kind,status}
job_retry_total{kind,reason}
worker_active{group}
```

## 生产启发

Logs 解释单个事件，Metrics 用于趋势和告警，Traces 串联跨组件路径。三者必须通过 `request_id` / `trace_id` / `job_id` 关联。
