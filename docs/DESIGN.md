# eval-prompt — Prompt 资产管理工具完整设计文档

**版本**: V1.0  
**状态**: 产品化开发定稿  
**日期**: 2026-04-24  
**目标读者**: 开发团队、架构评审  
**文档性质**: 可直接进入编码阶段的完整技术设计  

---

## 一、产品定位与边界

### 1.1 一句话定义

团队级 Prompt 资产的「本地私有化管理中枢」，以纯 Go 二进制单文件形式分发，通过浏览器访问 Web UI，Agent 通过 CLI/MCP 协议消费，所有数据不出域。

### 1.2 核心边界

**做什么**：Prompt 的编写、分类、版本控制（Git）、效果验证（Eval）、团队共享、Agent 消费。  
**不做什么**：不托管模型、不提供在线协作编辑（如 Google Docs）、不做云端同步、不做权限复杂的 RBAC（V1.0 阶段）。  
**技术边界**：单二进制文件（Go 编译），零运行时依赖（除浏览器外），SQLite 本地存储，go-git 纯 Go 实现。

---

## 二、技术栈总览

| 层级 | 选型 | 理由 |
|------|------|------|
| **API 协议** | connect-go (HTTP JSON 模式) | 基于 net/http，无额外框架依赖，未来可切 gRPC |
| **ORM** | ent (SQLite dialect) | 类型安全、自动迁移、代码生成，支持 SQLite |
| **Git 操作** | go-git/v6 | 纯 Go，无 CGO，可嵌入二进制，支持 init/add/commit/diff/log |
| **配置** | koanf | 显式依赖注入，YAML + env + 命令行，与 vibe-go 一致 |
| **日志** | log/slog | 结构化 JSON，标准库，零外部依赖 |
| **可观测性** | opentelemetry-go | Trace + Metrics，本地 File Exporter，可选 Jaeger |
| **前端** | React + Vite (静态资源嵌入 Go) | 不依赖 Electron，浏览器访问，Go embed 打包 |
| **ID 生成** | oklog/ulid | 全局唯一、按时间排序、26 字符 Base32 |
| **测试** | testify + gomock + miniredis | 与 vibe-go 测试规范一致 |

**不使用的库**：gin/echo/fiber（connect-go 基于 net/http 足够）、gorm（ent 替代）、zap（slog 替代）、viper（koanf 替代）。

---

## 三、架构总览：vibe-go 5层 + N插件

### 3.1 依赖铁律

```
L5-Gateway → L3-Authz → L4-Service → L2-Domain → L1-Storage
```

核心层定义接口，插件层实现接口，通过依赖注入连接。**核心层禁止 import 插件层具体实现**。

### 3.2 各层职责与 Prompt 资产管理映射

#### L5-Gateway：入口网关

**职责**：TLS终止（本地可省略）、协议适配（HTTP JSON）、全局中间件、请求路由、静态资源服务（React 前端）。

**Prompt 资产管理专属**：
- `PromptAssetHandler`：Asset CRUD、版本管理、Eval 触发
- `EvalHandler`：Eval 执行、报告查询、A/B 比对
- `TriggerHandler`：Prompt 匹配查询
- `MCPHandler`：MCP 协议 SSE 端点
- `StaticHandler`：React 静态资源（由 Go embed 提供）

**中间件注册顺序**：Recover → RequestID → Metrics → Logging → CORS → Auth → Routing

**关键决策**：本地模式 Auth 退化为「操作审计」而非「权限校验」，所有请求默认放行，但记录操作者标识（从请求头或 JWT 解析，本地模式下可为空）。

#### L3-Authz：权限决策（本地模式退化）

**职责**：请求准入控制、Eval 门禁校验、操作审计、Rate Limiting。

**Prompt 资产管理专属**：
- `EvalGateGuard`：拦截 `Label` 移动到 `prod` 的请求，校验目标 Snapshot 的 Eval 得分是否 ≥ 阈值（默认 80）
- `SandboxGuard`：拦截涉及文件系统修改的请求，校验目标路径是否在 Git 仓库目录内（防止路径遍历）
- `AuditLogger`：记录所有写操作（Asset 创建/修改/删除、Label 移动、Eval 触发）到审计日志表

**降级策略**：本地单机无外部依赖，503 降级逻辑保留接口但默认不触发。

#### L4-Service：业务编排

**职责**：输入校验、事务边界、工作流触发、领域协调、插件调度。

**Prompt 资产管理专属 Service**：

| Service | 职责 | 依赖接口 |
|---------|------|---------|
| `AssetService` | Asset CRUD、版本管理、Label 操作 | `GitBridger`, `AssetIndexer` |
| `EvalService` | Eval 编排、A/B 矩阵、报告生成 | `EvalRunner`, `LLMInvoker`, `TraceCollector` |
| `SyncService` | Reconcile（对账）、索引重建、导出备份 | `GitBridger`, `AssetIndexer` |
| `TriggerService` | 触发匹配、Anti-Pattern Guard、变量注入 | `AssetIndexer` |

**插件接口定义（在 `internal/service/interfaces.go`）**：

