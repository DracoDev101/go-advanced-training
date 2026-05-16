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
