# Eval Panel 完整设计方案

**版本**: V2.0
**状态**: 设计完成，待实现
**日期**: 2026-04-25
**目标读者**: 开发团队、架构评审

---

## 一、概述

### 1.1 设计目标

Eval Panel 是 Prompt 资产管理的核心质量保障界面，负责：

1. **用例管理** — 定义、编辑、删除 Eval Cases
2. **执行控制** — 单次/多次、多用例/单用例、并发执行
3. **结果展示** — 即时报告、历史趋势、Trace 可视化
4. **聚合分析** — 跨执行的历史数据挖掘

### 1.2 核心价值

Eval 运行结果是高价值数据：
- 直接反映 Prompt 质量
- 消耗真实 token（金钱成本）
- 可用于趋势分析、模型对比、回归测试

### 1.3 设计原则

**三层分离**：LLM Call 原子与评价逻辑分离，支持历史数据重新评分。

```
┌─────────────────────────────────────────────────────────────────┐
│                    三层分离架构                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 1: Run（LLM Call 原子）                                 │
│  ─────────────────────────────────────                          │
│  每次 LLM 调用是独立不可变的原子单位                              │
│  永久保留，可跨模型/跨时间比较                                     │
│                                                                 │
│  Layer 2: Rubric（评价量表版本化）                               │
│  ─────────────────────────────────────                          │
│  评价标准独立版本化管理                                            │
│  变更后不影响历史 Run                                             │
│                                                                 │
│  Layer 3: Evaluation（评分结果）                                 │
│  ─────────────────────────────────────                          │
│  Run × Rubric = Evaluation                                     │
│  同一 Run 可用不同 Rubric 多次评分                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 二、核心概念

### 2.1 Run（LLM Call 原子）

**定义**：一次 LLM 调用及其完整输入输出，是不可变的最小原子单位。

**为什么这样设计**：
- LLM 调用最贵，要保留
- 可复现：相同 prompt + 相同 model + 相同 temperature = 可复现
- 支持后续用新 Rubric 重新评分
- 支持跨模型对比（相同 prompt）

### 2.2 Rubric（评价量表版本化）

**定义**：独立的评价标准版本，可归属于某个 Case。

**为什么版本化**：
- Rubric 会频繁迭代
- 历史评分不能因为 Rubric 变更而失效
- 新 Rubric 可对历史 Run 重新评分

### 2.3 Evaluation（评分结果）

**定义**：用特定 Rubric 对特定 Run 的评分结果。

**特点**：
- Run × Rubric = Evaluation
- 同一 Run 可有多个 Evaluation（不同 Rubric 版本）
- 记录评分时的 Check 快照，不怕 Rubric 被修改

---

## 三、数据模型

### 3.1 实体关系图

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              数据关系图                                       │
└──────────────────────────────────────────────────────────────────────────────┘

                        ┌─────────────┐
                        │   Asset     │
                        │             │
                        │ - id        │
                        │ - name      │
                        └──────┬──────┘
                               │
                               │
                        ┌──────▼──────┐
                        │  EvalCase   │
                        │              │
                        │ - id         │
                        │ - asset_id  │ ───────────────────────────────┐
                        │ - name      │                                │
                        │ - prompt    │                                │
                        │ - rubric_id │ ───────────────────────────┐  │
                        └──────┬──────┘                                │  │
                               │                                       │  │
           ┌───────────────────┼───────────────────┐                   │  │
           │                   │                   │                   │  │
           ▼                   ▼                   ▼                   │  │
    ┌───────────┐       ┌───────────┐       ┌───────────┐            │  │
    │  Rubric   │       │  Rubric   │       │  Rubric   │            │  │
    │  V1      │       │  V2      │       │  V3 (new) │◀───────────┘  │
    └─────┬─────┘       └─────┬─────┘       └───────────┘              │
          │                   │                                       │
          │                   │                                       │
          └───────────────────┼───────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │      Run        │
                    │                  │
                    │ - id             │
                    │ - eval_case_id  │
                    │ - prompt_hash   │  ◀── 用于跨模型/跨时间去重和比较
                    │ - prompt_text   │
                    │ - response      │  ◀── 永久保留的 LLM 输出
                    │ - model         │
                    │ - temperature   │
                    │ - tokens_in/out │
                    │ - duration_ms   │
                    └────────┬────────┘
                             │
           ┌─────────────────┼─────────────────┐
           │                 │                 │
           ▼                 ▼                 ▼
    ┌───────────┐     ┌───────────┐     ┌───────────┐
    │Evaluation │     │Evaluation │     │Evaluation │
    │ V1 评 Run│     │ V2 评 Run │     │ V3 评 Run │  ← 新的！
    │ score: 85 │     │ score: 72 │     │ score: 88 │  ← 用 V3 重新算
    └───────────┘     └───────────┘     └───────────┘
```

### 3.2 EvalCase

测试用例定义，存储在 SQLite 和 `.md` 文件双写。

```go
type EvalCase struct {
    ID           string   `json:"id"`           // 唯一标识
    AssetID      string   `json:"asset_id"`     // 所属资产
    Name         string   `json:"name"`         // 可读名称

    // Prompt 模板（渲染时注入变量）
    PromptTemplate string  `json:"prompt_template"`

    // 当前活跃的 Rubric 版本
    ActiveRubricID string `json:"active_rubric_id"`

    // 变量定义
    Variables     []Variable `json:"variables"`

    // 元数据
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

type Variable struct {
    Name        string `json:"name"`         // 变量名
    Description string `json:"description"`  // 用途描述
    Default     string `json:"default"`      // 默认值
    Required    bool   `json:"required"`     // 是否必填
}
```

### 3.3 Rubric（评价量表）

Rubric 是独立版本化的评价标准。

```go
type Rubric struct {
    ID          string   `json:"id"`           // 唯一标识
    EvalCaseID  string   `json:"eval_case_id"` // 关联的 Case

    Name        string   `json:"name"`         // "SQL注入检测 V2"
    Version     int      `json:"version"`      // 版本号，自动递增

    MaxScore    int      `json:"max_score"`    // 满分
    Checks      []Check  `json:"checks"`       // 检查项列表

    Description string   `json:"description"`  // 说明
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Check struct {
    ID          string `json:"id"`          // 检查项 ID
    Description string `json:"description"` // 检查描述
    Weight      int    `json:"weight"`     // 权重，总和应为 MaxScore
}
```

### 3.4 Run（LLM Call 原子）

**Run 是不可变的 LLM 调用原子单位。**

