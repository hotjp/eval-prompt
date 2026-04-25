# eval-prompt - Prompt 资产管理工具

## 项目概述

**eval-prompt** 是团队级 Prompt 资产的本地私有化管理中枢，以纯 Go 二进制单文件形式分发，通过浏览器访问 Web UI，Agent 通过 CLI/MCP 协议消费，所有数据不出域。

## 技术栈

### 核心框架
| 组件 | 库 | 用途 |
|---|---|---|
| API 协议 | `connect-go` | Connect (gRPC + HTTP 双模) |
| Protobuf | `buf` + `protoc-gen-go` | API 定义与代码生成 |
| ORM | `ent` (SQLite dialect) | 数据库模型、迁移、查询 |
| ID 生成 | `oklog/ulid` | 全局唯一、按时间排序的 ID |

### 存储
| 组件 | 库 | 用途 |
|---|---|---|
| SQLite | `mattn/go-sqlite3` | 本地存储，单二进制文件 |
| Git 操作 | `go-git/v6` | 版本控制，纯 Go 无 CGO |

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
| 前端 | React + Vite (嵌入 Go) | Web UI，通过 go:embed 打包 |

### 测试
| 组件 | 库 | 用途 |
|---|---|---|
| 断言 | `testify` | 断言 + suite |
| Mock | 手动 mock | 接口 mock 实现 |

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
| L5-Gateway | TLS终止（本地可省略）、协议适配（HTTP JSON）、全局中间件、请求路由、静态资源服务 | 仅监听 127.0.0.1 |
| L3-Authz | 权限检查(EvalGate/Sandbox)、操作审计、Rate Limiting | 本地模式退化为操作审计 |
| L4-Service | 输入校验、事务边界、工作流触发、领域协调、插件调度 | 通过 interface 依赖插件 |
| L2-Domain | 领域实体、状态机、事件收集(Outbox)、业务不变量 | 纯Go struct，零外部依赖 |
| L1-Storage | ent 实现、事务管理、Outbox 表轮询 | SQLite 本地存储 |

### 插件层（接口倒置）
- 接口定义在 L4-Service（`internal/service/interfaces.go`），实现在 `plugins/` 目录
- 插件可选，未启用时使用 noop 空实现
- 典型插件：搜索引擎、Git 操作、工作流等

### 文件操作模式：AssetFileManager
- **目的**：统一 frontmatter 文件的读修改写循环，保证 Git commit 原子性
- **接口**（`internal/service/asset_file.go`）：
  - `GetFrontmatter` - 读取并解析 frontmatter
  - `UpdateFrontmatter` - 读取→应用 updater→写回，保留原 body
  - `WriteContent` - 读取→应用 updater→替换 body→写回，用于内容更新
  - `GetBody` - 剥离 frontmatter 返回纯 markdown body
- **实现**：`plugins/search/search.go` 的 `Indexer` 类型
- **原则**：所有文件操作必须通过 AssetFileManager，确保 Git 操作原子性

### Frontmatter 与 API 分离
- **Frontmatter 是 Git/filesystem 内部实现，API 从不直接操作**
- GET `/assets/{id}/content` 返回剥离 frontmatter 后的纯 body
- PUT `/assets/{id}/content` 只接收 body，server-side 合并到 frontmatter
- **好处**：前端不需要理解 frontmatter 格式，避免元数据泄露

### 并发冲突检测：Content Hash 模式
- **机制**：类似 HTTP ETag，基于内容 SHA256 前 8 字节
- **流程**：
  1. GET 返回 `content_hash`
  2. PUT 时携带 `content_hash` 用于冲突检测
  3. 服务端对比 hash，不匹配返回 409 Conflict
- **解决冲突**：localStorage 暂存草稿，冲突时弹出 DiffEditor 让用户选择

### HTTP 语义：Preference-Applied 与 Last-Modified
- `Preference-Applied: return=representation` - PUT 成功后返回完整资源表示
- `Last-Modified` - 基于 frontmatter `updated_at` 字段
- 前端显示"已保存 X 分钟前"

### 状态变更操作必须写 Git
- Archive/Restore 等操作使用 `UpdateFrontmatter` 原子性更新 state 并 commit
- 不可只改内存 index，必须写盘 + Git commit

---

## 项目结构

```
cmd/
  ep/commands/          # CLI 命令实现
  server/main.go        # 服务入口（TODO: 完善初始化）
internal/
  gateway/              # L5: HTTP handler, 中间件, 静态资源
  authz/                 # L3: EvalGateGuard, SandboxGuard, AuditLogger
  service/               # L4: 业务 service + interfaces.go + mocks
  domain/                # L2: 领域实体、状态机、事件
  storage/               # L1: ent schema + SQLite 客户端
  config/                # koanf 配置加载
  telemetry/             # slog, otel 初始化
plugins/
  gitbridge/             # GitBridger 实现 (go-git)
  llm/                   # LLMInvoker 实现 (OpenAI/Claude/Ollama)
  eval/                  # EvalRunner 实现
  modeladapter/          # ModelAdapter 实现
  search/                # AssetIndexer 实现
api/{package}/v1/       # Protobuf 定义
web/                     # React 前端源码
  dist/                  # 构建产物（go:embed 打包）
```

---

## 代码生成规则

### 错误码格式
`L{层号}{3位序号}`，范围：L1=[001,199], L2=[200,399], L3=[400,599], L4=[600,799], L5=[800,999]

### 领域事件
- 格式：`{Aggregate}{Action}V{Version}`
- 必须包含：event_id(ULID), aggregate_type, aggregate_id, event_type, payload, occurred_at, idempotency_key, version
- 通过 Outbox 模式发布（事务内写入，后台轮询处理）

### 状态机
- 声明式定义（states, transitions, guards, actions）
- 每次转换自动 increment_version（乐观锁）

### 配置管理
- 使用 `koanf` 加载，禁止全局单例
- 配置结构体显式定义，通过构造函数注入
- 支持 YAML 文件 + 环境变量覆盖（`APP_` 前缀）

### 日志规范
- 使用 `log/slog`，禁止 fmt.Println
- 必带字段：layer
- 敏感字段自动脱敏（password, token, api_key）

### 测试策略
- **单元测试**：零外部依赖，mock 接口
- **集成测试**：SQLite 内存模式
- **E2E测试**：启动完整服务，HTTP 调用

### 可观测性
- Tracing：OpenTelemetry OTLP，概率采样
- Metrics：:9090/metrics，Prometheus 格式
- Logging：slog JSON Handler
- Health：/healthz（存活）+ /readyz（就绪，检查 SQLite）
- pprof：独立端口 :6060，仅内网访问

---

## 详细规范
完整架构规范见 [docs/DESIGN.md](docs/DESIGN.md)
完整测试规范见 [docs/TEST-COVERAGE.md](docs/TEST-COVERAGE.md)
