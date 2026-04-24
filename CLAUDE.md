# vibe-go - Go Production Framework

## 技术栈

### 核心框架
| 组件 | 库 | 用途 |
|---|---|---|
| API 协议 | `connect-go` | Connect (gRPC + HTTP 双模) |
| Protobuf | `buf` + `protoc-gen-go` | API 定义与代码生成 |
| API 文档 | `buf` → TS Client | 不 Swagger，用 buf 生成 TS 客户端给前端/Agent |
| ORM | `ent` | 数据库模型、迁移、查询（底层 pgx） |
| ID 生成 | `oklog/ulid` | 全局唯一、按时间排序的 ID |

### 存储
| 组件 | 库 | 用途 |
|---|---|---|
| PostgreSQL | `pgx/v5` | 主数据库驱动（ent 底层使用） |
| Redis | `go-redis/v9` | 缓存、事件流、分布式锁 |

### 可观测性
| 组件 | 库 | 用途 |
|---|---|---|
| 日志 | `log/slog` | 结构化日志（标准库，JSON Handler） |
| 链路追踪 | `opentelemetry-go` | OTLP 导出，概率采样 |
| Metrics | `opentelemetry-go` + Prometheus | 请求延迟、错误率、业务指标 |

### 基础设施
| 组件 | 库 | 用途 |
|---|---|---|
| 配置 | `koanf` | 多源配置加载（YAML + 环境变量），显式依赖注入 |
| 数据库迁移 | `golang-migrate` | Schema 版本管理 |
| HTTP 框架 | `net/http` (标准库) | Connect 底层使用 |

### 测试
| 组件 | 库 | 用途 |
|---|---|---|
| 断言 | `testify` | 断言 + suite |
| Mock | `gomock` | 接口 mock 生成 |
| 集成测试 | `testcontainers-go` | PostgreSQL / Redis 独立容器 |
| Redis Mock | `miniredis` | 单元测试 Redis mock |

---

## 架构概览：5层核心 + N插件

```
依赖方向：L5-Gateway → L3-Authz → L4-Service → L2-Domain → L1-Storage
```

### 核心设计原则
- 核心层定义接口，插件层实现接口，通过依赖注入连接
- **禁止核心层 import 插件层具体实现**
- L2-Domain 零外部依赖（纯 Go struct + 标准库）

### 分层职责
| 层 | 职责 | 关键约束 |
|---|---|---|
| L5-Gateway | TLS终止、协议适配、中间件(Recover/Metrics/CORS)、请求路由 | JWT仅解密不验证，调用L3 |
| L3-Authz | 权限检查(RBAC/OpenFGA)、Rate Limiting、身份验证 | 所有RPC必须通过L3才能到L4（Fail Fast） |
| L4-Service | 输入校验、事务边界、工作流触发、领域协调、插件调度 | 不重复验证权限，通过interface依赖插件 |
| L2-Domain | 领域实体、状态机、事件收集(Outbox)、业务不变量 | 纯Go struct，零外部依赖 |
| L1-Storage | Ent实现、事务管理、Outbox轮询、事件转发Redis | Outbox同库同事务 |

### 插件层（接口倒置）
- 接口定义在 L4-Service（`interfaces.go`），实现在 `plugins/` 目录
- 插件可选，未启用时使用 noop 空实现
- 典型插件：搜索引擎(Meilisearch)、工作流(Temporal)、消息推送等

---

## 项目结构

```
cmd/server/main.go           # 入口，依赖注入组装
internal/
  gateway/                   # L5: Connect handler, 中间件
  authz/                     # L3: 权限校验
  service/                   # L4: 业务编排（含 interfaces.go）
  domain/                    # L2: 领域核心（零外部依赖）
  storage/                   # L1: Ent + PostgreSQL + Redis
plugins/                     # 插件实现（每个子目录一个插件）
api/{package}/v1/            # Protobuf 定义
```

---

## 代码生成规则

### 错误码格式
`L{层号}{3位序号}`，范围：L1=[001,199], L2=[200,399], L3=[400,599], L4=[600,799], L5=[800,999]

### 领域事件
- 格式：`{Aggregate}{Action}V{Version}`
- 必须包含：event_id(ULID), aggregate_type, aggregate_id, event_type, payload, occurred_at, idempotency_key, version
- 通过 Outbox 模式发布（事务内写入，后台轮询转发 Redis Stream）

### 状态机
- 声明式定义（states, transitions, guards, actions）
- 每次转换自动 increment_version（乐观锁）

### 配置管理
- 使用 `koanf` 加载，禁止全局单例
- 配置结构体显式定义，通过构造函数注入
- 支持 YAML 文件 + 环境变量覆盖（`APP_` 前缀）

### 日志规范
- 使用 `log/slog`，禁止 fmt.Println
- 必带字段：trace_id, span_id, layer
- 敏感字段自动脱敏（password, token, api_key）

### 测试策略
- **单元测试**：零外部依赖，gomock + testify + miniredis
- **集成测试**：Testcontainers，每测试独立容器 + 独立 schema
- **E2E测试**：命名空间隔离

### 可观测性
- Tracing：OpenTelemetry OTLP，概率采样
- Metrics：:9090/metrics，Prometheus 格式
- Logging：slog JSON Handler
- Health：/healthz（存活）+ /readyz（就绪，检查 DB + Redis）
- pprof：独立端口 :6060，仅内网访问

---

## 详细规范
完整架构规范见 [docs/architecture.md](docs/architecture.md)

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Project**:
**Description**:

## Architecture

{project_architecture}

## Task Management

This project uses **LRA** for task tracking.
See [lra.md](lra.md) for command reference.

## Quick Start

```bash
lra ready              # Find available work
lra show <id>          # View task details
```

<!-- BEGIN LRA CLAUDE SECTION -->

## LRA Task Management

This project uses **LRA** profile: ****

- Detailed guide: [lra.md](lra.md)
- Use `lra` for all task management
- Run `lra ready` before starting work
- ❌ Do not use markdown TODO lists

<!-- END LRA CLAUDE SECTION -->
