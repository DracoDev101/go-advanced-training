# Lab 12：Benchmark 与 pprof 性能诊断

本实验展示三件事：可信 benchmark、优化前后对比、pprof 定位。

## 运行

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

## 1. Bad benchmark vs Good benchmark

```bash
go test -bench='Benchmark(Bad|Good)Summary' -benchmem ./...
```

`BenchmarkBadSummaryIncludesSetup` 把输入构造也算进热路径；`BenchmarkGoodSummaryExcludesSetup` 用 `b.ResetTimer()` 排除 setup。

## 2. 优化前后对比

```bash
go test -bench='Benchmark(Slow|Fast)Summary' -benchmem -count=10 ./... > summary.txt
```

如果安装了 benchstat：

```bash
go install golang.org/x/perf/cmd/benchstat@latest
# 分别保存 old.txt/new.txt 后：
benchstat old.txt new.txt
```

## 3. CPU profile

```bash
go test -run=^$ -bench=BenchmarkSlowSummary -cpuprofile cpu.out ./...
go tool pprof -http=:0 cpu.out
```

在 pprof 中看 `top`、`list SlowSummary`。

## 4. Heap / alloc profile

```bash
go test -run=^$ -bench=BenchmarkSlowSummary -benchmem -memprofile mem.out ./...
go tool pprof -http=:0 -alloc_space mem.out
```

`alloc_space` 更适合找分配热点；`inuse_space` 更适合找长期持有。

## 生产启发

Production Job Runner 的性能优化必须走闭环：指标发现瓶颈 → pprof 定位 → benchmark 重现 → 最小修改 → benchstat/线上指标验证。不要先凭直觉改 JSON、锁或缓存。
