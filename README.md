# Go 深度进阶与生产级系统设计训练营

> 面向已有 Go 初步编程经验的学习者。重点不是语法入门，而是 Go 语言底层原理、设计哲学、基础概念深入理解，以及生产环境下的系统设计、组件选择与最佳实践。

## 课程定位

这套资料围绕三个核心问题展开：

1. **Go 为什么这样设计？**  
   理解 Go 在简单性、显式错误处理、组合优于继承、CSP 并发模型、统一工具链等方面的工程取舍。

2. **Go 的基础概念底层到底是什么？**  
   深入 slice、map、string、interface、error、goroutine、channel、context、GC、memory model、escape analysis 等机制。

3. **如何用 Go 构建生产级后端系统？**  
   覆盖 HTTP/gRPC、PostgreSQL、Redis、消息队列、observability、pprof、可靠性模式、部署与故障排查。

## 学习目标

完成本课程后，学习者应该能够：

- 解释 Go 的核心设计哲学与工程约束。
- 理解 Go runtime、GC、调度器、内存模型的关键机制。
- 写出可测试、可观测、可维护、可演进的 Go 服务端代码。
- 合理选择 HTTP/gRPC、PostgreSQL、Redis、NATS/Kafka 等组件。
- 能定位 goroutine leak、data race、GC 压力、锁竞争、慢请求、慢 SQL 等生产问题。
- 独立设计并实现一个生产级 Go 后端系统。

## 推荐节奏

建议周期：**8–12 周**。

每节课建议结构：

```text
1. 问题引入
2. 设计哲学
3. 底层机制
4. 代码实验
5. 常见误区
6. 生产实践
7. 工具验证
8. 作业
9. 延伸阅读
```

## 目录结构

```text
go-advanced-training/
  README.md
  syllabus.md
  lessons/
    01-go-design-philosophy.md
  labs/
    README.md
  projects/
    production-job-runner.md
  references/
    reading-list.md
```

## 主线模块

1. Go 设计哲学与工程观
2. 基础概念深度理解
3. Runtime、并发与 memory model
4. GC、内存与性能诊断
5. 生产级 HTTP/gRPC 服务设计
6. 数据库、缓存、消息队列选型
7. 可观测性、可靠性与运维设计
8. 综合项目：Production Job Runner

## Web 预览

本仓库内置 MkDocs Material 配置。

本地预览：

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
mkdocs serve
```

默认访问：

```text
http://127.0.0.1:8000
```

构建静态站点：

```bash
mkdocs build --strict
```

GitHub Pages：

- 已包含 `.github/workflows/pages.yml`。
- 推送到 `main` 后会用 GitHub Actions 构建 `site/` 并部署到 Pages。
- 如果第一次部署失败，需要在 GitHub 仓库 `Settings → Pages` 中把 Source 设置为 `GitHub Actions`。

## 综合项目

最终项目是一个 **Production Job Runner**：任务提交、调度、执行、取消、超时、重试、事件发布、状态持久化、observability、pprof 与 Docker Compose 一体化的生产级 Go 后端系统。
