# Lesson 13：编译、链接与二进制部署

## 学习目标

完成本课后，学习者应该能够掌握：Go build cache、module graph、`CGO_ENABLED`；`-trimpath`、`-ldflags`、版本注入、可复现构建；静态/动态链接、scratch/distroless/alpine 镜像取舍；signal、graceful shutdown、readiness。

---

## 关键问题

1. 本课主题解决哪个具体生产问题？
2. 它的正确边界是什么，哪些事情不该由它承担？
3. 默认行为中有哪些容易踩坑的地方？
4. 失败时如何观测、定位和恢复？
5. 在 Production Job Runner 中如何落地并验证？

---

## 核心结论

`go build` 不是最后一步；生产部署要回答：二进制从哪个 commit 来、是否可复现、运行时依赖是什么、如何优雅退出、如何回滚。

---

## 硬核要点

```bash
go build -trimpath -ldflags "-s -w -X main.version=$(git rev-parse --short HEAD)" ./cmd/server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/server
go version -m ./server
```

CGO 关掉更容易静态部署，但 DNS、SQLite、某些驱动或系统库场景可能需要 CGO；Alpine/musl 不是“更小就更好”，要关注证书、时区、DNS 行为。

---

## Production Job Runner 落地

本课内容必须映射到综合项目中的一个可验证设计点：明确 API / worker / repository / infrastructure 的责任边界；给出 timeout、retry、幂等、错误映射或一致性策略；定义至少 3 个观测信号；写一个失败路径测试或故障演练步骤。

---

## 代码实验

配套实验目录：`labs/13-build-link-deploy/`

```bash
cd labs/13-build-link-deploy
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
go vet ./...
```

建议后续把当前最小 Lab 升级为本课专项 Lab，而不是只复用通用事件模型。

---

## 常见误区

1. 用框架或组件名替代设计边界。
2. 只写 happy path，不验证失败路径。
3. 只讨论“能不能跑”，不讨论容量、超时、回滚和观测。
4. 在没有指标和 profile 的情况下过早优化或过早抽象。

---

## 作业与评估

作业：写一页 Production Job Runner 落地设计；补充一个失败路径测试；定义 3 个指标和 2 个日志字段；说明一个不应该使用本技术/模式的场景。

评估：能说清楚机制和边界；能解释失败模式；能把设计落到代码、测试或 profile；能在综合项目中做出取舍并说明理由。
