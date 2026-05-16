# Lesson 9：sync、atomic 与 Go Memory Model

## 学习目标

完成本课后，学习者应该能够：

- 解释 data race 与 race condition 的区别。
- 使用 `go test -race` 发现数据竞争。
- 正确使用 `sync.Mutex`、`sync.RWMutex`、`sync.Once`、`sync.WaitGroup`、`sync.Cond` 的基本场景。
- 理解 `sync/atomic` 的适用边界：简单计数、状态标记、不可变快照发布。
- 解释 happens-before 的核心意义。
- 判断什么时候应该用 mutex，什么时候可以用 atomic，什么时候应该避免共享状态。
- 识别常见并发错误：复制锁、WaitGroup 误用、RWMutex 滥用、atomic 与普通读写混用。

---

## 关键问题

1. 什么是 data race？它和业务层 race condition 有什么不同？
2. 为什么没有 data race 的程序仍可能有逻辑竞态？
3. `Mutex` 保护的到底是什么？代码块还是共享状态？
4. `RWMutex` 一定比 `Mutex` 快吗？
5. `atomic` 是不是比锁更高级、更快？
6. `WaitGroup.Add` 为什么通常要在启动 goroutine 前调用？
7. `sync.Once` 适合哪些初始化场景？
8. Go memory model 里的 happens-before 对我们写业务代码有什么意义？

---

## 核心结论

- data race 是多个 goroutine 并发访问同一变量，至少一个写，且没有同步。
- Go 程序只要存在 data race，行为就不可靠；先用 race detector 消除 data race。
- `Mutex` 是保护共享状态的最常用工具，语义清晰通常比过早 atomic 更好。
- `RWMutex` 只在读多写少且临界区足够大时可能收益；写多或临界区很短时可能更慢。
- `atomic` 适合非常小的同步问题，不适合维护复杂不变量。
- happens-before 描述一个操作的结果对另一个操作可见的顺序保证。
- channel send/receive、mutex unlock/lock、atomic 操作、goroutine start、WaitGroup 等都可建立同步关系。
- 并发正确性的优先级：清晰所有权 > 简单同步 > race detector 验证 > benchmark 优化。

---

## 设计哲学

Go 的并发工具并不追求“魔法自动安全”。语言允许共享内存，也提供 channel、mutex、atomic 等工具，但正确性仍需要设计。

实用原则：

```text
能不共享就不共享；必须共享就明确谁保护；需要优化时再考虑 atomic。
```

不要把 atomic 当作“无锁万能药”。它能解决的问题非常窄：单个变量的原子读写、计数、标志、不可变指针发布。只要涉及多个字段之间的一致性，mutex 通常更合适。

---

## 1. Data Race vs Race Condition

### Data race

```go
var n int

go func() { n++ }()
go func() { n++ }()
```

多个 goroutine 访问同一变量，至少一个写，没有同步，就是 data race。

用 race detector：

```bash
go test -race ./...
```

### Race condition

没有 data race，也可能有逻辑竞态。

例如两个请求都先查余额再扣款，如果事务边界不正确，即使用了锁保护内存状态，也可能在数据库层出现业务竞态。

```text
data race 是内存访问层面的错误；
race condition 是系统状态转移层面的错误。
```

---

## 2. Mutex

`sync.Mutex` 保护共享状态：

```go
type Counter struct {
    mu sync.Mutex
    n  int
}

func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.n++
}
```

注意：锁不是保护代码，而是保护某个不变量。

正确问题应该是：

```text
哪些字段由这个 mutex 保护？
所有读写是否都持有同一把锁？
锁内是否调用外部未知代码？
锁的粒度是否合理？
```

### 不要复制锁

包含 mutex 的 struct 一旦使用后不应复制。

```go
c2 := c1 // 可能复制锁状态，错误
```

可以用 pointer receiver，或避免复制包含锁的对象。

---

## 3. RWMutex

`sync.RWMutex` 允许多个 reader 同时持有读锁，但 writer 需要独占。

适合：

- 读远多于写。
- 读临界区相对较重。
- 写不频繁。

不适合：

- 写多。
- 临界区很短。
- 读锁内仍做阻塞 IO。
- 只是为了“看起来更高级”。

需要 benchmark 验证：

```bash
go test -bench=. -benchmem ./...
```

---

## 4. WaitGroup

`sync.WaitGroup` 用于等待一组 goroutine 完成。