```go
type Run struct {
    ID         string    `json:"id"`          // ULID
    EvalCaseID string   `json:"eval_case_id"` // 关联的 Case

    // Prompt 信息（用于追溯和比较）
    PromptHash  string   `json:"prompt_hash"`  // prompt 内容 hash，用于去重/比较
    PromptText  string   `json:"prompt_text"`  // 实际发送的完整 prompt

    // LLM 调用结果（不可变）
    Model       string   `json:"model"`        // gpt-4o, claude-3-5-sonnet, etc.
    Temperature float64  `json:"temperature"`   // 0.0-2.0
    Response    string   `json:"response"`      // 原始 LLM 输出
    TokensIn    int     `json:"tokens_in"`     // 输入 token 数
    TokensOut   int     `json:"tokens_out"`    // 输出 token 数
    DurationMs  int64    `json:"duration_ms"`   // 耗时

    // 状态
    Status      RunStatus `json:"status"`  // pending | completed | failed

    // 元数据
    ExecutionID string    `json:"execution_id"` // 所属 Execution
    RunNumber  int       `json:"run_number"`   // 同 Case 的第几次执行
    Error      string    `json:"error,omitempty"` // 失败原因

    CreatedAt   time.Time `json:"created_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type RunStatus string
const (
    RunStatusPending   RunStatus = "pending"
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
)
```

### 3.5 Evaluation（评分结果）

**Evaluation = Run × Rubric，同一 Run 可有多个不同 Rubric 的评分。**

```go
type Evaluation struct {
    ID       string   `json:"id"`
    RunID    string   `json:"run_id"`     // 评的是哪次 LLM Call
    RubricID string   `json:"rubric_id"`  // 用哪个 Rubric 版本

    // 评分结果
    Score    int      `json:"score"`      // 综合得分
    Passed   bool     `json:"passed"`      // 是否通过

    // 详细结果（快照，避免 Rubric 未来被修改影响历史评分）
    DeterministicScore float64            `json:"deterministic_score,omitempty"` // 0.0-1.0
    RubricScore       int                `json:"rubric_score,omitempty"`        // 0-max_score
    Details           []CheckResultSnapshot `json:"details"`                      // 各检查项快照

    // 元数据
    EvaluatedAt time.Time `json:"evaluated_at"`
}

// CheckResultSnapshot 保存评分时的检查项快照
type CheckResultSnapshot struct {
    CheckID   string `json:"check_id"`    // 检查项 ID
    CheckDesc string `json:"check_desc"`  // 评分时的描述（快照）
    Weight    int    `json:"weight"`      // 评分时的权重（快照）
    Passed    bool   `json:"passed"`      // 是否通过
    Score     int    `json:"score"`      // 得分
    Details   string `json:"details"`      // 详细说明
}
```

### 3.6 EvalExecution

执行批次，代表一次完整的评估请求。

```go
type EvalExecution struct {
    ID             string           `json:"id"`
    AssetID       string           `json:"asset_id"`
    SnapshotID    string           `json:"snapshot_id"`

    Mode          ExecutionMode    `json:"mode"`           // single | batch | matrix
    RunsPerCase   int             `json:"runs_per_case"`  // matrix 模式下的重复次数
    CaseIDs       []string        `json:"case_ids"`        // 参与的用例 ID 列表

    TotalRuns     int             `json:"total_runs"`
    CompletedRuns int             `json:"completed_runs"`
    FailedRuns    int             `json:"failed_runs"`

    Status        ExecutionStatus `json:"status"`         // pending | running | completed | partial_failure | failed | cancelled

    Concurrency   int             `json:"concurrency"`    // 并发数
    Model        string          `json:"model"`          // 覆盖的模型
    Temperature  float64         `json:"temperature"`    // 覆盖的温度

    CreatedAt    time.Time       `json:"created_at"`
    StartedAt   *time.Time     `json:"started_at,omitempty"`
    CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}

type ExecutionMode string
const (
    ModeSingle ExecutionMode = "single"  // 1 Case × 1 Run
    ModeBatch  ExecutionMode = "batch"   // N Cases × 1 Run
    ModeMatrix ExecutionMode = "matrix"  // N Cases × M Runs
)

type ExecutionStatus string
const (
    StatusPending        ExecutionStatus = "pending"
    StatusRunning        ExecutionStatus = "running"
    StatusCompleted      ExecutionStatus = "completed"
    StatusPartialFailure ExecutionStatus = "partial_failure"
    StatusFailed         ExecutionStatus = "failed"
    StatusCancelled      ExecutionStatus = "cancelled"
)
```

### 3.7 TraceEvent（可选，用于 Timeline）

评估过程中的事件记录。

```go
type TraceEvent struct {
    ID        string      `json:"id"`         // 事件唯一 ID
    RunID     string      `json:"run_id"`     // 关联的 Run
    SpanID    string      `json:"span_id"`     // Span ID
    ParentID  string      `json:"parent_id"`   // 父 Span ID
    Name      string      `json:"name"`        // 事件名称
    Timestamp time.Time   `json:"timestamp"`
    Type      EventType   `json:"type"`        // span_start | span_end | event | error
    Phase     TracePhase  `json:"phase"`       // prompt_render | llm_call | rubric_eval | scoring
    Data      map[string]any `json:"data,omitempty"`
}

type TracePhase string
const (
    PhasePromptRender TracePhase = "prompt_render"
    PhaseLLMCall     TracePhase = "llm_call"
    PhaseRubricEval  TracePhase = "rubric_eval"
    PhaseScoring     TracePhase = "scoring"
)
```

---

## 四、核心操作

### 4.1 执行新 Eval

```
EvalCase + Variables ──▶ Render Prompt ──▶ LLM Call ──▶ Run
                                                     │
                                                     ▼
                                              Rubric (active version)
                                                     │
                                                     ▼
                                              Evaluation ──▶ Score
```

### 4.2 Rubric 变更后召回重测

```
场景：Rubric V3 发布，想看历史 Run 用新量表的评分

历史数据：
  Run #1 (V1 Rubric 评: 85分)
  Run #2 (V2 Rubric 评: 72分)

新操作：
  对 Run #1, Run #2 用 V3 重新评分
                                    │
                                    ▼
                            ┌───────────────┐
                            │  Evaluation   │
                            │  V3 评 Run#1 │ ──▶ 88分
                            │  V3 评 Run#2 │ ──▶ 75分
                            └───────────────┘
```

### 4.3 批量召回重测

```
POST /api/v1/eval-cases/{caseId}/reevaluate

Request:
{
  "to_rubric_id": "rubric-v3",        // 评估到哪个版本
  "run_filters": {
    "from_date": "2026-04-01",       // 可选
    "model": "gpt-4o"                // 可选
  }
}
```

### 4.4 Rubric 版本对比

```
GET /api/v1/eval-cases/{caseId}/rubric-compare?from=v1&to=v3

