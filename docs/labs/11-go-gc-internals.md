# Lab 11：Go GC 原理与调优边界

本实验关注 allocation rate，而不是直接“调 GC 参数”。`EncodeAllocHeavy` 故意制造 map/string/buffer 临时对象；`EncodePreallocated` 展示低风险优化方式。

## 运行

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 1. 对比分配

```bash
go test -bench=BenchmarkEncode -benchmem ./...
```

重点看：`B/op` 和 `allocs/op` 是否下降，而不是只看 `ns/op`。

## 2. gctrace

```bash
GODEBUG=gctrace=1 go test -run=^$ -bench=BenchmarkEncodeAllocHeavy -benchmem ./...
```

观察：GC 频率、`before->after MB`、heap goal、GC CPU 百分比。

## 3. Heap profile / alloc profile

```bash
go test -run=^$ -bench=BenchmarkEncodeAllocHeavy -memprofile mem.out ./...
go tool pprof -http=:0 -alloc_space mem.out
go tool pprof -http=:0 -inuse_space mem.out
```

`alloc_space` 找历史分配热点；`inuse_space` 找当前仍存活对象。

## 4. GOGC / GOMEMLIMIT 思考

`WithGOGC` 和 `WithMemoryLimit` 用于演示参数入口。生产中不要先调参数，应先用 benchmark/profile 证明 allocation hot path。

## 生产启发

Production Job Runner 的 GC 压力通常来自：大 payload 长期持有、每 job 构造临时 `map[string]any`、无上限缓存、批处理过大。优先做 payload 外置化、batch 上限、固定日志字段和容量控制。
