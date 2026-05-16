# Labs

本目录用于存放课程实验代码。

建议每个实验使用独立目录：

```text
labs/
  01-design-philosophy/
  02-escape-analysis/
  03-slice-map-string/
  04-interface/
  05-error-handling/
```

## 已有实验

- Lab 2：值、指针与逃逸分析：`02-escape-analysis/`
- Lab 3：Slice、Map、String 底层结构：`03-slice-map-string/`
- Lab 4：Interface 深入：`04-interface-deep-dive/`
- Lab 5：Error Handling 生产实践：`05-error-handling/`
- Lab 6：Goroutine 与 GMP 调度模型：`06-goroutine-scheduler/`
- Lab 7：Channel 原理与使用边界：`07-channel-boundaries/`
- Lab 8：Context 与取消传播：`08-context-cancellation/`
- Lab 9：sync、atomic 与 Go Memory Model：`09-sync-atomic-memory-model/`
- Lab 10：并发 Debugging 与 Race Detector：`10-concurrency-debugging/`
- Lab 11：Go GC 原理与调优边界：`11-go-gc-internals/`
- Lab 12：Benchmark 与 pprof 性能诊断：`12-benchmark-pprof/`
- Lab 13：Go Build、Link 与部署形态：`13-build-link-deploy/`
- Lab 14：项目结构与分层设计：`14-project-structure-layering/`
- Lab 15：生产级 HTTP API：`15-http-api-production/`
- Lab 16：生产级 gRPC：`16-grpc-production/`
- Lab 17：PostgreSQL、事务与 Repository：`17-postgres-transaction-repository/`
- Lab 18：Redis、缓存与一致性边界：`18-redis-cache-boundaries/`
- Lab 19：消息队列、事件与幂等：`19-message-queue-events/`
- Lab 20：Observability：日志、指标与追踪：`20-observability/`
- Lab 21：可靠性模式：`21-reliability-patterns/`
- Lab 22：部署与运行维护：`22-deployment-operations/`
- Lab 23：系统设计案例：`23-system-design-cases/`
- Lab 24：综合项目 Review：`24-final-project-review/`

## 通用要求

实验代码要求：

```bash
go test ./...
go test -race ./...
go vet ./...
```

性能相关实验额外要求：

```bash
go test -bench=. -benchmem ./...
```

逃逸分析实验额外要求：

```bash
go build -gcflags="-m -m" ./...
```
