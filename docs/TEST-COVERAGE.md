# 测试覆盖设计文档

**版本**: V1.1
**状态**: 待开发
**日期**: 2026-04-25
**更新日志**: V1.1 - 补充 CLI 测试章节、修正 MCP 端点路径、更新 gitbridge 测试策略、补充 AuditLogRepository 和 ModelAdaptationRepository
**目标读者**: 开发团队

---

## 一、测试策略总览

### 1.1 三层测试体系

| 类型 | 策略 | 工具 | 覆盖范围 |
|------|------|------|---------|
| **单元测试** | L2 Domain 零外部依赖；Service 层 mock 插件接口 | `testify` + `gomock` | L2 Domain、L4 Service、L3 Authz、Plugins |
| **集成测试** | SQLite 内存模式（`:memory:`），不启动容器 | `ent` SQLite dialect + `testify` | L1 Storage |
| **E2E 测试** | httptest 启动真实 Handler，测完整请求链路 | `httpexpect` | L5 Gateway、跨层集成 |

### 1.2 当前状态

- `internal/authz/` 有 3 个测试文件（eval_gate_guard_test.go, sandbox_guard_test.go, audit_logger_test.go）
- `go test ./...` 可通过，但大部分层无测试覆盖
- L2 Domain、L4 Service、Storage、Gateway 均需补充测试

### 1.3 优先补全顺序

```
L2 Domain          → P0  (纯单元，最快出效果)
L4 Service         → P0  (mock 接口，核心业务逻辑)
L1 Storage         → P1  (SQLite 内存集成测试)
L3 Authz           → P1  (补全边界情况)
L5 Gateway         → P2  (httptest E2E)
Plugins            → P2  (mock 外部依赖)
CLI Commands       → P2  (命令参数和输出)
```

---

## 二、分层测试设计

### 2.1 L2 Domain（优先级 P0）

**原则**：纯 Go struct，零外部依赖，直接 `require`/`assert` 断言。

#### 必测模块

| 文件 | 测试内容 | 覆盖方式 |
|------|---------|---------|
| `domain/asset.go` | `Validate()`、ID 格式校验、状态转换规则 | 等价类划分：合法/非法 ID、空 Name、超长 Name |
| `domain/state_machine.go` | 状态机转换、CanPromote、条件分支 | 覆盖所有状态转换路径 |
| `domain/errors.go` | 错误码格式（`L{层}{3位序号}`） | 校验错误码格式是否合规 |
| `domain/eval_case.go` | Rubric 校验、should_trigger 布尔逻辑 | 正常/异常 rubric JSON |
| `domain/eval_run.go` | Status 枚举、得分边界（0-100） | 边界值测试 |
| `domain/snapshot.go` | Version 格式校验、ContentHash | 合法/非法版本号 |
| `domain/label.go` | Label Name 校验（prod/draft 等） | 合法/非法标签名 |
| `domain/events.go` | 事件结构（ULID、occurred_at、idempotency_key） | 事件字段完整性校验 |
| `domain/model_adaptation.go` | 适配参数校验 | 参数范围测试 |

#### 示例：asset_test.go

```go
package domain

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestAsset_Validate(t *testing.T) {
    tests := []struct {
        name    string
        asset   Asset
        wantErr bool
        errCode string
    }{
        {
            name: "valid asset",
            asset: Asset{ID: "common/code-review", Name: "Code Review"},
            wantErr: false,
        },
        {
            name:    "empty ID",
            asset:   Asset{ID: "", Name: "Test"},
            wantErr: true,
            errCode: "L2_201", // Asset ID 格式非法
        },
        {
            name:    "empty name",
            asset:   Asset{ID: "common/test", Name: ""},
            wantErr: true,
            errCode: "L2_202", // Name 长度必须在 1-100 之间
        },
        {
            name:    "name too long",
            asset:   Asset{ID: "common/test", Name: strings.Repeat("a", 101)},
            wantErr: true,
            errCode: "L2_202",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.asset.Validate()
            if tt.wantErr {
                require.Error(t, err)
                require.Equal(t, tt.errCode, err.(*DomainError).Code)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

func TestAsset_CanPromote(t *testing.T) {
    require.False(t, (&Asset{State: StateCreated}).CanPromote())
    require.True(t,  (&Asset{State: StateEvaluated}).CanPromote())
    require.True(t,  (&Asset{State: StatePromoted}).CanPromote())
    require.False(t, (&Asset{State: StateArchived}).CanPromote())
}
```

---

### 2.2 L4 Service（优先级 P0）

**原则**：通过 `interfaces.go` 定义的接口注入 mock，测试业务编排逻辑，不测外部依赖。

#### 接口 mock 示例