Response:
{
  "from_rubric": {"id": "v1", "name": "V1"},
  "to_rubric": {"id": "v3", "name": "V3"},
  "runs_evaluated": 50,
  "comparisons": [
    {
      "run_id": "run-001",
      "from_score": 85,
      "to_score": 88,
      "delta": +3,
      "status_change": "passed → passed"
    },
    {
      "run_id": "run-002",
      "from_score": 72,
      "to_score": 68,
      "delta": -4,
      "status_change": "passed → failed"
    }
  ],
  "summary": {
    "improved": 20,
    "degraded": 8,
    "unchanged": 22,
    "now_failing": 3
  }
}
```

---

## 五、API 设计

### 5.1 端点总览

| 方法 | 路径 | 描述 |
|------|------|------|
| **Eval Case** | | |
| `GET` | `/api/v1/assets/{id}/eval-cases` | 获取资产的所有用例 |
| `POST` | `/api/v1/assets/{id}/eval-cases` | 创建用例 |
| `PUT` | `/api/v1/assets/{id}/eval-cases/{caseId}` | 更新用例 |
| `DELETE` | `/api/v1/assets/{id}/eval-cases/{caseId}` | 删除用例 |
| **Rubric** | | |
| `GET` | `/api/v1/eval-cases/{caseId}/rubrics` | 获取用例的所有 Rubric 版本 |
| `POST` | `/api/v1/eval-cases/{caseId}/rubrics` | 创建新 Rubric 版本 |
| `PUT` | `/api/v1/eval-cases/{caseId}/rubrics/{rubricId}` | 更新 Rubric |
| `GET` | `/api/v1/eval-cases/{caseId}/rubrics/{rubricId}` | 获取特定版本 |
| **Eval Execution** | | |
| `POST` | `/api/v1/evals/execute` | 发起执行 |
| `GET` | `/api/v1/evals/execute/{executionId}` | 获取执行状态 |
| `GET` | `/api/v1/evals/execute/{executionId}/report` | 获取聚合报告 |
| `POST` | `/api/v1/evals/execute/{executionId}/cancel` | 取消执行 |
| **Run** | | |
| `GET` | `/api/v1/runs/{runId}` | 获取单个 Run |
| `GET` | `/api/v1/runs/{runId}/response` | 获取 LLM 原始响应 |
| `GET` | `/api/v1/runs/{runId}/evaluations` | 获取 Run 的所有评分 |
| `POST` | `/api/v1/runs/{runId}/reevaluate` | 对已有 Run 执行新评分 |
| **Evaluation** | | |
| `GET` | `/api/v1/evaluations/{evalId}` | 获取单个 Evaluation |
| `POST` | `/api/v1/eval-cases/{caseId}/reevaluate` | 批量召回重测 |

### 5.2 请求/响应示例

#### POST /api/v1/evals/execute

**Request**
```json
{
  "asset_id": "code-reviewer",
  "snapshot_version": "latest",
  "case_ids": ["case-sql-injection", "case-code-quality"],
  "mode": "matrix",
  "runs_per_case": 3,
  "concurrency": 2,
  "model": "gpt-4o",
  "temperature": 0.7
}
```

**Response 202 Accepted**
```json
{
  "execution_id": "exec-01HV...",
  "mode": "matrix",
  "status": "running",
  "total_runs": 6,
  "run_ids": [
    "run-01HW...", "run-01HW...", "run-01HW...",
    "run-01HW...", "run-01HW...", "run-01HW..."
  ],
  "created_at": "2026-04-25T10:00:00Z",
  "_links": {
    "self": "/api/v1/evals/execute/exec-01HV...",
    "report": "/api/v1/evals/execute/exec-01HV.../report"
  }
}
```

#### POST /api/v1/runs/{runId}/reevaluate

对已有 Run 用新 Rubric 评分：

**Request**
```json
{
  "rubric_id": "rubric-v3-id"
}
```

**Response**
```json
{
  "evaluation_id": "eval-new-id",
  "run_id": "run-id",
  "rubric_id": "rubric-v3-id",
  "score": 88,
  "passed": true,
  "deterministic_score": 1.0,
  "rubric_score": 88,
  "details": [
    {
      "check_id": "sql_injection",
      "check_desc": "检测SQL注入（快照）",
      "weight": 40,
      "passed": true,
      "score": 40,
      "details": "未发现SQL注入风险"
    }
  ],
  "evaluated_at": "2026-04-25T10:05:00Z"
}
```

#### GET /api/v1/runs/{runId}/evaluations

获取某 Run 的所有评分（跨 Rubric 版本）：

**Response**
```json
{
  "run_id": "run-01HW...",
  "run_created_at": "2026-04-24T10:00:00Z",
  "model": "gpt-4o",
  "evaluations": [
    {
      "evaluation_id": "eval-v1",
      "rubric_id": "rubric-v1",
      "rubric_name": "SQL检测 V1",
      "rubric_version": 1,
      "score": 85,
      "passed": true,
      "evaluated_at": "2026-04-24T10:00:05Z"
    },
    {
      "evaluation_id": "eval-v2",
      "rubric_id": "rubric-v2",
      "rubric_name": "SQL检测 V2",
      "rubric_version": 2,
      "score": 72,
      "passed": true,
      "evaluated_at": "2026-04-25T08:00:00Z"
    },
    {
      "evaluation_id": "eval-v3",
      "rubric_id": "rubric-v3",
      "rubric_name": "SQL检测 V3",
      "rubric_version": 3,
      "score": 88,
      "passed": true,
      "evaluated_at": "2026-04-25T10:05:00Z"
    }
  ]
}
```

---

## 六、页面布局

### 6.1 整体布局

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  [Asset Name]                                    [state]    [Run Eval ▼]   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────┐  ┌─────────────────────────────────────────┐   │
│  │    Eval Cases          │  │  Summary Cards                          │   │
│  │    (Left Panel)        │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐   │   │
│  │                         │  │  │ Determ. │ │ Rubric  │ │ Overall │   │   │
│  │  ☑ ☑ Select: 2/5       │  │  │  0.95   │ │  87/100 │ │   83    │   │   │
│  │  ┌─────────────────┐   │  │  └─────────┘ └─────────┘ └─────────┘   │   │
│  │  │☑│ sql-injection │   │  │                                         │   │
│  │  │ │ Rubric: V3    │   │  │  Pass Rate: 83.3% (5/6)                 │   │
│  │  │ │ Runs: 12      │   │  │  Tokens: 3.2K in / 8.9K out           │   │
│  │  ├─────────────────┤   │  │  Cost: ~$0.42                          │   │
│  │  │☑│ code-quality  │   │  │                                         │   │
│  │  │ │ Rubric: V1    │   │  └─────────────────────────────────────────┘   │
│  │  │ │ Runs: 8       │   │                                                 │
│  │  ├─────────────────┤   │  ┌─────────────────────────────────────────┐   │
│  │  │☐│ boundary-test │   │  │  LLM Response Timeline (阶段式)         │   │
│  │  │ │ Rubric: V2    │   │  │  ┌───────┐ ┌───────┐ ┌───────┐ ┌────┐ │   │
│  │  └─────────────────┘   │  │  │Render │─▶│ LLM   │─▶│Rubric │─▶│Done│ │   │
│  │                         │  │  └───────┘ └───────┘ └───────┘ └────┘ │   │
│  │  [+ Add Case]           │  │  12ms      220ms      98ms      5ms    │   │
│  │  [Rubric Versions]      │  │                                         │   │
│  │                         │  │  ┌─ LLM Call Details ─────────────────┐ │   │
│  │  ─────────────────────  │  │  │ Model: gpt-4o │ Tokens: 320→180   │ │   │
│  │                         │  │  │ ───────────────────────────────────│ │   │
│  │  Selected Case Detail   │  │  │ Prompt Preview:                   │ │   │
│  │  ┌─────────────────┐   │  │  │ ┌──────────────────────────────┐│ │   │
│  │  │ Name: sql-inj   │   │  │  │ │ 你是一位Go安全专家...          ││ │   │
│  │  │ Active: V3 (88)│   │  │  │ └──────────────────────────────┘│ │   │
│  │  │ ─────────────── │   │  │  └──────────────────────────────────┘ │   │
│  │  │ Rubric V3:      │   │  │  └─────────────────────────────────────┘   │
│  │  │  ☐ sql (40)   │   │  └─────────────────────────────────────────┘   │
│  │  │  ☐ quality(30) │   │                                                 │
│  │  │  ☐ style (30) │   │  ┌─────────────────────────────────────────┐   │
│  │  │                 │   │  │  Evaluations (跨版本)                    │   │
│  │  │  [Compare V1-V3]│   │  │  V1: 85 │ V2: 72 │ V3: 88 │  ↑ +3  │   │
│  │  └─────────────────┘   │  │  └─────────────────────────────────────────┘   │
│  │                         │  │                                                 │
│  │  Runs History (12)      │  ┌─────────────────────────────────────────┐   │
│  │  ├─────────────────┐   │  │  Check Results (V3)                     │   │
│  │  │ Run #12 ✓  88  │   │  │  ┌───────────────────────────────────┐  │   │
│  │  │ Run #11 ✓  90  │   │  │  │ ✓ sql_injection   40   Passed   │  │   │
│  │  │ Run #10 ✗  72  │   │  │  │ ✗ code_quality    30   Failed   │  │   │
│  │  └─────────────────┘   │  │  │ ✓ style           30   Passed   │  │   │
│  │                         │  │  └───────────────────────────────────┘  │   │
│  │  [Re-evaluate with V3]   │  └─────────────────────────────────────────┘   │
│  └─────────────────────────┘  └─────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 Rubric 版本管理

```
┌─────────────────────────────────────────────────────────────────┐
│  Rubric Versions: sql-injection                        [Close]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Version History                                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  V3 (current)        2026-04-25  ← Active             │   │
│  │  ───────────────────────────────────────────────────  │   │
│  │  + sql_injection   40  ✓                            │   │
│  │  + code_quality   30  ✓                            │   │
│  │  + style          30  ✓                            │   │
│  │                                         [Set Active]    │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │  V2                  2026-04-20                        │   │
│  │  ───────────────────────────────────────────────────  │   │
│  │  + sql_injection   50  ✓                            │   │
│  │  + code_quality   50  ✓                            │   │
│  │                                         [Set Active]    │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │  V1                  2026-04-15                        │   │
│  │  ───────────────────────────────────────────────────  │   │
│  │  + sql_injection   100  ✓                           │   │
│  │                                         [Set Active]    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Compare V1 vs V3                                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Changed: 2 checks, 1 removed, 1 added               │   │
│  │  ───────────────────────────────────────────────────  │   │
│  │  - code_quality (weight: 50 → 30)                    │   │
│  │  + style (weight: 30) - NEW                         │   │
│  │  - sql_injection (weight: 100 → 40)                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ─────────────────────────────────────────────────────────────   │
│                                                                 │
│  Impact Analysis: 3 runs would change score if re-evaluated    │
│  • 2 runs would improve (avg: +5 points)                        │
│  • 1 run would degrade (avg: -3 points)                        │
│                                                                 │
│        [Cancel]                    [Create V4]                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 七、执行模式