```go
type GitBridger interface {
    InitRepo(ctx context.Context, path string) error
    StageAndCommit(ctx context.Context, filePath, message string) (string, error)
    Diff(ctx context.Context, commit1, commit2 string) (string, error)
    Log(ctx context.Context, filePath string, limit int) ([]CommitInfo, error)
    Status(ctx context.Context) (added, modified, deleted []string, error)
}

type AssetIndexer interface {
    Reconcile(ctx context.Context) (ReconcileReport, error)
    Search(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error)
    GetByID(ctx context.Context, id string) (*AssetDetail, error)
    Save(ctx context.Context, asset Asset) error
    Delete(ctx context.Context, id string) error
}

type LLMInvoker interface {
    Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)
    InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error)
}

type EvalRunner interface {
    RunDeterministic(ctx context.Context, trace []TraceEvent, checks []DeterministicCheck) (DeterministicResult, error)
    RunRubric(ctx context.Context, output string, rubric Rubric, invoker LLMInvoker) (RubricResult, error)
}

type TraceCollector interface {
    StartSpan(ctx context.Context, assetID, snapshotID string) (context.Context, error)
    RecordEvent(ctx context.Context, event TraceEvent) error
    Finalize(ctx context.Context) (string, error) // 返回 trace 文件路径
}

type ModelAdapter interface {
    // 将面向源模型的 Prompt 转换为目标模型格式
    Adapt(ctx context.Context, prompt PromptContent, sourceModel, targetModel string) (AdaptedPrompt, error)
    // 获取目标模型的推荐参数（temperature、max_tokens 等）
    RecommendParams(ctx context.Context, targetModel string, taskType string) (ModelParams, error)
    // 评估适配后的 Prompt 在目标模型上的预期得分（基于历史数据）
    EstimateScore(ctx context.Context, promptID string, targetModel string) (float64, error)
    // 获取目标模型的特性描述（上下文长度、指令遵循风格、格式偏好）
    GetModelProfile(ctx context.Context, model string) (ModelProfile, error)
}

type AdaptedPrompt struct {
    Content           string            // 转换后的 Prompt 正文
    ParamAdjustments  map[string]float64 // 参数调整：{"temperature_delta": -0.1, "max_tokens_multiplier": 1.5}
    FormatChanges     []string          // 格式变更说明：["XML 标签改为 Markdown 代码块", "减少 few-shot 数量从 5 到 3"]
    Warnings          []string          // 警告：["目标模型上下文窗口 32K，原 Prompt 长度 28K，余量紧张"]
}

type ModelProfile struct {
    ContextWindow      int      // 上下文长度（token）
    InstructionStyle   string   // 指令遵循风格：xml_preference | markdown_preference | explicit_preference
    FewShotCapacity    int      // 建议的 few-shot 示例数量上限
    TemperatureCurve   string   // temperature 响应曲线：linear | steep | flat
    SystemRoleSupport  bool     // 是否支持 system 角色
    JSONReliability    float64  // JSON 结构化输出的可靠性评分（0-1）
}
```

#### L2-Domain：领域核心

**职责**：领域实体、状态机、领域事件收集（Outbox）、业务不变量。

**技术要求**：纯 Go struct，零外部依赖（除标准库），禁止 import 任何第三方包。

**核心实体**：

```go
// internal/domain/asset.go
type Asset struct {
    ID          string
    Name        string
    Description string
    AssetType     string
    Tags        []string
    ContentHash string
    FilePath    string
    State       AssetState // CREATED | EVALUATING | EVALUATED | PROMOTED | ARCHIVED
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type AssetState int
const (
    StateCreated AssetState = iota
    StateEvaluating
    StateEvaluated
    StatePromoted
    StateArchived
)

func (a *Asset) Validate() error {
    if a.ID == "" || !isValidAssetID(a.ID) {
        return DomainError{Code: "L2_201", Message: "Asset ID 格式非法"}
    }
    if a.Name == "" || len(a.Name) > 100 {
        return DomainError{Code: "L2_202", Message: "Name 长度必须在 1-100 之间"}
    }
    return nil
}

func (a *Asset) CanPromote() bool {
    return a.State == StateEvaluated || a.State == StatePromoted
}
```

**领域事件**：

| 事件类型 | 触发时机 | 聚合根 |
|---------|---------|--------|
| `PromptAssetCreatedV1` | Asset 创建成功 | Asset |
| `PromptAssetUpdatedV1` | Asset 内容变更，生成新 Snapshot | Asset |
| `SnapshotCommittedV1` | Git 提交成功 | Snapshot |
| `EvalCompletedV1` | Eval 执行完成 | EvalRun |
| `LabelPromotedV1` | Label 移动到 prod | Label |
| `PromptAdaptedV1` | 跨模型适配完成 | ModelAdaptation |
| `OptimizationSuggestedV1` | Agent 生成优化建议 | Asset |
| `OptimizationAppliedV1` | 优化建议被采纳并写入 | Asset |
| `OptimizationDiscardedV1` | 优化建议被丢弃 | Asset |

**状态机（Asset 生命周期）**：

```
CREATED --[Eval Pass]--> EVALUATED --[Label Set Prod]--> PROMOTED
   ↑                      ↑________[Eval Fail]_________↓
   |________________________[Content Changed]___________|
```

**Outbox 模式**：L2 在业务事务内收集领域事件，通过 L1 接口写入 Outbox 表。L2 禁止直接调用 Git、LLM、Redis。

#### L1-Storage：数据持久

**职责**：ent 实现、事务管理、Outbox 表轮询、事件转发 Redis（可选）。

**关键决策：SQLite 本地模式**

```go
// internal/storage/client.go
func NewSQLite(cfg DatabaseConfig) (*Client, error) {
    drv, err := sql.Open("sqlite3", cfg.DSN+"?_fk=1&_journal_mode=WAL")
    if err != nil {
        return nil, fmt.Errorf("open sqlite: %w", err)
    }
    drv.SetMaxOpenConns(1) // SQLite 单写多读，限制连接数
    drv.SetConnMaxLifetime(0)

    client := ent.NewClient(ent.Driver(drv))

    // 自动迁移
    if err := client.Schema.Create(context.Background()); err != nil {
        return nil, fmt.Errorf("schema create: %w", err)
    }

    return &Client{ent: client}, nil
}
```

**表结构（ent schema）**：

- `Asset`：id(PK), name, description, asset_type, tags(JSON), content_hash, file_path, state, created_at, updated_at
- `Snapshot`：id, asset_id(FK), version, content_hash, commit_hash, author, reason, model, temperature, metrics(JSON), created_at
- `Label`：id, asset_id(FK), name, snapshot_id(FK), updated_at
- `EvalCase`：id(PK), asset_id(FK), name, prompt, should_trigger, expected_output, rubric(JSON), created_at
- `EvalRun`：id, eval_case_id(FK), snapshot_id(FK), status, deterministic_score, rubric_score, rubric_details(JSON), trace_path, token_input, token_output, duration_ms, created_at
- `OutboxEvent`：id, aggregate_type, aggregate_id, event_type, payload, occurred_at, idempotency_key, status, retry_count, created_at
- `AuditLog`：id, operation, asset_id, user_id, details, created_at

**Outbox 轮询器**：

