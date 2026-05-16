# Lesson 2：值、指针与逃逸分析

## 学习目标

完成本课后，学习者应该能够：

- 准确解释 Go 的参数传递模型：**所有参数传递都是值传递**。
- 区分值语义、指针语义、引用语义这几个容易混淆的概念。
- 解释 slice、map、channel、function、interface 作为参数传递时到底复制了什么。
- 使用 `go build -gcflags="-m -m"` 阅读逃逸分析输出。
- 解释常见逃逸来源：返回局部变量指针、interface boxing、闭包捕获、goroutine 捕获、动态大小分配。
- 使用 benchmark 和 `benchmem` 比较值传递、指针传递、interface 传参的分配与性能差异。
- 在生产代码中合理选择值还是指针，而不是凭“指针更快”的直觉写代码。

---

## 关键问题

1. Go 到底是值传递还是引用传递？
2. `slice`、`map`、`channel` 作为函数参数传递时，复制的是什么？
3. 返回局部变量指针为什么是安全的？
4. 变量什么时候分配在栈上，什么时候分配在堆上？
5. `interface{}` 为什么可能导致额外分配？
6. 指针传递一定比值传递更快吗？
7. 逃逸分析输出里的 `escapes to heap`、`moved to heap`、`does not escape` 应该如何理解？
8. 生产代码中应该优先按语义选择值/指针，还是按性能选择？

---

## 核心结论

- Go 的函数参数传递永远是值传递。
- slice、map、channel 看起来像“引用传递”，是因为它们的值本身包含指向底层数据结构的指针。
- 是否使用指针首先是**语义选择**，其次才是性能选择。
- 小的、不可变语义的 struct 通常适合值传递。
- 需要修改调用方可见状态、对象较大、包含锁或不可复制资源时，通常使用指针。
- 逃逸分析由编译器决定变量放在栈还是堆；写了 `&x` 不代表一定慢，写了值传递也不代表一定零分配。
- interface、闭包、goroutine、返回指针、动态大小对象都可能导致逃逸。
- 性能判断必须用 `-gcflags="-m -m"` 和 benchmark 验证，不能只靠经验。

---

## 设计哲学

Go 在值和指针上的设计非常朴素：没有 C++ 那种复杂引用语义，也没有 Java 那种“一切对象引用”的统一模型。

Go 更希望程序员明确表达：

```text
这个值是否会被修改？
这个对象是否有共享身份？
这个对象是否可以安全复制？
这个 API 是否希望调用者感知内部状态变化？
```

这也是为什么 Go 代码里经常能看到两种风格：

```go
func NormalizeEmail(email string) string
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (User, error)
```

前者是纯值转换，后者是带状态的服务对象。

本课的重点不是总结“什么时候用指针”的死规则，而是建立判断框架：

```text
语义优先 → 正确性优先 → 可读性优先 → 最后用工具验证性能
```

---

## 1. Go 的参数传递模型

### 1.1 所有参数都是值传递

看这个例子：

```go
func changeInt(x int) {
    x = 100
}

func main() {
    n := 1
    changeInt(n)
    fmt.Println(n) // 1
}
```

`changeInt` 收到的是 `n` 的副本，所以函数内修改不会影响调用方。

再看指针：

```go
func changeIntByPointer(p *int) {
    *p = 100
}

func main() {
    n := 1
    changeIntByPointer(&n)
    fmt.Println(n) // 100
}
```

这依然是值传递。只不过传递的值是一个地址。函数参数 `p` 是地址值的副本，`*p` 指向调用方的变量。

所以更准确的说法是：

```text
Go 永远按值复制参数。
如果复制的值里包含指针，那么函数可以通过这个指针影响共享底层数据。
```

---

## 2. slice、map、channel 传递时复制了什么

### 2.1 slice

slice 的运行时结构可以简化理解为：

```go
type sliceHeader struct {
    data *T
    len  int
    cap  int
}
```

当你传递一个 slice：

```go
func modify(xs []int) {
    xs[0] = 100
}
```

复制的是 slice header。但 header 里的 `data` 仍然指向同一个底层数组，所以修改元素会影响调用方。

但是如果你在函数里改 slice header 本身：

```go
func appendLocal(xs []int) {
    xs = append(xs, 4)
}
```

调用方不一定看到新的 len，因为函数里修改的是 header 副本。

如果要让调用方看到 append 后的新 header，必须返回：

```go
func appendReturn(xs []int) []int {
    return append(xs, 4)
}
```

### 2.2 map

map 值可以理解为指向 runtime hash table 的 descriptor。

```go
func put(m map[string]int) {
    m["a"] = 1
}
```

传参复制了 map descriptor，但 descriptor 指向同一个底层 hash table，所以写入会影响调用方。

注意：这不等于 map 是并发安全的。map 的底层结构会扩容、搬迁 bucket，并发读写会破坏内部状态，所以 Go 运行时会在部分场景直接 panic：

```text
fatal error: concurrent map read and map write
```