### 7.1 三种执行模式

| 模式 | 描述 | 公式 | 适用场景 |
|------|------|------|----------|
| **Single** | 单用例单次运行 | 1 Case × 1 Run | 快速验证、调试 |
| **Batch** | 多用例各单次运行 | N Cases × 1 Run | CI/CD gate、版本对比 |
| **Matrix** | 多用例各多次运行 | N Cases × M Runs | 稳定性测试、统计评估 |

### 7.2 执行流程

```
                    ┌─────────────────────────────────────────┐
                    │           Execution 执行流程               │
                    └─────────────────────────────────────────┘

    ┌──────────────────────────────────────────────────────────┐
    │  1. Plan: Case × Runs = Total Runs                      │
    │     └─▶ Execution created, status = pending           │
    └──────────────────────────────────────────────────────────┘
                              │
                              ▼
    ┌──────────────────────────────────────────────────────────┐
    │  2. For each Case × Run:                              │
    │     │                                                    │
    │     ├─▶ Render Prompt (注入变量)                        │
    │     │                                                    │
    │     ├─▶ LLM Call ──▶ Run (atom, immutable)            │
    │     │        │                                          │
    │     │        └─▶ Token usage, duration, response       │
    │     │                                                    │
    │     ├─▶ Run × Active Rubric = Evaluation               │
    │     │        │                                          │
    │     │        └─▶ Score, passed, details               │
    │     │                                                    │
    │     └─▶ Next iteration                                 │
    └──────────────────────────────────────────────────────────┘
                              │
                              ▼
    ┌──────────────────────────────────────────────────────────┐
    │  3. Aggregate: Execution completed                      │
    │     └─▶ Report with scores, pass rate, token usage     │
    └──────────────────────────────────────────────────────────┘
```

---

## 八、数据库 Schema

### 8.1 核心表结构