```go
// 后台 goroutine，每 5 秒轮询
func (c *Client) StartOutboxPoller(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                c.processOutbox(ctx)
            }
        }
    }()
}

func (c *Client) processOutbox(ctx context.Context) error {
    // 使用 SQLite 的 BEGIN IMMEDIATE 获取写锁
    tx, err := c.ent.Tx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    events, err := tx.OutboxEvent.Query().
        Where(outboxevent.StatusEQ(outboxevent.StatusPending)).
        Order(ent.Asc(outboxevent.FieldCreatedAt)).
        Limit(100).
        ForUpdate(). // SQLite 通过事务隔离实现
        All(ctx)

    for _, event := range events {
        // 发送到 Redis Stream（本地模式下可跳过）
        // 标记为 processed
        tx.OutboxEvent.UpdateOne(event).SetStatus(outboxevent.StatusProcessed).Exec(ctx)
    }

    return tx.Commit()
}
```

### 3.3 插件层实现

**plugins/gitbridge**（实现 `GitBridger`）：
- 使用 `github.com/go-git/go-git/v6`，纯 Go，无 CGO [^85^]
- 支持 Init、Add、Commit、Diff、Log、Status
- 管理 `.gitignore` 自动写入（排除 SQLite、traces、临时文件）
- 提交信息格式：`[prompt] {asset_id}@{version}: {reason}`

**plugins/llm**（实现 `LLMInvoker`）：
- 支持 OpenAI、Claude、Ollama 本地模型
- `InvokeWithSchema` 使用结构化输出（JSON Schema 约束）
- 配置在 `config.yaml` 的 `plugins.llm` 段

**plugins/eval**（实现 `EvalRunner`）：
- Deterministic Checker：解析 JSONL Trace，执行断言规则
- Rubric Grader：构造评审 Prompt，调用 LLMInvoker
- 内置断言库：command_executed、file_exists、file_count、json_valid、content_contains、json_path

**plugins/mcp**（实现 MCP 协议服务端）：
- 基于 SSE（Server-Sent Events）传输
- 暴露 `prompts/list`、`prompts/get`、`prompts/eval`
- 内部调用 L4-Service，不独立实现业务逻辑

**plugins/modeladapter**（实现 `ModelAdapter` 接口）：
- 内置规则库：Claude ↔ GPT ↔ 本地 7B/13B/72B 的适配映射
- 自动调整：XML/Markdown 格式转换、few-shot 数量裁剪、参数温度补偿
- 历史学习：从 `model_adaptations` 表中学习「哪些转换策略在目标模型上得分更高」
- 上下文感知：根据目标模型的 `context_window` 自动裁剪 Prompt 长度，保留核心指令

---

## 四、前端设计：Web 应用（非 Electron）

### 4.1 技术选型

| 技术 | 选型 | 理由 |
|------|------|------|
| 框架 | React 18 + Vite | 构建快，生态成熟 |
| UI 库 | Ant Design 5 | 企业级组件，密度高 |
| 状态管理 | Zustand | 轻量，无样板代码 |
| 编辑器 | Monaco Editor | VS Code 同款，YAML/Markdown 高亮 |
| 图表 | ECharts | Eval 雷达图、趋势图 |
| 路由 | React Router 6 | 本地单页应用 |

### 4.2 与 Go 后端的集成

React 构建产物通过 Go 的 `embed` 打包进二进制：

```go
// internal/gateway/static.go
import "embed"

//go:embed all:web/dist
var webDist embed.FS

func RegisterStaticRoutes(mux *http.ServeMux) {
    fs := http.FS(webDist)
    mux.Handle("/", http.FileServer(fs))
    // API 路由优先于静态资源
}
```

**开发模式**：
- 前端 `npm run dev` 启动 Vite 开发服务器（端口 5173）
- 后端 `go run cmd/server/main.go` 启动 API 服务（端口 8080）
- 前端通过 Vite proxy 配置将 `/api` 请求转发到 `http://127.0.0.1:8080`

**生产模式**：
- `make build` 执行：前端 `npm run build` → 产物放入 `web/dist/` → Go 编译时 embed → 输出单二进制文件 `ep`
- 用户执行 `./ep`，浏览器访问 `http://127.0.0.1:8080`，自动加载 React 应用

### 4.3 关键视图

**资产库视图**：左侧分类树，右侧资产卡片网格，顶部搜索筛选。每张卡片展示名称、版本、Label、Eval 得分。

**编辑器视图**：Monaco Editor 编辑 `PROMPT.md`，左侧编辑区，右侧实时预览（变量注入后的渲染结果）。底部「保存并提交」按钮，触发 Git 提交。

**版本树视图**：垂直时间轴展示 Snapshot 历史，节点显示版本号、提交信息、Eval 得分。点击节点查看内容，点击连线查看 Diff。

**Eval 面板**：总体得分仪表盘、测试用例列表、Trace 时间轴、Rubric 检查项明细。支持重新执行 Eval。

**A/B 比对视图**：左右分栏，支持内容 Diff、Eval 雷达图比对、Trace 路径差异。

---

## 五、安全设计：运行沙箱与文件系统隔离

### 5.1 威胁模型

| 威胁 | 场景 | 防护措施 |
|------|------|---------|
| 路径遍历 | 用户通过 API 传入 `../../../etc/passwd` 访问系统文件 | 路径 Sanitize + 沙箱根目录限制 |
| 命令注入 | Eval 执行时，Prompt 诱导模型执行 `rm -rf /` | 命令白名单 + 临时目录隔离 |
| 数据泄露 | 本地服务被局域网其他机器访问 | 仅监听 127.0.0.1 |
| 恶意 Prompt | 团队成员上传包含恶意指令的 Prompt | 操作审计 + Eval 门禁 |
| SQLite 注入 | 通过搜索框注入 SQL | 参数化查询（ent 自动处理）|

### 5.2 文件系统沙箱

**沙箱根目录**：Git 仓库根目录（用户初始化时指定，如 `~/eval-prompt-repo/`）。所有文件操作必须限制在此目录内。

**路径 Sanitize**：