### 2.3 channel

channel 值也是 descriptor，指向 runtime 的 `hchan` 结构。

```go
func send(ch chan<- int) {
    ch <- 1
}
```

复制 channel 值不会复制队列。多个 channel 变量可以指向同一个 `hchan`。

---

## 3. 值语义与指针语义

### 3.1 值语义

值语义表示这个对象可以被安全复制，复制后两个值相互独立。

适合值语义的例子：

```go
type Point struct {
    X, Y int
}

func Move(p Point, dx, dy int) Point {
    p.X += dx
    p.Y += dy
    return p
}
```

这类代码的优点：

- 调用者不用担心函数偷偷修改原值。
- 并发场景下更容易推理。
- 小对象复制成本低。
- API 更接近纯函数。

### 3.2 指针语义

指针语义表示对象有身份、生命周期或共享可变状态。

适合指针语义的例子：

```go
type Counter struct {
    n int64
}

func (c *Counter) Inc() {
    c.n++
}
```

如果使用值接收者：

```go
func (c Counter) Inc() {
    c.n++
}
```

修改的是副本，调用方看不到变化。

### 3.3 不要复制含锁对象

如果 struct 包含 `sync.Mutex`、`sync.WaitGroup`、`atomic` 相关字段，复制通常是 bug。

```go
type SafeCounter struct {
    mu sync.Mutex
    n  int
}
```

这类对象的方法通常应该使用指针接收者：

```go
func (c *SafeCounter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.n++
}
```

复制 mutex 会产生两个锁保护同一逻辑状态或复制锁状态，容易导致严重并发问题。

---

## 4. 栈、堆与逃逸分析

### 4.1 栈和堆的直觉模型

栈适合函数局部、生命周期明确的对象。分配和回收成本低，通常随函数调用自动管理。

堆适合生命周期超出当前函数、大小动态或编译器无法证明安全的对象。堆对象由 GC 管理，会带来分配和扫描成本。

但 Go 程序员不直接决定变量在栈还是堆。编译器通过 escape analysis 判断。

### 4.2 返回局部变量指针为什么安全

```go
func NewPoint() *Point {
    p := Point{X: 1, Y: 2}
    return &p
}
```

在 C 里返回局部变量地址是悬垂指针。但 Go 编译器会发现 `p` 的生命周期逃出了函数，于是把它分配到堆上。

逃逸分析可能输出：

```text
p escapes to heap
moved to heap: p
```

这说明它是安全的，但可能带来堆分配和 GC 成本。

---

## 5. 常见逃逸来源

### 5.1 返回局部变量指针

```go
func ReturnPointer() *Small {
    x := Small{A: 1, B: 2}
    return &x
}
```

`x` 必须在函数返回后继续存在，所以会逃逸。

### 5.2 interface boxing

```go
func ToAny(x Small) any {
    return x
}
```

当具体值被装箱进 interface 时，编译器可能需要把它放到堆上，尤其是返回 interface 或跨越不明确边界时。

不是所有 interface 都一定逃逸，但 interface 会增加编译器证明难度。

### 5.3 闭包捕获

```go
func MakeCounter() func() int {
    n := 0
    return func() int {
        n++
        return n
    }
}
```

`n` 被返回的闭包捕获，生命周期超过当前函数，因此通常会逃逸。

### 5.4 goroutine 捕获

```go
func Start() {
    x := 1
    go func() {
        fmt.Println(x)
    }()
}
```

goroutine 可能在 `Start` 返回后运行，捕获变量需要延长生命周期。

### 5.5 动态大小分配

```go
func MakeBytes(n int) []byte {
    return make([]byte, n)
}
```

动态大小对象是否逃逸取决于使用方式、大小和编译器能否证明生命周期。

---

## 6. 如何阅读逃逸分析输出

命令：

```bash
go build -gcflags="-m -m" ./...
```

常见输出：

```text
can inline Foo
inlining call to Foo
x escapes to heap
moved to heap: x
... does not escape
```

重点关注：

| 输出 | 含义 |
|---|---|
| `escapes to heap` | 值生命周期或使用方式导致堆分配 |
| `moved to heap` | 局部变量被移动到堆上 |
| `does not escape` | 编译器证明该值不需要逃逸 |
| `can inline` | 函数可内联，可能改变逃逸结果 |

注意：逃逸分析输出不是稳定 API，不同 Go 版本可能略有不同。课程中关注现象和判断方法，不要求记住每一行输出。

---

## 7. 指针一定更快吗？

不一定。

### 7.1 指针可能减少复制

对于很大的 struct，复制成本可能明显：

```go
type Large struct {
    Data [1024]int64
}
```

传指针可以避免大对象复制。

### 7.2 指针也可能更慢

指针可能带来：

- 堆分配。
- GC 扫描成本。
- cache locality 变差。
- 共享可变状态导致锁竞争。
- API 更难推理。

小 struct 值传递经常更快、更清晰。

例如：

```go
type Money struct {
    Amount   int64
    Currency string
}
```