```sql
-- Eval Case 表
CREATE TABLE eval_case (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL,
    name TEXT NOT NULL,
    prompt_template TEXT NOT NULL,
    variables TEXT,  -- JSON
    active_rubric_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_case_asset ON eval_case(asset_id);

-- Rubric 表（版本化）
CREATE TABLE eval_rubric (
    id TEXT PRIMARY KEY,
    eval_case_id TEXT NOT NULL,
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    max_score INTEGER NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(eval_case_id, version)
);

CREATE INDEX idx_rubric_case ON eval_rubric(eval_case_id);

-- Rubric Check 表
CREATE TABLE eval_rubric_check (
    id TEXT PRIMARY KEY,
    rubric_id TEXT NOT NULL REFERENCES eval_rubric(id),
    description TEXT NOT NULL,
    weight INTEGER NOT NULL
);

CREATE INDEX idx_check_rubric ON eval_rubric_check(rubric_id);

-- Run 表（LLM Call 原子）
CREATE TABLE eval_run (
    id TEXT PRIMARY KEY,
    eval_case_id TEXT NOT NULL,
    execution_id TEXT REFERENCES eval_execution(id),
    prompt_hash TEXT NOT NULL,
    prompt_text TEXT NOT NULL,
    model TEXT NOT NULL,
    temperature REAL NOT NULL,
    response TEXT NOT NULL,
    tokens_in INTEGER NOT NULL,
    tokens_out INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'completed',
    run_number INTEGER DEFAULT 1,
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,

    -- 用于跨模型/跨时间去重比较
    UNIQUE(eval_case_id, prompt_hash, model, temperature, created_at)
);

CREATE INDEX idx_run_case ON eval_run(eval_case_id);
CREATE INDEX idx_run_prompt_hash ON eval_run(prompt_hash);
CREATE INDEX idx_run_model ON eval_run(model);
CREATE INDEX idx_run_created ON eval_run(created_at);

-- Evaluation 表（评分结果）
CREATE TABLE eval_evaluation (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES eval_run(id),
    rubric_id TEXT NOT NULL REFERENCES eval_rubric(id),
    score INTEGER NOT NULL,
    passed BOOLEAN NOT NULL,
    deterministic_score REAL,
    rubric_score INTEGER,
    details TEXT,  -- JSON array of CheckResultSnapshot
    evaluated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- 同一 Run 用同一 Rubric 只能评一次
    UNIQUE(run_id, rubric_id)
);

CREATE INDEX idx_eval_run ON eval_evaluation(run_id);
CREATE INDEX idx_eval_rubric ON eval_evaluation(rubric_id);

-- Eval Execution 表
CREATE TABLE eval_execution (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL,
    snapshot_id TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode IN ('single', 'batch', 'matrix')),
    runs_per_case INTEGER DEFAULT 1,
    case_ids TEXT NOT NULL,  -- JSON array
    total_runs INTEGER NOT NULL,
    completed_runs INTEGER DEFAULT 0,
    failed_runs INTEGER DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    concurrency INTEGER DEFAULT 1,
    model TEXT,
    temperature REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE INDEX idx_execution_asset ON eval_execution(asset_id);
CREATE INDEX idx_execution_status ON eval_execution(status);
CREATE INDEX idx_execution_created ON eval_execution(created_at);

-- Trace Event 表（可选，用于 Timeline）
CREATE TABLE trace_event (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES eval_run(id),
    span_id TEXT NOT NULL,
    parent_id TEXT,
    name TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    type TEXT NOT NULL,
    phase TEXT,
    data TEXT,  -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trace_run ON trace_event(run_id);
```

### 8.2 迁移路径

```sql
-- 从现有 eval_run 表迁移
-- 1. 添加新字段
ALTER TABLE eval_run ADD COLUMN response TEXT;
ALTER TABLE eval_run ADD COLUMN prompt_hash TEXT;
ALTER TABLE eval_run ADD COLUMN prompt_text TEXT;
ALTER TABLE eval_run ADD COLUMN tokens_in INTEGER DEFAULT 0;
ALTER TABLE eval_run ADD COLUMN tokens_out INTEGER DEFAULT 0;

-- 2. 将现有的 deterministic_score, rubric_score 等数据迁移到新的 evaluation 表
-- 3. 保留旧的 eval_run 表作为历史兼容
```

---

## 九、分析能力

### 9.1 跨 Rubric 版本对比

```sql
-- 分析 Rubric 变更对历史评分的影响
SELECT
    r.id as run_id,
    e_old.score as score_v1,
    e_new.score as score_v3,
    e_new.score - e_old.score as delta
FROM eval_runs r
JOIN eval_evaluation e_old ON r.id = e_old.run_id
    AND e_old.rubric_id = 'rubric-v1'
JOIN eval_evaluation e_new ON r.id = e_new.run_id
    AND e_new.rubric_id = 'rubric-v3'
WHERE r.eval_case_id = 'case-sql-injection'
ORDER BY delta DESC;
```

### 9.2 跨模型对比

```sql
-- 同一 prompt 在不同模型上的表现
SELECT
    r.prompt_hash,
    r.model,
    e.score,
    r.tokens_in,
    r.tokens_out,
    r.response
FROM eval_runs r
JOIN eval_evaluation e ON r.id = e.run_id
WHERE r.prompt_hash IN (
    SELECT prompt_hash FROM eval_runs
    GROUP BY prompt_hash
    HAVING COUNT(DISTINCT model) > 1
)
ORDER BY r.prompt_hash, r.model;
```

### 9.3 稳定性分析

```sql
-- 检测得分波动异常的用例
SELECT
    ec.name as case_name,
    AVG(e.score) as avg_score,
    STDDEV(e.score) as std_dev,
    MIN(e.score) as min_score,
    MAX(e.score) as max_score
FROM eval_runs r
JOIN eval_evaluation e ON r.id = e.run_id
JOIN eval_case ec ON r.eval_case_id = ec.id
WHERE e.rubric_id = ec.active_rubric_id
    AND r.created_at > datetime('now', '-7 days')
GROUP BY ec.id
HAVING std_dev > 10
ORDER BY std_dev DESC;
```

---

## 十、实现优先级

| 优先级 | 功能 | 工作量 | 说明 |
|--------|------|--------|------|
| **P0** | | | |
| P0-1 | Rubric 版本化管理 | 中 | 核心数据结构变更 |
| P0-2 | Run = LLM Call 原子化 | 高 | 数据模型重构 |
| P0-3 | Re-evaluation API | 中 | 核心差异化功能 |
| **P1** | | | |
| P1-1 | Eval Execution API | 中 | 执行模式支持 |
| P1-2 | 前端 Eval Cases + Rubric 列表 | 高 | 核心 UI |
| P1-3 | Evaluation 跨版本展示 | 中 | UI 差异化 |
| **P2** | | | |
| P2-1 | Batch/Matrix 执行 | 中 | 执行模式 |
| P2-2 | 并发执行 | 高 | 性能优化 |
| P2-3 | Trace Timeline | 中 | 可视化 |
| **P3** | | | |
| P3-1 | 分析 API | 高 | 数据挖掘 |
| P3-2 | Rubric 对比分析 | 中 | 功能增强 |

---

## 十一、执行引擎

### 11.1 架构概览

执行引擎负责协调 Eval 的实际执行过程，支持并发执行和取消。

