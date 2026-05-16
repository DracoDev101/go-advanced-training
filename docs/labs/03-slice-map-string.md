# Lab 3：Slice、Map、String 底层结构

本实验配套 Lesson 3，用可运行代码观察 slice 共享底层数组、append 扩容、subslice copy、nil/empty slice、map 稳定输出、string byte/rune 以及字符串拼接性能。

## 观察目标

1. append 容量足够时，新旧 slice 共享底层数组。
2. append 容量不足时，会分配新底层数组。
3. full slice expression 可以限制 cap，强制下一次 append 分配新数组。
4. subslice 默认共享底层数组；copy 后才独立。
5. nil slice 和 empty slice JSON 输出不同。
6. map 需要排序 key 才能稳定输出。
7. `len(string)` 返回字节数，不是 rune 数。
8. `strings.Builder` 通常比循环 `+` 和 `fmt.Sprintf` 更适合大量拼接。

## 运行命令

```bash
cd labs/03-slice-map-string

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## 预期输出

### 测试

`go test ./...` 应通过，覆盖：

- append 共享底层数组。
- append 扩容后不共享。
- 限制 cap 后 append 不影响原数组。
- shared subslice 会观察到底层数组修改，copy subslice 不会。
- nil slice JSON 为 `null`，empty slice JSON 为 `[]`。
- map key 排序后输出稳定。
- 中文字符串 byte len 与 rune len 不同。
- 按 rune 截断不会破坏 UTF-8。

### Benchmark

`go test -bench=. -benchmem ./...` 会输出字符串拼接和 subslice/copy 的分配情况。

重点观察：

- `BenchmarkConcatPlus`
- `BenchmarkConcatSprintf`
- `BenchmarkConcatBuilder`
- `BenchmarkConcatBuilderGrow`
- `BenchmarkFirstNShared`
- `BenchmarkFirstNCopy`

具体数字会随 Go 版本和机器变化，不要死记。关注相对趋势：

- 循环 `+` 通常分配更多。
- `fmt.Sprintf` 通常更慢。
- `strings.Builder` 尤其是 `Grow` 后通常更稳定。
- shared subslice 零分配，但可能持有大数组。
- copy 有分配成本，但能切断大数组生命周期。

## 现象解释

### append 为什么有时共享，有时不共享？

slice header 包含 `data`、`len`、`cap`。当 cap 足够时，append 可写入原底层数组；当 cap 不足时，runtime 分配新数组并复制旧元素。

### full slice expression 为什么有用？

```go
limited := xs[:len(xs):len(xs)]
```

第三个索引把 cap 限制为 len，使下一次 append 没有剩余容量，从而强制分配新底层数组。

### nil slice 和 empty slice 为什么 JSON 不同？

`encoding/json` 把 nil slice 编码为 `null`，把非 nil 但长度为 0 的 slice 编码为 `[]`。这会影响对外 API contract。

### map 为什么要排序 key？

Go map 遍历顺序不稳定，不能用于稳定输出、签名、测试 golden file。需要稳定顺序时必须取 key 后排序。

### string 为什么会有 byte/rune 差异？

Go string 是 UTF-8 字节序列。`len(s)` 返回字节数。`range` string 时得到的是 rune，即 Unicode code point。

## 变体实验

1. 修改 slice cap，观察 append 是否共享。
2. 把 `FirstNCopy` 改成 `append([]byte(nil), buf[:n]...)`。
3. 增大 benchmark 中 parts 数量，观察拼接方法差异。
4. 把中文字符串换成 emoji，观察 rune 和用户可见字符的区别。
5. 写一个并发读写 map 的 demo，用 `go test -race` 观察 data race。

## 生产启发

- append 返回值必须接住。
- 长期保存小 slice 时，确认是否需要 copy 切断大数组引用。
- API response 如果要求数组，优先返回 empty slice 避免 `null`。
- map 输出需要稳定顺序时必须排序。
- 共享 map 必须有同步策略。
- 文本处理要明确 byte、rune、grapheme 的边界。
- 热点字符串拼接用 benchmark 选择实现。