正确模式：

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)
}
wg.Wait()
```

关键点：

- `Add` 通常在启动 goroutine 之前调用。
- 每个 `Add(1)` 必须对应一个 `Done()`。
- 不要复制 WaitGroup。
- 不要在上一轮 `Wait` 尚未结束时混乱复用。

---

## 5. Once

`sync.Once` 保证某个函数只执行一次：

```go
var once sync.Once
once.Do(initConfig)
```

适合：

- lazy initialization。
- 初始化全局资源。
- 缓存只初始化一次。

注意：如果 `Do` 中函数 panic，Once 会认为已经执行过。后续不会重试。

---

## 6. Cond

`sync.Cond` 用于复杂条件等待。

日常业务中不如 channel 常见，但适合多个 goroutine 等待某个状态条件变化。

```go
cond.L.Lock()
for !ready {
    cond.Wait()
}
cond.L.Unlock()
```

必须用 for 循环检查条件，因为被唤醒不代表条件一定满足。

---

## 7. Atomic

Go 1.19+ 推荐使用 typed atomic：

```go
var n atomic.Int64
n.Add(1)
fmt.Println(n.Load())
```

适合：

- counters。
- boolean flags。
- 状态枚举。
- immutable snapshot pointer。

不适合：

- 多字段一致性。
- 需要复合操作的不变量。
- 读写混用普通变量与 atomic。
- 过早性能优化。

### atomic.Value

`atomic.Value` 可用于发布不可变配置快照：

```go
var v atomic.Value
v.Store(Config{...})
cfg := v.Load().(Config)
```

发布后不要修改快照内部可变对象，否则仍可能 data race。

---

## 8. Go Memory Model 与 happens-before

happens-before 是一种可见性保证：如果 A happens-before B，那么 A 的写入对 B 可见。

常见同步关系：

- `Mutex.Unlock` happens-before 后续成功的 `Mutex.Lock`。
- channel send happens-before 对应 receive。
- close channel happens-before 接收到关闭信号。
- goroutine start 前的写入对 goroutine 可见。
- atomic 操作之间有顺序一致性保证。

没有 happens-before，就不能假设另一个 goroutine 一定能看到你的写入。

---

## 9. 生产实践

### 9.1 并发设计 checklist

```text
共享状态是什么？
谁拥有状态？
使用什么同步原语保护？
所有读写是否都走同一路径？
是否需要支持取消？
是否需要超时？
是否有 benchmark？
是否跑过 race detector？
```

### 9.2 选择同步原语

| 场景 | 推荐 |
|---|---|
| 简单共享 map/cache | Mutex/RWMutex |
| 任务分发 | channel |
| 计数指标 | atomic.Int64 |
| lazy init | sync.Once |
| 等待一组任务 | sync.WaitGroup |
| 发布不可变配置 | atomic.Value |
| 复杂条件等待 | sync.Cond |

---

## 代码实验

配套实验目录：

```text
labs/09-sync-atomic-memory-model/
```

实验覆盖：

1. race detector 识别不安全计数器。
2. MutexCounter 修复 data race。
3. AtomicCounter 修复简单计数。
4. RWMutex map cache。
5. sync.Once lazy initialization。
6. WaitGroup 正确等待任务完成。
7. atomic.Value 发布不可变配置快照。
8. channel send/receive 建立 happens-before。
9. Mutex vs Atomic vs Channel counter benchmark。

运行：

```bash
cd labs/09-sync-atomic-memory-model
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

---

## 常见误区

### 误区 1：只要测试通过就没有并发 bug

并发 bug 具有时序性，普通测试不一定覆盖。必须使用 race detector、压力测试和设计审查。

### 误区 2：atomic 一定比 mutex 好

atomic 只能解决非常窄的问题。复杂不变量用 mutex 更安全。

### 误区 3：RWMutex 一定比 Mutex 快

短临界区、写多场景下 RWMutex 可能更慢。

### 误区 4：锁住代码块就安全了

锁保护的是状态。所有访问该状态的路径都必须遵守同一把锁。

### 误区 5：atomic 读，普通写也可以

不可以。对同一变量混用 atomic 与非 atomic 访问仍可能 data race。

---

## 故障案例

### 案例 1：内存 cache 偶发 panic

原因：map 并发读写。

修复：用 RWMutex 保护 map，或使用 copy-on-write + atomic.Value 发布不可变快照。

### 案例 2：配置热更新后部分请求看到半更新状态

原因：多个字段分别 atomic 更新，读取方看到新旧混合。

修复：构造完整不可变 Config，一次性用 atomic.Value 发布。

### 案例 3：WaitGroup 计数偶发负数 panic

原因：goroutine 内部调用 Add，主 goroutine 可能已经 Wait 或 Done 次序混乱。

修复：启动 goroutine 前 Add。

---

## 作业

1. 在 Lab 9 中故意引入一个 data race，用 `go test -race` 捕获，然后修复。
2. 为 map cache 增加 `Delete`，确保所有路径都持有锁。
3. 用 atomic.Value 发布配置快照，证明旧快照不会被后续修改影响。
4. 对比 MutexCounter、AtomicCounter、ChannelCounter benchmark。
5. 写一段说明：你的 Production Job Runner 中哪些状态需要 mutex，哪些指标用 atomic。

---

## 评估标准

- 能区分 data race 和 race condition。
- 能使用 race detector 定位数据竞争。
- 能正确使用 Mutex/RWMutex/WaitGroup/Once。
- 能说明 atomic 的适用边界。
- 能解释 happens-before 的业务意义。
- 能通过 benchmark 判断同步原语选择是否合理。

---

## 本节不展开

- Go memory model 形式化证明。
- lock-free 数据结构设计。
- CPU cache coherence 与 memory barrier 底层细节。
- 分布式系统中的线性一致性。

---

## 延伸阅读

- The Go Memory Model
- Go Blog: Data Race Detector
- `sync` package documentation
- `sync/atomic` package documentation
- 100 Go Mistakes and How to Avoid Them
