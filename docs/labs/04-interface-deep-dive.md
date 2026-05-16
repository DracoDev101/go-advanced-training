# Lab 4：Interface 深入

本实验配套 Lesson 4，观察 Go interface 的隐式实现、方法集、typed nil、使用方小接口、Clock 测试替身，以及 interface 调用/装箱 benchmark。

## 运行命令

```bash
cd labs/04-interface-deep-dive

go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go build -gcflags="-m -m" ./...
```

## 观察目标

1. 类型不需要声明 `implements`，方法集匹配即可满足 interface。
2. 指针接收者会影响 `T` 与 `*T` 的方法集。
3. typed nil pointer 装进 `error` 后，interface 本身不等于 nil。
4. 返回 error 时要避免直接返回 typed nil。
5. 使用方定义小接口可以让测试 fake 更简单。
6. Clock interface 可以让时间相关逻辑可测试。
7. benchmark 对比具体类型调用、interface 调用和 `any` 装箱。

## 重点现象

### typed nil

```go
var err *ValidationError = nil
return err
```

返回类型是 `error` 时，结果包含动态类型 `*ValidationError` 和动态值 `nil`，所以 `err != nil`。

### 小接口

`Greeter` 只依赖：

```go
type UserFinder interface {
    FindByID(ctx context.Context, id string) (User, error)
}
```

它不需要依赖一个包含 create/update/delete/list 的大 repository。

## 生产启发

- interface 应从使用方需求中产生。
- 小接口比大接口更容易测试和演进。
- 返回 typed nil 是生产事故高发点。
- interface 成本需要用 benchmark 和 escape analysis 判断，不要迷信。