```
┌─────────────────────────────────────────────────────────────────┐
│                    Execution Coordinator                          │
│                                                                 │
│  - 管理 Execution 生命周期                                       │
│  - 分配 Run 到 Worker                                          │
│  - 聚合 Results                                               │
│  - 处理取消信号                                                │
│  - 维护进度状态                                                │
└───────────────────────────┬─────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
   ┌────────┐         ┌────────┐         ┌────────┐
   │Worker 1│         │Worker 2│         │Worker 3│
   │        │         │        │         │        │
   │  ctx  │         │  ctx  │         │  ctx  │
   │ cancel │         │ cancel │         │ cancel │
   └────┬───┘         └────┬───┘         └────┬───┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
                            ▼
                  ┌─────────────────┐
                  │   Result Chan    │
                  │  (RunResult)    │
                  └─────────────────┘
```

### 11.2 核心组件

#### RunContext（共享上下文）

```go
// RunContext 在所有 Worker 之间共享，包含取消信号
type RunContext struct {
    ctx    context.Context
    cancel context.CancelFunc

    mu          sync.RWMutex
    status      ExecutionStatus
    completed   int
    failed      int
    total       int

    cancelCh chan struct{}  // 关闭时表示已取消
}

func NewRunContext() *RunContext {
    ctx, cancel := context.WithCancel(context.Background())
    return &RunContext{
        ctx:      ctx,
        cancel:   cancel,
        cancelCh: make(chan struct{}),
        status:   StatusPending,
    }
}

func (rc *RunContext) IsCancelled() bool {
    select {
    case <-rc.cancelCh:
        return true
    default:
        return false
    }
}

func (rc *RunContext) Cancel() {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    rc.cancel()
    close(rc.cancelCh)
    rc.status = StatusCancelled
}

func (rc *RunContext) UpdateProgress(completed, failed int) {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    rc.completed = completed
    rc.failed = failed

    if completed+failed >= rc.total {
        if failed == 0 {
            rc.status = StatusCompleted
        } else if completed > 0 {
            rc.status = StatusPartialFailure
        } else {
            rc.status = StatusFailed
        }
    }
}
```

#### Worker（执行单元）

```go
type Worker struct {
    id    int
    coord *Coordinator
    llm   service.LLMInvoker
    runner service.EvalRunner
}

func (w *Worker) Run(ctx context.Context, case *EvalCase, runNum int) (*RunResult, error) {
    run := &RunResult{
        EvalCaseID: case.ID,
        RunNumber:  runNum,
        Status:     RunStatusRunning,
    }

    // 每个步骤前检查取消
    if w.checkCancelled(ctx) {
        return w.markCancelled(run), nil
    }

    // 1. Render Prompt
    prompt, err := w.renderPrompt(ctx, case, runNum)
    if err != nil {
        return w.markFailed(run, err), nil
    }
    run.PromptText = prompt
    run.PromptHash = hashPrompt(prompt)

    // 2. 幂等检查：相同 prompt+model+temperature 是否已有 Run
    if existing := w.findExistingRun(run); existing != nil {
        return w.createEvaluation(existing, case), nil
    }

    if w.checkCancelled(ctx) {
        return w.markCancelled(run), nil
    }

    // 3. LLM Call
    resp, err := w.llm.Invoke(ctx, prompt, w.coord.execution.Model, w.coord.execution.Temperature)
    if err != nil {
        return w.markFailed(run, err), nil
    }
    run.Response = resp.Content
    run.TokensIn = resp.TokensIn
    run.TokensOut = resp.TokensOut
    run.DurationMs = resp.DurationMs

    // 4. 保存 Run
    if err := w.saveRun(run); err != nil {
        return w.markFailed(run, err), nil
    }

    // 5. 创建 Evaluation
    return w.createEvaluation(run, case), nil
}

func (w *Worker) checkCancelled(ctx context.Context) bool {
    select {
    case <-ctx.Done():
        return true
    case <-w.coord.runCtx.cancelCh:
        return true
    default:
        return false
    }
}
```

#### Coordinator（协调器）

```go
type Coordinator struct {
    execution *EvalExecution
    workers  int
    runCtx  *RunContext

    results   chan *RunResult
    wg        sync.WaitGroup

    runRepo  *repository.RunRepository
    evalRepo *repository.EvaluationRepository
    caseRepo *repository.CaseRepository
    llm      service.LLMInvoker
    runner   service.EvalRunner
}

func (c *Coordinator) Execute(ctx context.Context) error {
    c.runCtx.status = StatusRunning
    c.runCtx.startedAt = time.Now()

    // 创建 Work 通道（带缓冲）
    workCh := make(chan *WorkItem, c.execution.TotalRuns)

    // 启动 Worker Pool
    for i := 0; i < c.workers; i++ {
        w := &Worker{id: i, coord: c, llm: c.llm, runner: c.runner}
        c.wg.Add(1)
        go func() {
            defer c.wg.Done()
            w.runLoop(ctx, workCh)
        }()
    }

    // 生产 Work
    go func() {
        for _, caseID := range c.execution.CaseIDs {
            for runNum := 1; runNum <= c.execution.RunsPerCase; runNum++ {
                workCh <- &WorkItem{CaseID: caseID, RunNumber: runNum}
            }
        }
        close(workCh)
    }()

    // 收集结果
    go c.collectResults()

    // 等待所有 Worker 完成
    c.wg.Wait()
    c.finalize()

    return nil
}

func (c *Coordinator) Cancel() error {
    c.runCtx.Cancel()
    return nil
}

func (c *Coordinator) collectResults() {
    completed := 0
    failed := 0

    for result := range c.results {
        if result.Status == RunStatusCompleted {
            completed++
        } else {
            failed++
        }

        c.runCtx.UpdateProgress(completed, failed)

        // 更新 Execution 进度
        c.updateExecutionProgress(completed, failed)
    }
}
```

### 11.3 幂等性保证

相同 `prompt_hash + model + temperature` 的 Run 不会重复执行。

```go
func (w *Worker) findExistingRun(run *RunResult) *Run {
    runs, err := w.runRepo.GetByHash(run.PromptHash, w.coord.execution.Model)
    if err != nil || len(runs) == 0 {
        return nil
    }

    // 找到完全匹配的 Run
    for _, r := range runs {
        if r.Temperature == w.coord.execution.Temperature &&
           r.EvalCaseID == run.EvalCaseID &&
           r.RunNumber == run.RunNumber &&
           r.Status == RunStatusCompleted {
            return r
        }
    }

    return nil
}
```

### 11.4 执行流程状态机