如果它是不可变值对象，值传递通常比到处传 `*Money` 更合理。

---

## 8. 生产实践判断框架

### 8.1 优先使用值的场景

- 小 struct。
- 不需要修改调用方状态。
- 表达不可变值对象。
- 并发读多、希望减少共享状态。
- DTO、配置快照、时间点、坐标、金额等值对象。

### 8.2 优先使用指针的场景

- 需要修改接收者状态。
- struct 较大，复制成本高。
- 包含 `sync.Mutex`、`sync.WaitGroup` 等不可复制字段。
- 需要表达对象身份，例如 service、repository、connection、cache。
- 方法集需要满足某个 interface，且方法使用指针接收者。

### 8.3 receiver 选择

同一个类型的方法接收者尽量保持一致。

如果某些方法必须使用指针接收者，通常整个类型都使用指针接收者，避免方法集和调用语义混乱。

### 8.4 API 设计建议

```go
// 好：请求是值对象，返回领域值和错误
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (User, error)

// 谨慎：到处传指针可能让调用方担心被修改
func (s *UserService) Create(ctx context.Context, req *CreateUserRequest) (*User, error)
```

除非请求体很大、需要区分 nil、或确实要修改请求，否则 request DTO 通常用值更清晰。

---

## 代码实验

配套实验目录：

```text
labs/02-escape-analysis/
```

实验覆盖：

1. 值传递不会修改调用方变量。
2. 指针传递可以修改调用方变量。
3. slice header 复制但底层数组共享。
4. append 后必须返回新 slice header。
5. 返回局部变量指针触发逃逸。
6. interface boxing 可能触发逃逸。
7. 闭包捕获变量触发逃逸。
8. benchmark 比较小对象值传递、大对象指针传递和 interface 装箱。

运行：

```bash
cd labs/02-escape-analysis
go test ./...
go test -bench=. -benchmem ./...
go build -gcflags="-m -m" ./...
```

---

## 常见误区

### 误区 1：Go 有引用传递

Go 没有引用传递。slice/map/channel 只是值里包含指针。

### 误区 2：使用指针一定更快

指针可能减少复制，也可能导致堆分配、GC 压力和 cache miss。

### 误区 3：返回局部变量指针不安全

Go 中这是安全的，因为编译器会让变量逃逸到堆上。

### 误区 4：看到 escape 就必须优化

不一定。逃逸本身不是 bug。只有当 profile 显示分配成为瓶颈时，才需要优化。

### 误区 5：为了减少复制，把所有参数都改成指针

这会增加共享可变状态，降低 API 可读性，并可能引入并发问题。

---

## 故障案例

### 案例：高 QPS API 的 allocation rate 异常

现象：

```text
P99 延迟升高
GC 次数增加
CPU 中 GC 占比上升
heap allocation rate 明显高于预期
```

排查路径：

1. 查看 metrics：heap、alloc rate、GC pause。
2. 使用 pprof heap profile 找 allocation hotspot。
3. 使用 `go test -bench=. -benchmem` 复现热点路径。
4. 使用 `go build -gcflags="-m -m"` 查看是否有 interface boxing 或闭包捕获导致逃逸。
5. 优化后再次 benchmark 和 profile 对比。

可能修复：

- 避免在热点路径把小对象装箱到 `any`。
- 减少不必要的 `fmt.Sprintf`。
- 重用 buffer，但要注意生命周期和并发安全。
- 把纯值对象保持为值传递，避免无意义指针导致堆分配。

---

## 作业

1. 在 `labs/02-escape-analysis` 中新增一个函数，让它因为 goroutine 捕获变量而逃逸。
2. 运行 `go build -gcflags="-m -m" ./...`，截取相关输出并解释。
3. 写一个 benchmark，对比 `Small` 值传递和 `*Small` 指针传递。
4. 写一个 benchmark，对比 `Small` 返回具体类型和返回 `any`。
5. 从一个已有 Go 项目中找一个“无脑传指针”的 API，分析它是否真的需要指针。

---

## 评估标准

- 能准确说明 Go 是值传递。
- 能画出 slice header 复制与底层数组共享的关系。
- 能解释返回局部变量指针为什么安全以及代价是什么。
- 能读懂至少三类逃逸分析输出。
- 能用 benchmark 数据支持值/指针选择。
- 能在 API 设计中按语义而不是迷信性能选择值或指针。

---

## 本节不展开

- Go runtime 栈增长细节。
- GC write barrier 的完整机制。
- escape analysis 编译器源码。
- unsafe 手动构造 slice/string header。

这些内容会在后续 runtime、GC、slice/string 课程中继续展开。

---

## 延伸阅读

- Go Blog: Go Slices: usage and internals
- Go Blog: Profiling Go Programs
- Go FAQ: Why do T and *T have different method sets?
- Go Code Review Comments: Receiver Type
- 100 Go Mistakes and How to Avoid Them
- `go help buildconstraint`
- `go tool compile -help`
