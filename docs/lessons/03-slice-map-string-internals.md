# Lesson 3：Slice、Map、String 底层结构

## 学习目标

完成本课后，学习者应该能够：

- 解释 slice header 的三个字段以及它们和底层数组的关系。
- 判断 `append` 是否会复用原底层数组，以及这对调用方可见性的影响。
- 识别 subslice 导致大数组被长期持有的内存保留问题。
- 区分 nil slice 和 empty slice 在语义、JSON 输出、API contract 中的差异。
- 解释 map 为什么无序、为什么不能并发读写、为什么 key 必须可比较。
- 理解 string 的不可变性、byte/rune 差异，以及字符串拼接的常见性能取舍。
- 在生产代码中选择合适的数据结构、同步策略和内存释放方式。

---

## 关键问题

1. slice 变量本身保存数据吗？
2. `append` 什么时候修改原底层数组，什么时候分配新数组？
3. 为什么一个很小的 subslice 可能让一个大 buffer 无法被 GC？
4. nil slice 和 empty slice 有什么区别？
5. Go map 为什么遍历顺序不稳定？
6. Go map 为什么不是并发安全的？
7. string 为什么不可变？`len(s)` 返回的是字符数还是字节数？
8. 字符串拼接应该用 `+`、`fmt.Sprintf`、`strings.Builder` 还是 `bytes.Buffer`？

---

## 核心结论

- slice 是一个 descriptor/header，不直接存储元素数据。
- slice header 包含指向底层数组的指针、长度和容量。
- `append` 的结果必须接住，因为它可能返回新的 header。
- subslice 会继续引用原底层数组；从大 buffer 中截取小片段长期保存时应考虑 `copy`。
- nil slice 和 empty slice 都可 `range`、可 `append`，但 JSON 表达不同。
- map 的遍历顺序故意不稳定，不能依赖顺序。
- map 并发读写不安全；生产中通常使用 `map + sync.RWMutex`，不要默认 `sync.Map`。
- string 是不可变字节序列；处理 Unicode 时要区分 byte 和 rune。
- 性能热点中的字符串拼接优先 benchmark，通常 `strings.Builder` 适合构建 string。

---

## 设计哲学

Go 的 slice、map、string 都体现了一个共同设计：

```text
语言层提供简单模型，runtime 层承担复杂实现，程序员通过少量规则写出可预测代码。
```

它们都很容易使用，但生产环境下的坑也很集中：

- slice 的共享底层数组会带来意外修改和内存保留。
- map 的高性能实现不提供并发安全。
- string 的不可变性让传递和共享更安全，但转换和拼接可能产生分配。

本课不是背 runtime 细节，而是建立可操作的判断：

```text
这个值是否共享底层存储？
这个操作是否可能重新分配？
这个结构是否能并发访问？
这个 API 的 nil/empty 语义是否稳定？
```

---

## 1. Slice 底层结构

slice 可以简化理解为：

```go
type sliceHeader struct {
    data *T
    len  int
    cap  int
}
```

注意：这只是教学模型。生产代码不要用自定义 header + unsafe 手动操作 slice。

### 1.1 len 和 cap

```go
xs := make([]int, 3, 5)
fmt.Println(len(xs)) // 3
fmt.Println(cap(xs)) // 5
```

- `len`：当前可访问元素数量。
- `cap`：从 `data` 指向的位置开始，到底层数组末尾还能容纳多少元素。

### 1.2 slice header 会被复制

```go
func setFirst(xs []int) {
    xs[0] = 100
}
```

传参复制的是 header，但两个 header 指向同一个底层数组，所以元素修改对调用方可见。

```go
func appendLocal(xs []int) {
    xs = append(xs, 4)
}
```

这里修改的是局部 header，调用方看不到新的 `len`，除非返回：

```go
xs = appendLocalReturn(xs)
```

---

## 2. append 与扩容

`append` 有两种情况。

### 2.1 容量足够：复用底层数组

```go
xs := make([]int, 2, 4)
ys := append(xs, 3)
```

此时 `ys` 通常和 `xs` 共享底层数组。

这意味着：

```go
ys[0] = 100
fmt.Println(xs[0]) // 100
```

### 2.2 容量不足：分配新底层数组

```go
xs := make([]int, 2, 2)
ys := append(xs, 3)
```

容量不够时，runtime 会分配新数组并复制旧元素。此时 `ys` 和 `xs` 不再共享底层数组。

