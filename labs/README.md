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
