# Go 生产问题排查手册

这份手册用于训练“看到症状后如何系统定位”，不是工具命令清单。排查顺序固定为：

```text
现象分型 → 复现/确认影响面 → 采集证据 → 缩小组件 → 形成假设 → 最小验证 → 修复根因 → 回归保护
```

核心原则：**没有证据前不修；没有 root cause 前不改架构；没有回归测试/指标前不宣称修复。**

---

## 1. 首轮分诊：先判断是哪类问题

| 现象 | 第一判断 | 首选证据 | 常用工具 |
|---|---|---|---|
| P99 升高、CPU 高 | CPU bound / 分配过高 / 热点函数 | CPU profile、alloc profile、RED 指标 | `pprof cpu`, `pprof allocs`, benchmark |
| P99 升高、CPU 低 | 阻塞、锁竞争、连接池、下游慢 | goroutine dump、block/mutex profile、DB pool 指标 | `goroutine?debug=2`, block/mutex profile |
| RSS/heap 持续上涨 | 泄漏或 live heap 增长 | heap inuse、gctrace after-GC、对象引用链 | `pprof heap`, `gctrace`, `runtime.MemStats` |
| GC 频繁、P99 抖动 | allocation rate 过高 | `alloc_space`、`B/op`、gctrace 频率 | `benchmem`, `pprof -alloc_space` |
| goroutine 数上涨 | goroutine leak / 下游卡住 | goroutine dump 聚类、队列/连接池指标 | goroutine profile, trace |
| queue depth 上涨 | worker 不消费 / 消费慢 / claim 失败 | worker active、claim latency、DB/queue 指标 | logs + metrics + pprof |
| 错误率上涨 | 依赖不可用 / 超时 / 部署回归 | error_kind、status code、trace span | logs, traces, deploy history |
| 偶发错误结果 | data race / 业务竞态 / 幂等缺失 | race detector、事务日志、重复 job 记录 | `go test -race`, DB audit |
| 只在发布后出现 | 配置/依赖/镜像/迁移差异 | diff、version、env、migration log | `git diff`, `go version -m`, deploy log |

---

## 2. 证据采集顺序

### 2.1 先问四个问题

```text
什么时候开始？是否与发布、配置、流量变化有关？
影响面多大？单 endpoint / 单 worker / 全服务？
是变慢、错误、卡死、泄漏，还是结果不一致？
有没有 request_id / job_id / trace_id 能串起一次失败？
```

### 2.2 最小线上采集包

生产环境建议一次性采集：

```bash
# 版本与构建信息
go version -m ./server

# goroutine dump
curl -s http://127.0.0.1:6060/debug/pprof/goroutine?debug=2 > goroutine.txt

# CPU profile：高 CPU 或整体慢时采集 30s
curl -s 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30' > cpu.out

# heap profile：内存问题采集
curl -s http://127.0.0.1:6060/debug/pprof/heap > heap.out

# allocs profile：分配速率问题采集
curl -s http://127.0.0.1:6060/debug/pprof/allocs > allocs.out

# mutex / block profile：需要服务内开启采样率
curl -s http://127.0.0.1:6060/debug/pprof/mutex > mutex.out
curl -s http://127.0.0.1:6060/debug/pprof/block > block.out
```

分析：

```bash
go tool pprof -http=:0 cpu.out
go tool pprof -http=:0 -alloc_space allocs.out
go tool pprof -http=:0 -inuse_space heap.out
go tool pprof -http=:0 mutex.out
go tool pprof -http=:0 block.out
```

> 注意：`block` 和 `mutex` profile 需要程序设置采样率，例如 `runtime.SetBlockProfileRate(1)`、`runtime.SetMutexProfileFraction(5)`。生产中可通过 debug endpoint 动态打开，避免长期高开销。

---

## 3. pprof 选择指南

| 要回答的问题 | Profile | 重点看 |
|---|---|---|
| CPU 时间花在哪里？ | CPU | `top`, `list`, flat/cum |
| 哪些代码制造了最多分配？ | allocs | `-alloc_space`, `runtime.mallocgc`, JSON/log/string |
| 哪些对象还活着？ | heap | `-inuse_space`, cache、全局 map、队列积压 |
| goroutine 卡在哪里？ | goroutine | 栈签名聚类、chan send/recv、DB conn 等待 |
| 哪里在等 channel/cond/select？ | block | 阻塞时间、具体代码行 |
| 哪里锁竞争？ | mutex | mutex wait、临界区过大 |
| 调度/网络/同步时间线如何？ | trace | goroutine analysis、network/sync blocking、scheduler latency |

---

## 4. 常见场景排查路径

### 4.1 CPU 高

```text
确认是否全实例 CPU 高
→ 采集 30s CPU profile
→ 看 top flat/cum
→ 若 runtime.mallocgc 高，转 allocation 排查
→ 若业务函数高，写 micro benchmark 复现
→ 优化后 benchstat + 线上指标验证
```

