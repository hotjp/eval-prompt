# vibe-go

中文 | **[English](README.md)**

Go 生产框架脚手架。

> **专为 AI Agent 和 Vibe Coding 设计。**
> vibe-go 从第一天起就为自主 Agent 写代码的时代而生。
> 它提供 AI 需要却不具备的东西 — 架构约束，
> 让 AI 产出的代码是生产级品质，而不只是"能编译通过"。

## 是什么

vibe-go 是一个可编译、可部署的 Go 后端框架脚手架，提供 5 层分层架构和插件倒置机制。fork 后填入业务逻辑即可产出生产级服务。

## 技术栈

| 类别 | 选型 | 用途 |
|---|---|---|
| API | `connect-go` | gRPC + HTTP 双模 |
| ORM | `ent` (pgx) | 类型安全查询、代码生成 |
| 配置 | `koanf` | YAML + 环境变量，显式依赖注入 |
| 日志 | `log/slog` | 结构化 JSON，标准库 |
| ID | `oklog/ulid` | 全局唯一、按时间排序 |
| 缓存 | `go-redis/v9` | 缓存、事件流、分布式锁 |
| 可观测性 | OpenTelemetry + Prometheus | 链路追踪、Metrics |
| 数据库迁移 | `golang-migrate` | 版本化 SQL 迁移 |
| 测试 | testify + gomock + testcontainers + miniredis | 单元 / 集成 / E2E |

## 架构

```
L5-Gateway → L3-Authz → L4-Service → L2-Domain → L1-Storage
```

- 核心层定义接口，插件层实现接口
- 核心层禁止 import 插件层具体实现
- L2-Domain 零外部依赖

```
cmd/server/main.go           # 入口，依赖注入组装
internal/
  gateway/                   # L5: Connect handler, 中间件
  authz/                     # L3: 权限校验
  service/                   # L4: 业务编排（含 interfaces.go）
  domain/                    # L2: 领域核心（零外部依赖）
  storage/                   # L1: Ent + PostgreSQL + Redis
plugins/                     # 插件实现
api/{package}/v1/            # Protobuf 定义
```

## 任务管理：LRA（强烈推荐）

vibe-go 的任务拆分和执行流程围绕 [LRA (Long-Running Agent)](https://hotjp.github.io/long-run-agent/) 设计。LRA 是专为 AI Agent 多轮迭代开发的任务管理工具，提供任务认领、质量门控、状态流转等能力。

LRA 是可选的，你完全可以用自己的任务管理方式。但我们强烈推荐搭配使用：

- 任务定义自带五段式上下文（目标/契约/依赖/约定/验收），agent 开箱即用
- 原子认领 + 锁机制，多 agent 并行不冲突
- Constitution 质量门控，确保每个 task 达标后才算完成
- 与 `TASK-BREAKDOWN.md` 的拆分方法论无缝衔接

**安装：**

```bash
# 检查是否已安装
lra --version

# 安装
pip install long-run-agent
```

**了解更多：** [文档主页](https://hotjp.github.io/long-run-agent/) | [GitHub](https://github.com/hotjp/long-run-agent)

## 快速开始

```bash
# 1. Fork 并重命名
# 2. 填写 CLAUDE.md（Project、Description、LRA profile）
# 3. 按业务域拆分任务
# 4. 开始开发
```

## Agent 指南

按以下顺序阅读文档。每个文档职责单一，不需要提前全部扫一遍。

```
CLAUDE.md               ← 架构约束与编码规则（始终加载）
    ↓
docs/TASK-BREAKDOWN.md  ← 选取任务 → 获取自包含的五段式上下文
    ↓                         （无需阅读其他文档，除非任务引用了它们）
docs/DESIGN.md          ← 业务细节：实体、API proto、DDL、流程
docs/architecture.md    ← 技术细节：配置、日志、遥测、测试
```

### Agent 工作流

```
lra ready                              # 查看可认领任务
lra claim <id>                         # 原子认领
lra show <id>                          # 查看任务详情
    ↓
阅读 TASK-BREAKDOWN.md §TaskID         # 自包含上下文（目标/契约/依赖/约定/验收）
    ↓
实现 → 测试 → 提交
    ↓
lra set <id> completed
lra check <id>                         # 运行 Constitution 质量门控
lra set <id> truly_completed           # 完成
```

### 任务依赖图

```
Phase 0:  T00 → T01 ∥ T02 ∥ T03 → T04 → T05
Phase 1:  T01 → T10 ∥ T12;  T10 → T11 ∥ T13
Phase 2:  T05 → T20∥T30∥T40∥T50∥T60          (Domain，可并行)
               → T21∥T31∥T41∥T51∥T61          (Storage，各自 Domain 完成后)
               → T22∥T32∥T42∥T53∥T62          (Service，各自 Storage 完成后)
Phase 5:  T05 → T70 ∥ T71                     (插件，可并行)
Phase 6:  T61+T52+T70+T71 → T80               (AutoTag 跨模块编排)
Phase 7:  全部 → T90 → T91 → T92 → T93        (Gateway + Authz + 组装)
```

## 文档

| 文档 | 用途 | 何时阅读 |
|---|---|---|
| [CLAUDE.md](CLAUDE.md) | 架构约束、编码规则 | 始终（自动加载） |
| [docs/TASK-BREAKDOWN.md](docs/TASK-BREAKDOWN.md) | 任务定义（含完整上下文） | 每个任务开始前 |
| [docs/DESIGN.md](docs/DESIGN.md) | 业务设计（实体、API、DDL、流程） | 任务引用业务细节时 |
| [docs/architecture.md](docs/architecture.md) | 技术规范（配置、日志、遥测、测试） | 任务引用技术细节时 |
| [docs/TASK-PROMPT.md](docs/TASK-PROMPT.md) | 任务拆分方法论 | 创建新任务拆分时 |
| [lra.md](lra.md) | LRA 命令参考 | 管理任务时 |

## 示例

`docs/DESIGN.md` 和 `docs/TASK-BREAKDOWN.md` 包含一个完整的标签管理系统（Tag Sense）作为示例业务，用于验证框架可用性。它们以 `[Demo]` 标注，不属于框架本身。

## API 文档：buf + proto → TypeScript

**不用 Swagger。** 所有 API 通过 `.proto` 定义，用 `buf` 生成 TypeScript 客户端，前端直接 import 调用。

```
.proto → buf generate → @bufbuild/connect-web (TS client)
                                   ↓
                             浏览器直接 import
```

**为什么不用 OpenAPI/Swagger：**
- Proto → TS 类型生成，前后端始终同步，零 drift
- Agent 直接拿到 TS 客户端代码，不需要理解 HTTP 细节
- 编译时类型检查，不是运行时文档

**核心命令：**
```bash
# 安装工具
brew install buf protobuf

# 生成 TS 客户端
buf generate

# 前端依赖
npm install @bufbuild/connect-web @bufbuild/protobuf
```

**关键文件：**
- `api/{package}/v1/*.proto` — API 定义
- `buf.yaml` — 生成配置
- `api/{package}/v1/generated/ts/` — 生成的 TS 客户端

## License

MIT