```go
// internal/gateway/middleware/sandbox.go
func SandboxGuard(repoRoot string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 提取请求中所有路径参数
            paths := extractPathParams(r)
            for _, p := range paths {
                cleanPath := filepath.Clean(p)
                if strings.Contains(cleanPath, "..") {
                    http.Error(w, "路径包含非法字符", http.StatusBadRequest)
                    return
                }
                absPath := filepath.Join(repoRoot, cleanPath)
                if !strings.HasPrefix(absPath, repoRoot) {
                    http.Error(w, "路径超出沙箱范围", http.StatusForbidden)
                    return
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**文件操作白名单**：系统只允许操作以下路径模式：
- `prompts/**/*.md`：Prompt 资产文件
- `.evals/**/*.yaml`：Eval 配置文件
- `.traces/**/*.jsonl`：Trace 日志（不纳入 Git）
- `.gitignore`：Git 忽略配置
- `.eval-prompt/*.db`：SQLite 数据库（被 Git 忽略）

**禁止操作**：
- 删除或修改 `.git/` 目录
- 访问沙箱根目录以外的任何路径
- 创建可执行文件（`.sh`, `.exe`, `.bat`）
- 修改系统配置文件（`/etc/*`, `C:\Windows\*`）

### 5.3 Eval 执行沙箱

**临时目录隔离**：Eval 执行时，所有文件系统操作在临时目录进行：

```go
// internal/plugins/eval/sandbox.go
func (r *EvalRunner) runInSandbox(ctx context.Context, task EvalTask) (EvalResult, error) {
    // 创建临时目录
    tmpDir, err := os.MkdirTemp("", "pa-eval-*")
    if err != nil {
        return EvalResult{}, err
    }
    defer os.RemoveAll(tmpDir) // 执行完毕后清理

    // 将 Prompt 需要的上下文文件复制到临时目录
    if err := copyWorkspaceFiles(task.WorkspaceFiles, tmpDir); err != nil {
        return EvalResult{}, err
    }

    // 在临时目录内执行模型调用和命令
    // 所有文件操作限制在 tmpDir 内
    // ...
}
```

**命令执行白名单**：Eval 过程中，模型可能执行命令（如 `npm install`）。系统维护一个「允许命令列表」：

```yaml
# config.yaml 中的沙箱配置
sandbox:
  allowed_commands:
    - "npm"
    - "go"
    - "python"
    - "python3"
    - "node"
    - "git"
    - "curl"
    - "mkdir"
    - "cp"
    - "mv"
    - "rm"
  forbidden_patterns:
    - "rm -rf /"
    - "rm -rf /*"
    - "> /etc/"
    - "| sh"
    - "| bash"
    - "curl .* | sh"
  max_execution_time: 60s
  max_file_size: 10MB
  max_file_count: 1000
```

**命令执行前校验**：

```go
func validateCommand(cmd string, cfg SandboxConfig) error {
    // 1. 解析命令名
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return errors.New("空命令")
    }
    cmdName := parts[0]

    // 2. 检查命令是否在白名单
    allowed := false
    for _, c := range cfg.AllowedCommands {
        if cmdName == c {
            allowed = true
            break
        }
    }
    if !allowed {
        return fmt.Errorf("命令 '%s' 不在允许列表中", cmdName)
    }

    // 3. 检查是否包含禁止模式
    for _, pattern := range cfg.ForbiddenPatterns {
        matched, _ := regexp.MatchString(pattern, cmd)
        if matched {
            return fmt.Errorf("命令包含禁止模式: %s", pattern)
        }
    }

    return nil
}
```

**资源限制**：
- 单条 Eval 执行超时：60 秒（可配置）
- 生成文件大小上限：10MB
- 生成文件数量上限：1000 个
- 内存使用上限：通过 ulimit 或 cgroup 限制（高级配置）

### 5.4 网络安全

**监听地址**：默认 `127.0.0.1:8080`，仅本机可访问。可通过环境变量 `APP_SERVER_HOST` 修改，但生产环境建议保持默认。

**CORS 配置**：仅允许 `http://localhost:8080` 和 `http://127.0.0.1:8080` 来源，禁止跨域。

**请求体大小限制**：`10MB`，防止超大请求导致内存溢出。

---

## 六、CLI 设计：Agent 优先入口

### 6.1 命令树

```bash
ep                          # 根命令
├── init <path>             # 初始化仓库 + SQLite
├── serve                   # 启动本地 HTTP 服务（包含 Web UI）
│   ├── --port 8080
│   ├── --host 127.0.0.1
│   └── --no-browser        # 不自动打开浏览器
├── asset                   # 资产操作
│   ├── list
│   ├── show <id>
│   ├── cat <id>            # 纯文本输出（管道首选）
│   ├── create --id <> --file <>
│   ├── edit <id> --stdin
│   └── rm <id>
├── snapshot
│   ├── list <id>
│   ├── diff <id> <v1> <v2>
│   └── checkout <id> <v>
├── label
│   ├── list <id>
│   ├── set <id> <name> <v>
│   └── unset <id> <name>
├── eval
│   ├── run <id>
│   ├── cases <id>
│   ├── compare <id> <v1> <v2>
│   ├── report <run-id>
│   └── diagnose <run-id>          # 失败归因，输出结构化优化建议
│       └── --format <json|md>     # 默认 json，md 供人类阅读
├── trigger
│   └── match <input>
├── sync
│   ├── reconcile
│   └── export
├── adapt <id> <version>           # 跨模型 Prompt 适配
│   ├── --from <source-model>      # 源模型（如 claude-3-5-sonnet）
│   ├── --to <target-model>        # 目标模型（如 gpt-4o）
│   ├── --save-as <new-id>         # 保存为新 Asset（默认覆盖当前）
│   └── --auto-eval                # 适配后自动触发目标模型 Eval
├── optimize <id>                  # Agent 自主优化入口
│   ├── --strategy <strategy>      # 优化策略：failure_driven | score_max | compact
│   ├── --iterations <n>           # 最大迭代次数（默认 3）
│   ├── --threshold-delta <n>      # 得分提升阈值（默认 5 分）
│   └── --auto-promote             # 优化通过后自动申请 Label 晋升
└── version                 # 显示版本信息
```

### 6.2 Agent 管道示例

```bash
# 场景 1：Agent 查询并消费 Prompt
ep trigger match "检查 Go 代码 SQL 注入" --top 1 --json   | jq -r '.[0].id'   | xargs ep cat   | llm --model claude-3-5-sonnet

# 场景 2：CI 自动执行 Eval，阻断构建
ep eval run common/code-review --snapshot v1.2.3 --json   | jq -e '.overall_pass == true' || exit 1

# 场景 3：批量比对生成 PR 评论
ep eval compare common/code-review v1.2.2 v1.2.3 --format markdown   | gh pr comment --body-file -

# 场景 4：启动服务并在后台运行
ep serve --port 8080 &
```

---

## 七、MCP 协议适配器

### 7.1 端点设计

通过 SSE（Server-Sent Events）暴露，路径 `/mcp/v1/sse`。

**方法**：

`prompts/list`：返回可用 Prompt 列表，支持 `asset_type` 和 `tag` 过滤。  
`prompts/get`：获取指定 Prompt 渲染后内容。参数：`id`、`variables`、`label`（默认 `prod`）。  
`prompts/eval`：触发 Eval。参数：`id`、`snapshot_version`、`case_id`。

### 7.2 示例交互

```json
// 请求
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "prompts/get",
  "params": {
    "id": "common/code-review",
    "variables": {"lang": "go"},
    "label": "prod"
  }
}

// 响应
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": "你是一位资深 Go 开发专家...",
    "model": "claude-3-5-sonnet",
    "temperature": 0.1,
    "version": "1.2.3",
    "commit_hash": "a1b2c3d"
  }
}
```

### 7.3 MCP 元能力供给层（Agent 理解 Prompt 而非仅消费）

MCP 协议不仅是「读取 Prompt」的管道，更是 Agent 与 Prompt 资产之间的「元能力协商层」。Agent 需要理解 Prompt 的输入契约、输出契约、质量历史、依赖环境，才能自主决策何时调用、如何调用、调用后如何验证。

**扩展响应格式**：`prompts/get` 的响应在 `result.content` 之外，增加 `meta` 字段：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": "你是一位资深 Go 开发专家...",
    "meta": {
      "input_schema": {
        "required_vars": ["lang", "code_snippet"],
        "optional_vars": ["severity", "focus_area"],
        "defaults": {
          "severity": "strict",
          "focus_area": "security,performance"
        }
      },
      "output_schema": {
        "format": "json",
        "schema": {
          "type": "object",
          "properties": {
            "issues": {
              "type": "array",
              "items": {
                "type": "object",
                "properties": {
                  "priority": {"enum": ["critical", "high", "medium", "low"]},
                  "category": {"type": "string"},
                  "line": {"type": "integer"},
                  "message": {"type": "string"},
                  "suggestion": {"type": "string"}
                }
              }
            }
          }
        }
      },
      "definition_of_done": [
        "输出必须是合法 JSON",
        "至少完成安全性维度的检查",
        "对 critical 级别问题必须给出具体修复代码示例"
      ],
      "anti_patterns": [
        "生成代码",
        "编写新功能",
        "重构建议"
      ],
      "eval_history": {
        "latest_score": 92,
        "deterministic_score": 1.0,
        "rubric_score": 92,
        "trend": "improving",
        "last_eval_at": "2026-04-24T10:00:00Z",
        "total_runs": 47,
        "pass_rate": 0.94
      },
      "dependencies": {
        "requires_files": ["*.go"],
        "requires_env": ["go >= 1.21"],
        "suggested_context": "当前文件路径、项目 go.mod 中的 Go 版本"
      },
      "model_compatibility": {
        "optimized_for": "claude-3-5-sonnet",
        "tested_on": ["claude-3-5-sonnet", "gpt-4o", "qwen-72b-chat"],
        "adaptations_available": true
      }
    }
  }
}
```

**MCP 写入方法（Agent 自主优化通道）**：

`prompts/suggest`：Agent 提交优化建议，生成 diff 但不直接写入主分支，进入「建议队列」等待人类审批或自动合并（若 Eval 通过且得分提升 ≥ 5 分）。

参数：`id`、`current_version`、`suggested_content`、`optimization_reason`、`expected_improvement`。

响应：`suggestion_id`、`diff_url`、`estimated_score`。

`prompts/eval_subscribe`：Agent 订阅 Eval 完成事件，通过 SSE push 推送结果，避免轮询。

参数：`asset_id`、`eval_run_id`。

事件流：`eval.started` → `eval.progress`（每 10% 进度）→ `eval.completed`（含完整评分）。

`prompts/label_request`：Agent 申请移动 Label 指针（如晋升到 `prod`），受 Eval 门禁约束，但增加「Agent 身份」审计字段。

参数：`asset_id`、`label_name`、`target_version`、`agent_identity`、`justification`。

**Agent 身份识别**：MCP 请求头携带 `X-Agent-Identity`（如 `cursor-v1.2` / `claude-code-3.5`），Authz 层记录到审计日志，用于区分人类操作和 Agent 操作。

---

## 八、数据模型（ent schema）

### 8.1 Asset

```go
// internal/ent/schema/asset.go
func (Asset) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").MaxLen(128).NotEmpty().Unique(),
        field.String("name").MaxLen(100).NotEmpty(),
        field.Text("description"),
        field.String("asset_type").MaxLen(64).Optional(),
        field.JSON("tags", []string{}).Optional(),
        field.String("content_hash").MaxLen(64).NotEmpty(),
        field.String("file_path").MaxLen(512).NotEmpty(),
        field.Enum("state").Values("created", "evaluating", "evaluated", "promoted", "archived").Default("created"),
        field.Time("created_at").Default(time.Now),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

func (Asset) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("snapshots", Snapshot.Type),
        edge.To("labels", Label.Type),
        edge.To("eval_cases", EvalCase.Type),
    }
}
```

### 8.2 Snapshot

```go
func (Snapshot) Fields() []ent.Field {
    return []ent.Field{
        field.String("version").MaxLen(32).NotEmpty(),
        field.String("content_hash").MaxLen(64).NotEmpty(),
        field.String("commit_hash").MaxLen(40).Optional(),
        field.String("author").MaxLen(128).Optional(),
        field.String("reason").MaxLen(512).Optional(),
        field.String("model").MaxLen(64).Optional(),
        field.Float("temperature").Optional(),
        field.JSON("metrics", map[string]any{}).Optional(),
        field.Time("created_at").Default(time.Now),
    }
}

func (Snapshot) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("asset", Asset.Type).Ref("snapshots").Unique().Required(),
        edge.To("eval_runs", EvalRun.Type),
        edge.To("labels", Label.Type),
    }
}
```

### 8.3 Label

```go
func (Label) Fields() []ent.Field {
    return []ent.Field{
        field.String("name").MaxLen(32).NotEmpty(),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

func (Label) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("asset", Asset.Type).Ref("labels").Unique().Required(),
        edge.From("snapshot", Snapshot.Type).Ref("labels").Unique().Required(),
    }
}
```

### 8.4 EvalCase

```go
func (EvalCase) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").MaxLen(128).NotEmpty().Unique(),
        field.String("name").MaxLen(128).NotEmpty(),
        field.Text("prompt").NotEmpty(),
        field.Bool("should_trigger").Default(true),
        field.Text("expected_output").Optional(),
        field.JSON("rubric", Rubric{}).Optional(),
        field.Time("created_at").Default(time.Now),
    }
}
```

### 8.5 EvalRun

```go
func (EvalRun) Fields() []ent.Field {
    return []ent.Field{
        field.Enum("status").Values("pending", "running", "passed", "failed").Default("pending"),
        field.Float("deterministic_score").Optional(),
        field.Int("rubric_score").Optional(),
        field.JSON("rubric_details", []RubricCheckResult{}).Optional(),
        field.String("trace_path").MaxLen(512).Optional(),
        field.Int("token_input").Optional(),
        field.Int("token_output").Optional(),
        field.Int("duration_ms").Optional(),
        field.Time("created_at").Default(time.Now),
    }
}
```

### 8.7 ModelAdaptation

```go
func (ModelAdaptation) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").MaxLen(128).NotEmpty().Unique(),
        field.String("prompt_id").MaxLen(128).NotEmpty(),
        field.String("source_model").MaxLen(64).NotEmpty(),
        field.String("target_model").MaxLen(64).NotEmpty(),
        field.Text("adapted_content").NotEmpty(),
        field.JSON("param_adjustments", map[string]float64{}).Optional(),
        field.JSON("format_changes", []string{}).Optional(),
        field.Float("eval_score").Optional(),
        field.Int("eval_run_id").Optional(),
        field.Time("created_at").Default(time.Now),
    }
}

func (ModelAdaptation) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("asset", Asset.Type).Ref("adaptations").Field("prompt_id").Required(),
    }
}
```

### 8.6 OutboxEvent

```go
func (OutboxEvent) Fields() []ent.Field {
    return []ent.Field{
        field.String("aggregate_type").MaxLen(64).NotEmpty(),
        field.String("aggregate_id").MaxLen(128).NotEmpty(),
        field.String("event_type").MaxLen(128).NotEmpty(),
        field.JSON("payload", map[string]any{}).NotEmpty(),
        field.Time("occurred_at").Default(time.Now),
        field.String("idempotency_key").MaxLen(256).Unique(),
        field.Enum("status").Values("pending", "processed", "failed").Default("pending"),
        field.Int("retry_count").Default(0),
        field.Time("created_at").Default(time.Now),
    }
}
```

---

## 九、配置管理（koanf）

```go
// internal/config/config.go
type Config struct {
    Server        ServerConfig       `koanf:"server"`
    Database      DatabaseConfig     `koanf:"database"`
    Telemetry     TelemetryConfig    `koanf:"telemetry"`
    Sandbox       SandboxConfig      `koanf:"sandbox"`
    Plugins       PluginsConfig      `koanf:"plugins"`
    PromptAssets  PromptAssetsConfig `koanf:"prompt_assets"`
}

type ServerConfig struct {
    Port         int           `koanf:"port"`
    Host         string        `koanf:"host"`
    MetricsPort  int           `koanf:"metrics_port"`
    PprofPort    int           `koanf:"pprof_port"`
    ReadTimeout  time.Duration `koanf:"read_timeout"`
    WriteTimeout time.Duration `koanf:"write_timeout"`
}

type DatabaseConfig struct {
    DSN         string        `koanf:"dsn"`
    MaxOpen     int           `koanf:"max_open"`
    MaxIdle     int           `koanf:"max_idle"`
    MaxLifetime time.Duration `koanf:"max_lifetime"`
}

type SandboxConfig struct {
    AllowedCommands   []string      `koanf:"allowed_commands"`
    ForbiddenPatterns []string      `koanf:"forbidden_patterns"`
    MaxExecutionTime  time.Duration `koanf:"max_execution_time"`
    MaxFileSize       int64         `koanf:"max_file_size"`
    MaxFileCount      int           `koanf:"max_file_count"`
}

type PromptAssetsConfig struct {
    RepoPath      string `koanf:"repo_path"`
    AssetsDir     string `koanf:"assets_dir"`
    EvalsDir      string `koanf:"evals_dir"`
    TracesDir     string `koanf:"traces_dir"`
    EvalThreshold int    `koanf:"eval_threshold"`
}

type PluginsConfig struct {
    LLM   LLMPluginConfig   `koanf:"llm"`
    Search SearchPluginConfig `koanf:"search"`
}
type ModelAdapterConfig struct {
    Enabled        bool                `koanf:"enabled"`
    DefaultSource  string              `koanf:"default_source"`  // 默认源模型
    RulesPath      string              `koanf:"rules_path"`      // 自定义适配规则 YAML 路径
    AutoLearn      bool                `koanf:"auto_learn"`      // 是否从历史适配数据自动学习
}

type LLMPluginConfig struct {
    Enabled      bool   `koanf:"enabled"`
    Provider     string `koanf:"provider"` // openai | claude | ollama
    APIKey       string `koanf:"api_key"`
    Endpoint     string `koanf:"endpoint"`
    DefaultModel string `koanf:"default_model"`
}
```

---

## 十、日志与可观测性

### 10.1 slog 规范

```go
// 正确示例
slog.Info("asset_created",
    "layer", "L4",
    "asset_id", req.ID,
    "asset_type", req.AssetType,
    "trace_id", span.SpanContext().TraceID(),
)

// 错误示例（禁止）
log.Println("asset created: " + req.ID)
fmt.Println(req)
```

### 10.2 OpenTelemetry

**Trace**：覆盖 Gateway → Service → Domain → Storage 全链路，Span 标签携带 `asset_id`、`eval_case_id`。  
**Metrics**：`prompt_assets_total`（按 asset_type）、`eval_runs_total`（按 status）、`eval_duration_seconds`（Histogram）。  
**健康检查**：`/healthz`（存活）、`/readyz`（就绪，检查 SQLite）。

---

## 十一、测试规范

### 11.1 分层测试

| 层级 | 策略 | 工具 |
|------|------|------|
| 单元测试 | 零外部依赖，所有接口可 mock | gomock, testify |
| 集成测试 | SQLite 内存模式或临时文件 | ent 的 SQLite dialect |
| E2E 测试 | 启动完整服务，HTTP 调用 | httpexpect |

### 11.2 架构合规检查

```bash
# scripts/check_architecture.sh
# 确保依赖铁律不被破坏
gateway → authz → service → domain → storage
```

---

## 十二、Agent 自主优化协议（Auto-Optimization Protocol）

### 12.1 定位

这是本工具区别于所有现有 Prompt 管理工具的核心竞争力。不仅管理 Prompt，更让 Agent 能够**自主发现缺陷、自主迭代优化、自主验证效果、自主申请发布**——形成完整的「人机协同」闭环。

### 12.2 优化流程状态机

```
IDLE → ANALYZING → SUGGESTING → EVALUATING → COMPARING → DECIDING → [APPLY|DISCARD] → IDLE
   ↑________________________________________________________[人类审批]________________________↓
```

**状态定义**：
- `IDLE`：等待触发（定时任务、Eval 失败通知、人类指令）
- `ANALYZING`：分析失败用例或低分项，定位问题根因
- `SUGGESTING`：生成优化建议（修改 Description/Examples/Instruction）
- `EVALUATING`：对新版本执行完整 Eval
- `COMPARING`：A/B 比对新旧版本得分
- `DECIDING`：根据决策规则判断是否保留
- `APPLY`：写入新版本，可选申请 Label 晋升
- `DISCARD`：记录失败原因，回到 IDLE

### 12.3 触发条件

**自动触发**（Agent 自主）：
- Eval 得分低于阈值（默认 70 分）
- Eval 趋势连续下降（最近 3 次运行得分递减）
- 负面控制误触发率上升（`should_trigger=false` 的用例被错误激活）
- 新模型发布（系统检测到目标模型版本升级，自动触发适配优化）

**人工触发**：
- `ep optimize common/code-review --strategy failure_driven`
- UI 中的「智能优化」按钮

### 12.4 失败归因引擎（Failure Attribution Engine）

`ep eval diagnose <run-id>` 的核心实现，将 Eval 失败转化为可操作的优化指令。

**归因维度**：

| 维度 | 分析内容 | 输出 |
|------|---------|------|
| **指令清晰度** | Instruction 是否存在歧义、步骤是否可执行 | 「步骤 3 的『配置 Tailwind』过于模糊，应增加具体命令」 |
| **示例质量** | Examples 是否覆盖边界情况、输入输出是否一致 | 「缺少空数组输入的示例，导致模型在空值场景下输出不一致」 |
| **格式约束** | 输出格式要求是否明确、schema 是否完整 | 「未声明 JSON 中 `line` 字段为可选，模型有时省略导致解析失败」 |
| **上下文缺失** | 是否缺少必要的背景信息或环境假设 | 「Prompt 假设用户已安装 Node.js，但未声明，导致环境错误」 |
| **负面控制** | Anti-patterns 是否过于宽泛或过于严格 | 「Anti-pattern『生成代码』过于宽泛，导致合法场景被误拦截」 |
| **模型适配** | 当前 Prompt 是否针对运行模型做了最优适配 | 「目标模型为 GPT-4o，但 Prompt 仍使用 Claude 偏好的 XML 标签」 |

**输出格式**（结构化，Agent 可直接消费）：

```json
{
  "diagnosis_id": "diag-20260424-001",
  "eval_run_id": "run-123",
  "overall_severity": "high",
  "findings": [
    {
      "category": "instruction_clarity",
      "severity": "high",
      "location": "Instruction 步骤 3",
      "problem": "配置 Tailwind 步骤缺少具体命令",
      "evidence": "Trace 显示模型执行了 3 种不同的 Tailwind 配置方式",
      "suggestion": "明确指定 'npm install tailwindcss @tailwindcss/vite' 和 'vite.config.ts 修改内容'",
      "expected_score_improvement": 15
    },
    {
      "category": "model_adaptation",
      "severity": "medium",
      "location": "全局",
      "problem": "Prompt 使用 XML 标签，但目标模型 GPT-4o 对 Markdown 响应更好",
      "evidence": "Rubric 中 code_style 得分 65，低于 Claude 版本的 90",
      "suggestion": "调用 ep adapt --from claude-3-5-sonnet --to gpt-4o",
      "expected_score_improvement": 10
    }
  ],
  "recommended_strategy": "failure_driven",
  "estimated_iterations": 2,
  "confidence": 0.85
}
```

### 12.5 优化策略库

**failure_driven（失败驱动）**：
- 输入：最近一次 Eval 的失败用例
- 行为：针对失败项修改 Prompt，增加约束、补充示例、澄清步骤
- 停止条件：失败用例全部通过，或迭代 3 次仍未通过

**score_max（得分最大化）**：
- 输入：当前 Prompt 的 Rubric 明细
- 行为：针对低分项优化，如「suggestion_quality 得分低 → 增加修复代码示例的详细程度」
- 停止条件：总分达到阈值（默认 90），或迭代 3 次无提升

**compact（压缩优化）**：
- 输入：当前 Prompt 的 Token 消耗和上下文长度
- 行为：删除冗余描述、合并重复示例、精简 Instruction，在保持得分的前提下降低 Token 消耗
- 停止条件：Token 消耗降低 20% 且得分不下降，或迭代 3 次

### 12.6 决策规则

优化版本是否保留，由以下规则自动判定：

```yaml
auto_optimization_rules:
  must_conditions:
    - "新版本的 deterministic_score >= 旧版本的 deterministic_score"
    - "新版本无任何 critical 级 Rubric 检查项失败"
  scoring_rules:
    - condition: "rubric_score >= 旧版本 + 5"
      action: "自动保留，申请 Label 晋升"
    - condition: "rubric_score >= 旧版本 + 2"
      action: "保留，但不自动晋升，等待人类确认"
    - condition: "rubric_score < 旧版本"
      action: "丢弃，记录退化原因"
  safety_rules:
    - "禁止删除原有 Examples（只能增加或修改）"
    - "禁止修改 Definition of Done（只能增加）"
    - "禁止放宽 Anti-patterns（只能收紧）"
```

### 12.7 人类审批与 Agent 自治的平衡

**完全自治模式**（CI/CD 场景）：
- Agent 发现退化 → 自动优化 → Eval 通过 → 自动写入新版本 → 自动申请 Label 晋升
- 人类只在邮件/Slack 收到「Prompt 已自动升级」通知
- 适用于成熟业务线、低风险的 Prompt（如代码格式化、日志解析）

**审批模式**（核心业务场景）：
- Agent 生成优化建议 → 提交 PR（Git diff）→ 人类 Review → 合并后触发 Eval → 手动晋升 Label
- UI 中的「智能优化」面板展示「建议修改点」「预期得分变化」「风险评级」
- 适用于安全检测、代码评审、部署脚本等高风险 Prompt

**混合模式**（默认）：
- `compact` 策略：完全自治（低风险）
- `failure_driven` 策略：审批模式（中风险）
- `score_max` 策略：审批模式（高风险，可能过度优化导致泛化能力下降）

### 12.8 跨模型自适应优化

当团队引入新模型（如从 GPT-4o 升级到 GPT-5，或引入本地 DeepSeek-V3）时，Agent 自动执行：

1. **扫描**：识别所有 `optimized_for` 为旧模型的 Prompt 资产
2. **适配**：调用 `ModelAdapter.Adapt` 生成目标模型版本
3. **评估**：在目标模型上执行 Eval，与原模型得分比对
4. **决策**：若目标模型得分 ≥ 原模型得分的 95%，自动标记为「目标模型兼容」；若差距 > 10%，进入人工审批流程
5. **发布**：通过后，为同一 Asset 增加多模型兼容标签（`gpt-4o-compatible`、`deepseek-v3-compatible`）

---

## 十二、构建与分发

### 12.1 单二进制构建

```makefile
# Makefile
.PHONY: build
build:
	cd web && npm run build
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/ep cmd/server/main.go

.PHONY: install
install:
	cp bin/ep /usr/local/bin/
```

### 12.2 分发方式

- **GitHub Release**：预编译二进制（Linux/macOS/Windows）
- **直接下载**：`curl -fsSL https://raw.githubusercontent.com/hotjp/eval-prompt/main/install.sh | sh`

### 12.3 首次运行

```bash
$ ep init ~/my-team-prompts
Initialized prompt assets repository at /Users/alice/my-team-prompts
SQLite database: /Users/alice/.eval-prompt/index.db
Git repository: /Users/alice/my-team-prompts/.git

$ ep serve
Starting server at http://127.0.0.1:8080
Opening browser...
```

---

## 十三、附录

### 13.1 目录结构

```
eval-prompt/
├── cmd/
│   └── server/
│       └── main.go              # 入口
├── internal/
│   ├── config/                  # koanf 配置
│   ├── gateway/                 # L5：HTTP handler、中间件、静态资源
│   ├── authz/                   # L3：Eval 门禁、沙箱守卫、审计
│   ├── service/                 # L4：业务 service + 插件接口定义
│   ├── domain/                  # L2：领域实体、状态机、事件
│   ├── storage/                 # L1：ent schema、SQLite 客户端、Outbox
│   └── telemetry/               # slog、otel 初始化
├── plugins/
│   ├── gitbridge/               # go-git 实现
│   ├── llm/                     # 模型调用实现
│   ├── eval/                    # Eval 引擎实现
│   └── mcp/                     # MCP 协议服务端
├── web/                         # React 前端
│   ├── src/
│   ├── public/
│   └── dist/                    # 构建产物（Go embed）
├── scripts/
│   └── check_architecture.sh
├── config.yaml                  # 默认配置
├── go.mod
├── Makefile
└── README.md
```

### 13.2 示例 PROMPT.md

```yaml
---
id: common/code-review
description: 对 Go 代码进行结构化评审...
version: 1.2.3
model: claude-3-5-sonnet
temperature: 0.1
asset_type: common
tags: ["go", "review", "quality"]
author: "dev-lead"
eval_required: true
anti_patterns:
  - "生成代码"
  - "编写新功能"
---

## Description
你是一位资深 Go 开发专家...

## Examples
### Input:
```go
func GetUser(id int) (*User, error) { ... }
```

### Output:
```json
{ "issues": [...] }
```

## Instruction
1. 检查编译错误...
2. 按维度检查...

## Definition of Done
- 输出合法 JSON
- 完成安全性检查
```

### 13.3 示例 eval.yaml

```yaml
eval_cases:
  - id: go-sql-injection
    should_trigger: true
    prompt: "请评审以下 Go 代码..."

deterministic_checks:
  - id: output_is_json
    type: json_valid
    path: "output"

rubric:
  max_score: 100
  checks:
    - id: json_correctness
      description: "输出是否为合法 JSON"
      weight: 20
```

---

## 附录：Tag 分类规范

### Tag 维度

| 维度 | 用途 | 示例 |
|------|------|------|
| **类型** | 区分 prompt 类型 | `agent`, `skill`, `workflow`, `system` |
| **业务线** | 按业务领域分类 | `payment`, `auth`, `search` |
| **模型** | 针对的模型 | `gpt-4o`, `claude-3`, `ollama` |
| **场景** | 使用场景 | `internal`, `external`, `prod`, `dev` |

### 常用 Tag 组合

```yaml
# Agent Prompt
tags: [agent, code-review, gpt-4o]

# Skill Prompt
tags: [skill, translation, claude-3]

# Workflow Prompt
tags: [workflow, multi-step, gpt-4o]

# System Prompt
tags: [system, jailbreak-detection, internal]

# Production Ready
tags: [prod, agent, code-review, gpt-4o]
```

### 搜索示例

```bash
# 搜索所有 agent 类型
ep asset list --tags agent

# 搜索 gpt-4o 相关
ep asset list --tags gpt-4o

# 组合搜索
ep asset list --tags agent,gpt-4o
```

---

**文档结束**