命令：

```bash
go tool pprof cpu.out
(pprof) top
(pprof) list FunctionName
```

判断：`flat` 高是函数自身耗 CPU；`cum` 高说明下游调用耗 CPU。

### 4.2 延迟高但 CPU 低

```text
看 RED 指标确认慢 endpoint/job kind
→ goroutine dump 聚类
→ block/mutex profile
→ DB pool / queue / downstream latency 指标
→ 找到等待点后再修
```

典型证据：

```text
大量 goroutine at database/sql.(*DB).conn     => DB pool 等待
大量 goroutine at chan send worker.go:83      => 下游消费者缺失或 channel 背压
mutex profile 指向 cache.Get                  => 锁竞争
block profile 指向 WaitGroup.Wait             => 生命周期协议错误
```

### 4.3 内存上涨

```text
先分清 RSS、heap inuse、alloc_space
→ 看 gctrace after-GC 是否持续上涨
→ heap profile 用 inuse_space 找活对象
→ alloc profile 用 alloc_space 找分配速率
→ 检查 cache/queue/map/slice 是否无上限
```

命令：

```bash
GODEBUG=gctrace=1 ./server
go tool pprof -http=:0 -inuse_space heap.out
go tool pprof -http=:0 -alloc_space allocs.out
```

判断：`alloc_space` 高但 `inuse_space` 低是分配压力；`inuse_space` 持续上涨更像长期持有或泄漏。

### 4.4 Goroutine 泄漏

```text
采集两次 goroutine dump，间隔 1-5 分钟
→ 对比同一栈签名数量是否增长
→ 找阻塞点：chan send/recv、select、DB conn、HTTP client
→ 检查是否缺 ctx.Done、close、timeout、response body close
```

常见根因：

- channel send 没有取消分支。
- worker 只监听 jobs，不监听 context。
- HTTP client 没 timeout。
- `resp.Body` 未关闭导致连接泄漏。
- ticker/timer 没 stop。

### 4.5 Data race / 业务竞态

```text
进程内共享内存问题 → go test -race
跨 goroutine 业务时序问题 → 状态机/DB 条件更新/幂等表
跨服务重复执行问题 → idempotency key/outbox/consumer dedupe
```

命令：

```bash
go test -race ./...
go test -race -run TestName -count=100 -shuffle=on ./...
```

记住：`-race` 通过不代表没有业务竞态。

### 4.6 Queue backlog

```text
queue_depth 上升
→ worker_active 是否下降？
→ claim latency 是否上升？
→ handler duration 是否上升？
→ DB pool 是否耗尽？
→ 下游 error/timeout 是否上升？
→ DLQ/retry 是否爆炸？
```

Production Job Runner 必备指标：

```text
job_queue_depth{queue}
job_claim_duration_seconds{status}
job_execution_duration_seconds{kind,status}
job_retry_total{kind,reason}
worker_active{group}
worker_blocked_total{reason}
```

---

## 5. Trace 什么时候用

`go tool trace` 适合在 pprof 仍不能解释时使用，尤其是调度、网络、同步等待混在一起时。

```bash
go test -run TestX -trace trace.out ./...
go tool trace trace.out
```

重点页面：

- Goroutine analysis：哪个 goroutine 阻塞最久。
- Synchronization blocking：同步等待在哪里。
- Network blocking：是否卡网络。
- Scheduler latency：是否调度延迟异常。

不要一上来就 trace；trace 信息密度高，适合第二阶段深挖。

---

## 6. 日志字段：排查必须能串起来

最低字段：

```text
ts, level, service, version, request_id, trace_id, job_id,
component, operation, duration_ms, status, error_kind, retry, worker_id
```

规则：

- `job_id` 不放 Prometheus label，放日志和 trace。
- error 要分类：`validation`, `timeout`, `dependency`, `conflict`, `internal`。
- 重要状态转换要打结构化事件：`pending -> running -> succeeded/failed/retrying`。

---

## 7. 复盘模板

```text
问题：
影响面：
开始时间：
触发因素：发布/配置/流量/依赖？
用户可见症状：
核心指标变化：
关键证据：profile/log/trace/DB record
根因：
为什么监控没有更早发现：
修复：
回归测试：
新增指标/告警：
预防措施：
```

---

## 8. Production Job Runner 专用排查 Checklist

- 一个 `job_id` 能否查到 API submit、DB state、worker claim、execution、result、event publish 的完整链路？
- worker crash 后 running job 如何恢复？有没有 stale lock 扫描？
- retry 是否有 backoff + jitter？是否有最大次数？
- result payload 是否过大？是否外置化？
- DB claim 是否是条件更新？是否可能重复执行？
- 队列积压时是生产太快、消费太慢，还是消费者卡住？
- 每个外部调用是否有 timeout？
- 每个阻塞 channel send 是否有 `ctx.Done()`？