```go
// service/asset_service_test.go
package service

import (
    "testing"
    "context"
    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/require"

    "github.com/eval-prompt/internal/domain"
)

func TestAssetService_CreateAsset(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockGitBridge := NewMockGitBridger(ctrl)
    mockIndexer := NewMockAssetIndexer(ctrl)

    svc := NewAssetService(&Config{
        GitBridge:  mockGitBridge,
        Indexer:    mockIndexer,
    })

    mockGitBridge.EXPECT().
        StageAndCommit(gomock.Any(), gomock.Any(), gomock.Any()).
        Return("abc123", nil)

    mockIndexer.EXPECT().
        Save(gomock.Any(), gomock.Any()).
        Return(nil)

    asset := &domain.Asset{
        ID:   "common/test",
        Name: "Test Asset",
    }

    err := svc.CreateAsset(context.Background(), asset)
    require.NoError(t, err)
    require.NotEmpty(t, asset.ID)
}
```

#### 必测场景

| Service | 必测场景 |
|---------|---------|
| `AssetService` | CreateAsset（Git 提交 + 索引保存）、UpdateAsset（版本递增）、DeleteAsset |
| `EvalService` | RunEval（触发评测）、CompareVersions（得分对比）、Diagnose（失败归因） |
| `TriggerService` | Match（语义匹配，返回 top N） |
| `SyncService` | Reconcile（Git 与 DB 对账）、Export（全量导出） |

---

### 2.3 L1 Storage（优先级 P1）

**原则**：使用 SQLite `:memory:` 模式，不启动容器，ent 内置支持。

#### SQLite 内存测试配置

```go
// internal/storage/client_test.go
package storage

import (
    "context"
    "testing"
    "github.com/eval-prompt/internal/storage/ent"
    "github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestAssetRepository_Create(t *testing.T) {
    client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
    defer client.Close()

    repo := NewAssetRepository(client)
    asset := &domain.Asset{
        ID:   "common/test",
        Name: "Test",
    }

    err := repo.Create(context.Background(), asset)
    require.NoError(t, err)

    got, err := repo.GetByID(context.Background(), "common/test")
    require.NoError(t, err)
    require.Equal(t, "Test", got.Name)
}
```

#### 必测 Repository

| Repository | 覆盖场景 |
|-----------|---------|
| `AssetRepository` | Create、GetByID、List、Update、Delete |
| `SnapshotRepository` | Create、GetByAssetID、GetByVersion、Delete |
| `LabelRepository` | SetLabel、UnsetLabel、GetByAssetID |
| `EvalCaseRepository` | Create、ListByAssetID |
| `EvalRunRepository` | Create、GetByCaseID、ListBySnapshotID、GetStatus |
| `OutboxRepository` | WriteEvent、QueryPending、MarkProcessed |
| `AuditLogRepository` | Create、ListByAssetID、ListByOperation |
| `ModelAdaptationRepository` | Create、GetByID、List、Delete |

---

### 2.4 L3 Authz（优先级 P1）

**原则**：已有一部分，继续补全边界情况。

#### 需补全测试

| 文件 | 缺失场景 |
|------|---------|
| `sandbox_guard_test.go` | 路径遍历攻击（`../etc/passwd`）、沙箱目录外路径 |
| `eval_gate_guard_test.go` | 刚好 80 分阈值、79 分拒绝、100 分通过 |
| `audit_logger_test.go` | 写入成功、无资产操作日志 |

---

### 2.5 L5 Gateway（优先级 P2）

**原则**：用 `httptest` 启动真实 Handler，测 HTTP 请求链路。

#### E2E 测试示例

```go
// internal/gateway/handlers/asset_handler_test.go
package handlers

import (
    "testing"
    "net/http"
    "net/http/httptest"
    "encoding/json"
    "github.com/stretchr/testify/require"
)

func TestAssetHandler_List(t *testing.T) {
    mux := http.NewServeMux()
    // 注册 handler（注入 mock service）
    handler := NewAssetHandler(mockAssetService)
    mux.HandleFunc("/api/v1/assets", handler.List)

    req := httptest.NewRequest("GET", "/api/v1/assets", nil)
    rec := httptest.NewRecorder()

    mux.ServeHTTP(rec, req)

    require.Equal(t, http.StatusOK, rec.Code)

    var resp ListAssetsResponse
    err := json.Unmarshal(rec.Body.Bytes(), &resp)
    require.NoError(t, err)
    require.NotEmpty(t, resp.Assets)
}
```

#### 必测端点

| Handler | 路由 | 覆盖场景 |
|---------|------|---------|
| `AssetHandler` | `GET /api/v1/assets`、`POST /api/v1/assets` | 列表、创建、参数校验 |
| `EvalHandler` | `POST /api/v1/eval/run`、`GET /api/v1/eval/report/:id` | 触发评测、查询报告 |
| `TriggerHandler` | `GET /api/v1/trigger/match` | 语义匹配、参数校验 |
| `MCPHandler` | `POST /mcp/v1`、`GET /mcp/v1/sse` | MCP JSON-RPC 协议端点 |

---

### 2.6 Plugins（优先级 P2）

**原则**：mock 外部依赖（Git、LLM），测内部逻辑。

#### 插件测试策略

| Plugin | Mock 依赖 | 测什么 |
|--------|---------|-------|
| `gitbridge` | 系统 git 命令（exec.Command） | Init、StageAndCommit、Diff、Status（需 mock exec） |
| `llm` | HTTP Client mock | OpenAI/Claude/Ollama 调用封装 |
| `eval` | LLM mock | 断言执行、Rubric 评分 |
| `modeladapter` | 无外部依赖 | 格式转换规则、参数补偿 |