### 2.3 生产建议

- 永远接住 `append` 返回值。
- 不要假设 append 后是否共享底层数组。
- 如果要避免后续 append 修改原数组，可以使用 full slice expression 限制容量：

```go
safe := xs[:len(xs):len(xs)]
safe = append(safe, 1) // 强制分配新数组
```

---

## 3. Subslice 内存保留问题

### 3.1 问题示例

```go
func firstKB(buf []byte) []byte {
    return buf[:1024]
}
```

如果 `buf` 是 100MB，返回的 1KB slice 仍然引用原来的 100MB 底层数组。只要这个小 slice 还活着，大数组就不能被 GC。

### 3.2 修复方式

```go
func firstKBCopy(buf []byte) []byte {
    out := make([]byte, 1024)
    copy(out, buf[:1024])
    return out
}
```

或者 Go 1.20+：

```go
out := bytes.Clone(buf[:1024])
```

### 3.3 生产场景

常见于：

- 读取大文件后截取 header。
- 网络 buffer 中截取 token。
- 大 JSON payload 中截取字段。
- 日志、消息队列、缓存中长期保存小片段。

---

## 4. nil slice vs empty slice

```go
var a []int        // nil slice
b := []int{}       // empty slice
c := make([]int,0) // empty slice
```

共同点：

```go
len(a) == 0
len(b) == 0
range 正常
append 正常
```

区别：

```go
a == nil // true
b == nil // false
```

JSON 输出：

```go
json.Marshal(a) // null
json.Marshal(b) // []
```

生产建议：

- 内部逻辑通常可以接受 nil slice。
- 对外 API 如果 contract 要求数组，通常返回 empty slice，避免 `null`。
- 不要在业务逻辑中无意义地区分 nil 和 empty，除非 API 语义确实不同。

---

## 5. Map 底层与使用边界

Go map 是 hash table。可以简化理解为：

```text
hash(key) → bucket → key/value slots → overflow bucket
```

runtime 会根据负载情况扩容和搬迁 bucket。

### 5.1 map key 必须 comparable

合法 key：

```go
map[string]int
map[int]string
map[struct{ A int; B string }]int
```

非法 key：

```go
map[[]byte]int
map[map[string]string]int
map[func()]int
```

因为 slice、map、function 不可比较。

### 5.2 遍历顺序不稳定

```go
for k := range m {
    fmt.Println(k)
}
```

不要依赖遍历顺序。需要稳定输出时，先取 key，再排序：

```go
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
```

Go 故意让 map iteration 顺序不稳定，防止程序依赖实现细节。

### 5.3 map 非并发安全

并发读写 map 可能 panic：

```text
fatal error: concurrent map read and map write
```

更重要的是，即使没有 panic，也不代表正确。

常见选择：

| 场景 | 推荐 |
|---|---|
| 普通共享 map | `map + sync.RWMutex` |
| 读多写少、整体替换配置 | `atomic.Value` |
| key 稳定、读多写少、缓存类场景 | 可考虑 `sync.Map` |
| 需要过期、容量、淘汰 | 专用 cache 库或自己加策略 |

不要默认使用 `sync.Map`。它为特定模式优化，不是通用 map 替代品。

---

## 6. String 底层与 Unicode

string 是不可变字节序列。可以简化理解为：

```go
type stringHeader struct {
    data *byte
    len  int
}
```

### 6.1 len 返回字节数

```go
s := "你好"
fmt.Println(len(s)) // 6，UTF-8 下每个汉字 3 bytes
```

如果要按 Unicode code point 遍历：

```go
for i, r := range s {
    fmt.Println(i, r)
}
```

`i` 是 byte offset，`r` 是 rune。

### 6.2 string 不可变

不可变的好处：

- 可以安全共享。
- map key 稳定。
- API 更容易推理。

代价：

- 修改字符串需要创建新字符串。
- `[]byte` 与 `string` 转换通常会复制。

### 6.3 字符串拼接

小规模、编译期可优化的拼接：

```go
s := "hello" + name
```

循环中拼接：

```go
var b strings.Builder
for _, p := range parts {
    b.WriteString(p)
}
s := b.String()
```

`fmt.Sprintf` 可读性好，但通常更慢、分配更多，适合格式化而不是热点拼接。

---

## 代码实验

配套实验目录：

```text
labs/03-slice-map-string/
```

实验覆盖：