```
                    Coordinator 状态机
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  ┌─────────┐                                     │
    │  │pending  │ ◀── Create                           │
    │  └────┬────┘                                     │
    │       │ Start()                                    │
    │       ▼                                            │
    │  ┌─────────┐     所有 Worker 完成                  │
    │  │running  │ ───────────────────────────────┐     │
    │  └────┬────┘                              │     │
    │       │ Worker 完成                        │     │
    │       ▼                                     ▼     ▼
    │  ┌───────────┐              ┌─────────────────────┐
    │  │ collecting │              │      completed      │
    │  └─────┬─────┘              └─────────────────────┘
    │        │                                             ▲
    │        │ results channel 关闭                       │
    │        ▼                                             │
    │  ┌─────────────┐                                    │
    │  │  finalizing │ ──────────────────────────────────┘
    │  └─────────────┘
    │
    │  ┌─────────┐
    └──▶│cancelled│ ◀── Cancel()
        └─────────┘
    │
    │  ┌─────────┐
    └──▶│  failed │ ◀── 全部 Worker 失败
        └─────────┘
    │
    └─────────────────────────────────────────────────────┘
```

---

## 十二、前端状态管理

### 12.1 Zustand Store 结构

```typescript
// stores/evalStore.ts

interface EvalState {
  // ===== Cases =====
  cases: EvalCase[]
  casesLoading: boolean
  selectedCaseIds: Set<string>

  // ===== Rubrics =====
  rubrics: Record<string, Rubric[]>           // caseId -> rubrics
  activeRubricVersions: Record<string, string> // caseId -> active rubricId

  // ===== Runs =====
  runs: Record<string, Run[]>                // caseId -> runs
  selectedRunId: string | null

  // ===== Evaluations =====
  evaluations: Record<string, Evaluation[]>   // key = "runId:rubricId"

  // ===== Execution =====
  currentExecution: Execution | null
  executionProgress: Record<string, number>   // executionId -> progress %

  // ===== UI State =====
  runEvalModalOpen: boolean
  runEvalConfig: RunEvalConfig
  rubricVersionModalOpen: boolean
  selectedCaseForRubric: string | null

  // ===== Actions =====
  loadCases: (assetId: string) => Promise<void>
  createCase: (assetId: string, data: CreateCaseRequest) => Promise<void>
  updateCase: (caseId: string, data: UpdateCaseRequest) => Promise<void>
  deleteCase: (caseId: string) => Promise<void>
  selectCase: (caseId: string, selected: boolean) => void
  selectAllCases: (selected: boolean) => void

  loadRubrics: (caseId: string) => Promise<void>
  createRubric: (caseId: string, data: CreateRubricRequest) => Promise<void>
  setActiveRubric: (caseId: string, rubricId: string) => Promise<void>

  loadRuns: (caseId: string) => Promise<void>
  loadEvaluations: (runId: string) => Promise<void>

  openRunEvalModal: () => void
  closeRunEvalModal: () => void
  setRunEvalConfig: (config: Partial<RunEvalConfig>) => void
  executeEval: (config: ExecuteEvalRequest) => Promise<string>
  pollExecution: (executionId: string) => void
  cancelExecution: (executionId: string) => Promise<void>

  reevaluate: (runId: string, rubricId: string) => Promise<void>
  batchReevaluate: (caseId: string, toRubricId: string) => Promise<void>
}

interface RunEvalConfig {
  mode: 'single' | 'batch' | 'matrix'
  runsPerCase: number
  concurrency: number
  model: string
  temperature: number
}

interface ExecuteEvalRequest {
  asset_id: string
  case_ids: string[]
  mode: ExecutionMode
  runs_per_case?: number
  concurrency?: number
  model?: string
  temperature?: number
}
```

### 12.2 乐观更新

执行开始时立即反映 UI，不等待 API 响应。

```typescript
// stores/evalStore.ts

async function executeEval(config: ExecuteEvalRequest): Promise<string> {
  const tempId = `temp-${Date.now()}`

  // 1. 乐观更新：立即显示执行开始
  set(state => ({
    currentExecution: {
      id: tempId,
      status: 'running',
      mode: config.mode,
      total_runs: calculateTotalRuns(config),
      completed_runs: 0,
      ...config
    },
    executionProgress: {
      ...state.executionProgress,
      [tempId]: 0
    }
  }))

  try {
    // 2. 调用 API
    const response = await evalApi.execute(config)

    // 3. 替换为真实 ID
    set(state => ({
      currentExecution: state.currentExecution
        ? { ...state.currentExecution, id: response.execution_id }
        : null,
    }))

    // 4. 开始轮询
    this.pollExecution(response.execution_id)

    return response.execution_id
  } catch (error) {
    // 5. 回滚
    set(state => ({
      currentExecution: null,
      executionProgress: {
        ...state.executionProgress,
        [tempId]: undefined,
      }
    }))
    throw error
  }
}

// 轮询更新进度
function pollExecution(executionId: string) {
  const poll = async () => {
    const status = await evalApi.getExecution(executionId)

    set(state => ({
      currentExecution: status,
      executionProgress: {
        ...state.executionProgress,
        [executionId]: calculateProgress(status),
      }
    }))

    if (status.status === 'running') {
      setTimeout(poll, 2000)
    } else if (status.status === 'completed' || status.status === 'failed') {
      // 执行结束，刷新数据
      refreshRunsAndEvaluations()
    }
  }

  poll()
}
```

### 12.3 数据获取策略

```typescript
// 路由切换时获取数据
function useEvalData(assetId: string) {
  const { cases, loadCases } = useEvalStore()

  useEffect(() => {
    loadCases(assetId)
  }, [assetId])

  // 选中 case 时自动加载 rubrics 和 runs
  const selectedCases = cases.filter(c => isSelected(c.id))

  useEffect(() => {
    for (const c of selectedCases) {
      loadRubrics(c.id)
      loadRuns(c.id)
    }
  }, [selectedCases.map(c => c.id).join(',')])

  // 选中 run 时加载 evaluations
  const selectedRun = runs.find(r => r.id === selectedRunId)

  useEffect(() => {
    if (selectedRunId) {
      loadEvaluations(selectedRunId)
    }
  }, [selectedRunId])
}
```

### 12.4 UI 组件使用示例

