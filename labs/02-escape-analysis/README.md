# Lab 2：值、指针与逃逸分析

本实验配套 Lesson 2，用最小 Go 代码观察值传递、指针传递、slice header 复制、逃逸分析和 benchmark 结果。

## 观察目标

通过本实验观察：

1. Go 函数参数永远是值传递。
2. 指针参数也是值传递，只是复制的是地址。
3. slice 参数复制的是 header，底层数组仍然共享。
4. append 后如果要让调用方看到新 header，必须返回并赋值。
5. 返回局部变量指针在 Go 中是安全的，但可能触发堆分配。
6. interface boxing、闭包捕获等场景可能导致逃逸。
7. 小 struct 值传递通常很便宜，大 struct 复制成本可能更高。

## 运行命令

```bash
cd labs/02-escape-analysis

go test ./...
go test -bench=. -benchmem ./...
go build -gcflags="-m -m" ./...
```

## 预期现象

### 测试

`go test ./...` 应该通过。测试覆盖：

- `ChangeInt` 不会修改调用方变量。
- `ChangeIntByPointer` 会修改调用方变量。
- `ModifySliceElement` 会通过共享底层数组修改元素。
- `AppendLocal` 不会修改调用方看到的 slice len。
- `AppendReturn` 返回新 slice header 后，调用方能看到新 len。
- `ReturnPointer` 返回局部变量指针是安全的。
- `MakeCounter` 通过闭包保留状态。

### Benchmark

`go test -bench=. -benchmem ./...` 会输出类似：

```text
BenchmarkReturnValue-...              ...   0 B/op   0 allocs/op
BenchmarkReturnPointer-...            ...  16 B/op   1 allocs/op
BenchmarkReturnAny-...                ...  16 B/op   1 allocs/op
BenchmarkSmallByValue-...             ...   0 B/op   0 allocs/op
BenchmarkSmallByPointer-...           ...   0 B/op   0 allocs/op
BenchmarkLargeByValue-...             ...   0 B/op   0 allocs/op
BenchmarkLargeByPointer-...           ...   0 B/op   0 allocs/op
```

具体数字会因 Go 版本、CPU、内联优化而变化，不要死记数字。重点看：

- `B/op`
- `allocs/op`
- 大对象值传递和指针传递的相对成本
- 返回具体类型和返回 `any` 的差异

### 逃逸分析

`go build -gcflags="-m -m" ./...` 中可能看到：

```text
x escapes to heap
moved to heap: x
func literal escapes to heap
```

重点观察这些函数附近的输出：

- `ReturnPointer`
- `ReturnAny`
- `MakeCounter`
- `ConsumeAny`

不同 Go 版本输出措辞可能不同，但现象类似。

## 现象解释

### 为什么 `ChangeInt` 不修改调用方？

因为 `x` 是调用方变量的副本。

### 为什么 `ChangeIntByPointer` 能修改调用方？

因为传入的是地址值的副本。副本和原地址都指向同一个变量。

### 为什么 slice 元素会被修改？

slice header 被复制，但 header 中的 data pointer 指向同一个底层数组。

### 为什么 append 后 len 不变？

`append` 返回的是新的 slice header。函数内修改的是 header 副本，调用方原 header 不会自动更新。

### 为什么返回局部变量指针安全？

Go 编译器会发现局部变量生命周期超过函数返回，于是把它移动到堆上。

### 为什么 interface 可能导致分配？

具体值进入 interface 时需要携带类型信息和数据。跨越返回值、全局变量、动态调用边界时，编译器可能无法证明其生命周期局限在栈上，于是让它逃逸。

## 变体实验

1. 修改 `Small`，增加字段，看 benchmark 是否变化。
2. 修改 `Large` 的数组大小，观察值传递成本。
3. 把 `ReturnAny` 改为接收参数再返回 `any`，观察逃逸输出变化。
4. 把 benchmark 中的全局 sink 去掉，观察编译器是否优化掉代码。
5. 新增一个 goroutine 捕获局部变量的函数，观察是否逃逸。

## 生产启发

- 不要说 Go 有引用传递；准确说法是“按值复制 descriptor/header”。
- request DTO、值对象、小 struct 通常适合值传递。
- service、repository、client、cache、包含锁的对象通常适合指针。
- 在性能敏感路径中，interface boxing 和闭包捕获可能增加 allocation。
- 看到逃逸不等于必须优化；只有 profile 或 benchmark 证明它是瓶颈时才优化。
- 优先写清楚语义，再用工具验证性能。