1. append 容量足够时共享底层数组。
2. append 容量不足时分配新数组。
3. full slice expression 限制 cap，避免后续 append 影响原数组。
4. subslice 内存保留与 copy 修复。
5. nil slice 和 empty slice 的 JSON 输出差异。
6. map 稳定输出需要排序 key。
7. string 的 byte/rune 差异。
8. `+`、`fmt.Sprintf`、`strings.Builder` 拼接 benchmark。

运行：

```bash
cd labs/03-slice-map-string
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

---

## 生产实践

### Slice

- 返回 append 结果，不要忽略。
- 长期保存小切片前确认是否持有大底层数组。
- 对外 API 需要稳定数组语义时返回 empty slice 而不是 nil slice。
- 在并发场景中，不要多个 goroutine 无同步修改同一个 slice 或其底层数组。

### Map

- 不依赖遍历顺序。
- 并发访问必须同步。
- 需要稳定序列化输出时排序 key。
- 不要默认 `sync.Map`，先考虑 `map + RWMutex`。

### String

- 明确 byte、rune、grapheme cluster 的差异。
- 热点路径避免无意义 `fmt.Sprintf`。
- 大量拼接优先 `strings.Builder`。
- 不要频繁在 `string` 和 `[]byte` 之间来回转换。

---

## 常见误区

### 误区 1：slice 是引用传递

slice 传参复制 header。底层数组共享，但 header 本身不是共享的。

### 误区 2：append 会原地修改

append 可能原地，也可能分配新数组。必须使用返回值。

### 误区 3：小 slice 不占内存

小 slice 可能持有大数组，关键看底层数组生命周期。

### 误区 4：map 遍历顺序只是随机但稳定

不能依赖任何顺序。需要顺序就排序。

### 误区 5：sync.Map 比 map+mutex 更高级

`sync.Map` 是特定场景优化，不是默认选择。

### 误区 6：len(string) 是字符数

`len(string)` 是字节数。

---

## 故障案例

### 案例 1：内存持续增长但 heap profile 显示大对象来源不明显

现象：

```text
服务从大 payload 中截取 token 保存到缓存。
缓存里每个 token 只有几十字节，但 heap 持续增长。
```

原因：

```text
token slice 仍然引用整个 payload 的底层数组。
```

修复：

```go
token = bytes.Clone(token)
```

或：

```go
copyToken := append([]byte(nil), token...)
```

### 案例 2：线上偶发 concurrent map read and map write

现象：

```text
fatal error: concurrent map read and map write
```

原因：

```text
多个 goroutine 读写普通 map，没有同步。
```

修复：

- 使用 `sync.RWMutex` 保护 map。
- 或改为单 goroutine ownership + channel。
- 或按读多写少场景使用 `atomic.Value` 发布不可变快照。

### 案例 3：中文字符串截断乱码

现象：

```go
s := "你好世界"
fmt.Println(s[:5]) // 可能截断 UTF-8 字节序列
```

修复：

- 按 rune 截断。
- 或使用专门处理 grapheme cluster 的库。

---

## 作业

1. 写一个函数从大 `[]byte` 中截取前 16 字节并长期保存，分别实现共享版本和 copy 版本。
2. 用测试证明 append 容量足够和容量不足时的共享差异。
3. 写一个 map 输出函数，保证输出 key 按字典序稳定。
4. 写 benchmark 比较循环中 `+`、`fmt.Sprintf`、`strings.Builder` 拼接。
5. 找一个真实 API，判断它应该返回 nil slice 还是 empty slice，并说明原因。

---

## 评估标准

- 能画出 slice header 与底层数组关系。
- 能解释 append 共享和扩容的可见性差异。
- 能识别 subslice 内存保留风险并修复。
- 能解释 map 无序和并发不安全的原因。
- 能处理 string 的 byte/rune 差异。
- 能用 benchmark 支持字符串拼接选择。

---

## 本节不展开

- `runtime.growslice` 源码逐行阅读。
- map evacuation 的完整源码细节。
- Unicode grapheme cluster 的完整文本处理。
- unsafe 零拷贝 string/[]byte 转换。

这些内容可以作为拓展阅读或专项性能优化课题。

---

## 延伸阅读

- Go Blog: Go Slices: usage and internals
- Go Blog: Strings, bytes, runes and characters in Go
- Go Specification: Map types
- Go Code Review Comments
- 100 Go Mistakes and How to Avoid Them