```tsx
// components/EvalPanel.tsx

function EvalPanel() {
  const {
    cases,
    selectedCaseIds,
    runs,
    currentExecution,
    loadCases,
    selectCase,
    openRunEvalModal,
    executeEval,
    selectedRunId,
  } = useEvalStore()

  const hasSelection = selectedCaseIds.size > 0

  return (
    <div>
      {/* Case 列表 */}
      <CaseList
        cases={cases}
        selectedIds={selectedCaseIds}
        onSelect={(id) => selectCase(id, !selectedCaseIds.has(id))}
      />

      {/* Run Eval 按钮 */}
      <Button
        type="primary"
        disabled={!hasSelection}
        onClick={openRunEvalModal}
      >
        Run Eval ({selectedCaseIds.size})
      </Button>

      {/* 执行进度（执行中时显示） */}
      {currentExecution?.status === 'running' && (
        <ExecutionProgress execution={currentExecution} />
      )}

      {/* Run 列表 */}
      <RunList runs={runs} selectedId={selectedRunId} />
    </div>
  )
}

// components/RunEvalModal.tsx

function RunEvalModal() {
  const {
    runEvalModalOpen,
    runEvalConfig,
    closeRunEvalModal,
    setRunEvalConfig,
    executeEval,
  } = useEvalStore()

  const handleConfirm = async () => {
    await executeEval({
      asset_id: currentAssetId,
      case_ids: Array.from(selectedCaseIds),
      ...runEvalConfig,
    })
    closeRunEvalModal()
  }

  return (
    <Modal open={runEvalModalOpen} onCancel={closeRunEvalModal}>
      <Select
        value={runEvalConfig.mode}
        onChange={(mode) => setRunEvalConfig({ mode })}
      >
        <Option value="single">Single</Option>
        <Option value="batch">Batch</Option>
        <Option value="matrix">Matrix</Option>
      </Select>

      {runEvalConfig.mode === 'matrix' && (
        <NumberInput
          label="Runs per case"
          value={runEvalConfig.runsPerCase}
          onChange={(v) => setRunEvalConfig({ runsPerCase: v })}
        />
      )}

      <Button onClick={handleConfirm}>Run</Button>
    </Modal>
  )
}
```

---

## 十三、存储策略

### 13.1 分层存储

数据都在文件系统（SQLite + JSONL），不需要传统意义上的"归档"。但随着数据增长，需要分层管理以保证查询性能。

```
┌─────────────────────────────────────────────────────────┐
│                    分层存储策略                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  热数据 (主 SQLite)                                     │
│  ────────────────────                                   │
│  - 最近 30 天的 Runs                                   │
│  - 活跃 Case 的 Rubric                                  │
│  - 未完成的 Execution                                   │
│  → 高频查询，需要优化索引                               │
│                                                          │
│  温数据 (归档 SQLite)                                   │
│  ──────────────────────                                 │
│  - 30-90 天的 Runs                                     │
│  - 按 Asset 分区（多个小库）                            │
│  → 低频查询，只读                                        │
│                                                          │
│  历史数据 (压缩 JSONL)                                  │
│  ──────────────────────                                 │
│  - 90 天以上的 Trace JSONL                              │
│  - gzip 压缩                                           │
│  → 审计用，保留但不常查询                               │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 13.2 自动归档

```go
// storage/archive_manager.go

type ArchiveManager struct {
    hotDB  *Client
    config ArchiveConfig
}

const (
    HotDays  = 30
    WarmDays = 90
)

type ArchiveConfig struct {
    HotDays:             30,
    WarmDays:            90,
    TraceCompressAfter:  7,  // 天
    ArchiveEnabled:      true,
}

// 每日维护任务
func (am *ArchiveManager) RunMaintenance(ctx context.Context) error {
    if !am.config.ArchiveEnabled {
        return nil
    }

    // 1. 归档旧 Runs
    if err := am.archiveOldRuns(ctx); err != nil {
        slog.Error("failed to archive runs", "error", err)
    }

    // 2. 压缩旧 Trace 文件
    if err := am.compressOldTraces(ctx); err != nil {
        slog.Error("failed to compress traces", "error", err)
    }

    // 3. 清理过期数据
    if err := am.purgeExpiredData(ctx); err != nil {
        slog.Error("failed to purge data", "error", err)
    }

    return nil
}

// 分区策略：按 Asset 分区
func (am *ArchiveManager) getArchiveDB(assetID string) (*Client, error) {
    archiveName := fmt.Sprintf("archive_%s.db", assetID)
    path := filepath.Join(am.archiveDir, archiveName)

    if _, err := os.Stat(path); os.IsNotExist(err) {
        return am.createArchiveDB(path)
    }

    return am.openArchiveDB(path)
}
```

### 13.3 用户配置

```yaml
# config.yaml
eval_storage:
  hot_days: 30
  warm_days: 90
  archive_enabled: true
  trace_compress_after_days: 7
```

---

## 十四、成本后置计算

### 14.1 设计原则

**不实时计算，事后聚合**。Token 计数在 Run 保存时记录，成本计算在分析时按需进行。

### 14.2 Token 记录

```go
// Run 保存时记录 token 数量
type Run struct {
    // ...
    TokensIn  int `json:"tokens_in"`
    TokensOut int `json:"tokens_out"`
}
```

### 14.3 成本计算公式

```typescript
interface TokenPricing {
  model: string
  inputPer1M: number   // $/1M tokens
  outputPer1M: number
}

const PRICING: Record<string, TokenPricing> = {
  'gpt-4o': { inputPer1M: 5.0, outputPer1M: 15.0 },
  'gpt-4o-mini': { inputPer1M: 0.15, outputPer1M: 0.6 },
  'claude-3-5-sonnet': { inputPer1M: 3.0, outputPer1M: 15.0 },
}

function calculateCost(tokensIn: number, tokensOut: number, model: string): number {
  const pricing = PRICING[model]
  if (!pricing) return 0

  return (tokensIn / 1_000_000) * pricing.inputPer1M +
         (tokensOut / 1_000_000) * pricing.outputPer1M
}
```

### 14.4 按需计算

```typescript
// 分析页面：按资产汇总成本
async function getCostSummary(assetId: string, from: Date, to: Date) {
  const runs = await evalApi.listRuns({ asset_id: assetId, from, to })

  const byModel = runs.reduce((acc, run) => {
    const cost = calculateCost(run.tokens_in, run.tokens_out, run.model)
    acc[run.model] = (acc[run.model] || 0) + cost
    return acc
  }, {} as Record<string, number>)

  return {
    total: Object.values(byModel).reduce((a, b) => a + b, 0),
    byModel,
    totalTokensIn: runs.reduce((a, r) => a + r.tokens_in, 0),
    totalTokensOut: runs.reduce((a, r) => a + r.tokens_out, 0),
  }
}
```

---

## 十五、风险与备选

### 15.1 数据迁移

**风险**：现有 eval_run 表结构和新的 Run 模型不兼容。

**方案**：
1. 保留旧表，新增新表
2. 提供迁移脚本，将历史数据的 deterministic_score/rubric_score 转为第一条 Evaluation
3. 迁移是可选的，不影响现有功能

### 15.2 存储成本

**风险**：每个 Run 永久保留完整 LLM Response，存储成本增长。

**方案**：
1. 分层存储：热数据在主库，温数据在归档库
2. Trace 压缩：gzip 压缩历史 JSONL
3. 可配置保留策略

### 15.3 Rubric 删除限制

**风险**：Rubric 被删除后，历史 Evaluation 失去参考。

**方案**：
1. Rubric 不删除，只标记为 deprecated
2. 删除时检查是否有 Evaluation 引用
3. 强制删除需要二次确认