---

### 2.7 CLI Commands（优先级 P2）

**原则**：用 `os/exec` 启动真实命令，测 CLI 参数和输出。

#### CLI 测试示例

```go
// cmd/ep/commands/asset_test.go
package commands

import (
    "os/exec"
    "testing"
    "github.com/stretchr/testify/require"
)

func TestAssetList(t *testing.T) {
    cmd := exec.Command("./ep", "asset", "list", "--json")
    cmd.Dir = t.TempDir()

    // 先 init
    initCmd := exec.Command("./ep", "init", ".")
    initCmd.Dir = cmd.Dir
    require.NoError(t, initCmd.Run())

    // 执行 list
    out, err := cmd.CombinedOutput()
    require.NoError(t, err)
    require.Contains(t, string(out), "assetID")
}
```

#### 必测命令

| 命令 | 覆盖场景 |
|------|---------|
| `ep init <path>` | Git 初始化、SQLite 创建 |
| `ep asset list` | 列表输出、--json 格式 |
| `ep asset create` | 创建资产、参数校验 |
| `ep serve --port` | 启动服务、端口占用 |
| `ep sync reconcile` | 对账功能 |

---

## 三、测试骨架生成

### 3.1 生成所有空测试文件

以下所有包需要创建空的 `_test.go` 文件，先让 coverage 命令不遗漏：

```bash
# L2 Domain
touch internal/domain/asset_test.go
touch internal/domain/audit_log_test.go
touch internal/domain/errors_test.go
touch internal/domain/eval_case_test.go
touch internal/domain/eval_run_test.go
touch internal/domain/events_test.go
touch internal/domain/label_test.go
touch internal/domain/model_adaptation_test.go
touch internal/domain/snapshot_test.go
touch internal/domain/state_machine_test.go
touch internal/domain/types_test.go

# L4 Service
touch internal/service/asset_service_test.go
touch internal/service/eval_service_test.go
touch internal/service/sync_service_test.go
touch internal/service/trigger_service_test.go

# L1 Storage
touch internal/storage/asset_repository_test.go
touch internal/storage/audit_log_repository_test.go
touch internal/storage/eval_case_repository_test.go
touch internal/storage/eval_run_repository_test.go
touch internal/storage/label_repository_test.go
touch internal/storage/model_adaptation_repository_test.go
touch internal/storage/outbox_repository_test.go
touch internal/storage/snapshot_repository_test.go

# L5 Gateway
touch internal/gateway/handlers/asset_handler_test.go
touch internal/gateway/handlers/eval_handler_test.go
touch internal/gateway/handlers/trigger_handler_test.go
touch internal/gateway/handlers/mcp_handler_test.go
touch internal/gateway/middleware/middleware_test.go
touch internal/gateway/health_test.go

# Plugins
touch plugins/eval/runner_test.go
touch plugins/eval/assertions_test.go
touch plugins/gitbridge/bridge_test.go
touch plugins/llm/invoker_test.go
touch plugins/modeladapter/adapter_test.go
touch plugins/search/search_test.go

# CLI
touch cmd/ep/commands/asset_test.go
touch cmd/ep/commands/eval_test.go
touch cmd/ep/commands/trigger_test.go
```

### 3.2 生成 Mock 接口

```bash
# 安装 mockgen（如果未安装）
go install github.com/golang/mock/mockgen@latest

# 生成 service 层所有接口的 mock
mockgen -source=internal/service/interfaces.go -destination=internal/service/mocks/mock.go -package=mocks

# 生成 plugins 层接口的 mock（如需要）
mockgen -source=plugins/search/indexer.go -destination=plugins/search/mocks/mock.go -package=mocks
mockgen -source=plugins/gitbridge/bridge.go -destination=plugins/gitbridge/mocks/mock.go -package=mocks
```

---

## 四、CI 集成

### 4.1 Makefile 测试目标

```makefile
.PHONY: test test-unit test-integration test-cover

test: test-unit test-integration

test-unit:
	go test -v -race -short ./...

test-integration:
	go test -v -race ./internal/storage/...

test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
```

### 4.2 覆盖率目标

| 阶段 | 覆盖率目标 |
|------|----------|
| 骨架完成后 | >10% |
| L2 Domain 完成 | >40% |
| L4 Service + L1 Storage 完成 | >70% |
| 全量完成 | >85% |

---

## 五、测试规范

### 5.1 命名规范

- 测试文件：`<package>_test.go`
- 测试函数：`Test<Method>_<Scenario>`（如 `TestAsset_Validate_EmptyName`）
- Mock 生成：`mock_<interface>.go`

### 5.2 断言规范

- 使用 `testify/require`（失败即停）
- 使用 `testify/assert`（失败继续）
- **禁止** `t.Error` / `t.Fatal` 裸调用

### 5.3 目录结构

```
internal/
  domain/
    asset.go
    asset_test.go          ← 同包测试
  service/
    asset_service.go
    asset_service_test.go
    mocks/
      mock.go              ← gomock 生成
```

---

**文档结束**
