# Eval 存储架构设计

**版本**: V1.1
**状态**: 设计中
**日期**: 2026-04-26
**目标读者**: 开发团队、架构评审

---

## 核心决策

**Eval 相关数据全部存储在文件系统，SQLite 仅保留 Asset 元数据索引（id, name, description, tags, content_hash）。**

> ⚠️ **状态模型待更新**：INDEX-ARCHITECTURE.md 中的二分法状态（ACTIVE/ARCHIVED）已过时，新的状态模型待设计。本文档中 `state` 字段仅供参考，需与新的状态设计保持一致。

---

## 概念模型

| 概念 | 实体 | category | 说明 |
|------|------|----------|------|
| 被测 Prompt | Asset | `content` | 待评测的 prompt |
| 评测集 | Asset | `eval` | 包含测试用例和 Metric 引用 |
| 评价标准 | Asset | `metric` | 评分标准，可复用 |
| 执行批次 | Execution | - | 批次元数据 |
| 调用记录 | WorkItem/Call | - | LLM 单次调用 |

> **关于无限递归**：用户可以对任何 category 的 Asset 做 eval。是否产生无限递归、是否有意义，取决于用户场景。文档暂不预设约束，观察实际使用后再评估。

---

## 一、数据分类与存储策略

### 1.1 数据分类

| 数据类型 | 存储位置 | Git 同步 | 说明 |
|----------|----------|---------|------|
| **Asset 索引** | SQLite | 否 | 仅索引字段：id, name, description, tags, content_hash |
| **Asset 文件** | prompts/*.md | 是 | content/eval/metric 三种类型 |
| **Execution 执行记录** | .evals/executions/*.json | 是 | 批次元数据 |
| **LLM 调用记录** | .evals/calls/*/calls.jsonl | 是 | 完整调用详情，用于分析/计费/训练 |

### 1.2 为什么这样分

- **SQLite 不进 Git**：无法做 diff，团队协作困难
- **LLM 调用记录要共享**：用于分析、计费、甚至模型训练
- **Execution 要共享**：团队需要知道谁在什么时间跑了什么 eval
- **eval_history 在 frontmatter**：Git 同步，完整历史

---

## 二、文件结构

```
.
├── prompts/                    # Asset 文件（Git 同步）
│   ├── code-review.md         # 被测 prompt，eval_history 在 frontmatter
│   ├── code-review-eval.md    # 评测集（category: eval）
│   └── code-quality-metric.md # 评价标准（category: metric）
│
├── .evals/                    # Execution 执行记录（Git 同步）
│   ├── executions/
│   │   ├── exec_01AR3C.json   # Execution 元数据
│   │   └── exec_01AR3D.json
│   │
│   └── calls/
│       ├── exec_01AR3C/
│       │   └── calls.jsonl    # 该 execution 的所有 LLM 调用
│       └── exec_01AR3D/
│           └── calls.jsonl
```

> **说明**：评测集和评价标准都是 Asset，只是 `category` 不同。评测集通过 frontmatter 中的 `metric_refs` 引用评价标准。

---

## 三、文件格式

### 3.1 Execution 元数据

文件：`.evals/executions/{execution_id}.json`

```json
{
  "id": "exec_01AR3C",
  "asset_id": "code-review",
  "snapshot_id": "snap_01B2K9",
  "mode": "batch",
  "status": "completed",
  "total_runs": 10,
  "completed_runs": 10,
  "failed_runs": 0,
  "cancelled_runs": 0,
  "concurrency": 3,
  "model": "gpt-4o-2024-05-13",
  "temperature": 0.7,
  "runs_per_case": 1,
  "created_at": "2026-04-26T10:00:00Z",
  "started_at": "2026-04-26T10:00:05Z",
  "completed_at": "2026-04-26T10:15:30Z"
}
```

**status 枚举**：
- `pending` - 已创建，未开始
- `running` - 执行中
- `completed` - 全部完成
- `partial_failure` - 部分失败
- `failed` - 全部失败
- `cancelled` - 已取消

### 3.2 LLM 调用记录

文件：`.evals/calls/{execution_id}/calls.jsonl`

格式：JSON Lines（每行一条记录）

```jsonl
{"run_id":"run_001","execution_id":"exec_01AR3C","asset_id":"code-review","snapshot_id":"snap_01B2K9","case_id":"case-001","run_number":1,"status":"completed","model":"gpt-4o-2024-05-13","temperature":0.7,"tokens_in":1500,"tokens_out":350,"latency_ms":1200,"response_content":"模型的原始回答内容...","error":"","timestamp":"2026-04-26T10:00:30Z"}
{"run_id":"run_002","execution_id":"exec_01AR3C","asset_id":"code-review","snapshot_id":"snap_01B2K9","case_id":"case-002","run_number":2,"status":"completed","model":"gpt-4o-2024-05-13","temperature":0.7,"tokens_in":1200,"tokens_out":280,"latency_ms":980,"response_content":"模型的原始回答内容...","error":"","timestamp":"2026-04-26T10:00:35Z"}
{"run_id":"run_003","execution_id":"exec_01AR3C","asset_id":"code-review","snapshot_id":"snap_01B2K9","case_id":"case-001","run_number":1,"status":"failed","model":"gpt-4o-2024-05-13","temperature":0.7,"tokens_in":1500,"tokens_out":0,"latency_ms":0,"response_content":"","error":"rate_limit_exceeded","timestamp":"2026-04-26T10:00:40Z"}
```

**字段说明**：
- `run_id` - 唯一标识该次调用
- `execution_id` - 所属 Execution ID
- `asset_id` - 被测 Asset ID（用于索引关系）
- `snapshot_id` - 被测 Asset 的快照版本
- `case_id` - 使用的 EvalCase ID
- `run_number` - 在该 case 中的第几次运行
- `status` - completed / failed / cancelled
- `model` - 使用的模型
- `temperature` - 温度参数
- `tokens_in/out` - Token 消耗
- `latency_ms` - 延迟
- `response_content` - **LLM 返回的原始内容**（去包装），Git 同步
- `error` - 错误信息

**本地私有（SQLite）**：完整 response（含 usage、id 等元数据）

> **注意**：`response_content` 是 Git 同步的 LLM 回答原文，用于团队共享和后续分析。完整原始 response 存本地 SQLite，不进 Git。

### 3.3 Asset frontmatter 中的 eval_history

文件：`prompts/{asset_id}.md`

```yaml
---
id: code-review
name: Code Review Prompt
version: v1.2.3
content_hash: sha256:abc123...
state: active
tags: [go, review]
eval_history:
  - run_id: run-001
    snapshot_id: snap-001
    score: 85
    deterministic_score: 0.92
    rubric_score: 85
    model: gpt-4o-2024-05-13
    eval_case_version: v2.3
    tokens_in: 1500
    tokens_out: 350
    duration_ms: 1200
    date: 2026-04-25
    by: alice
  - run_id: run-002
    snapshot_id: snap-001
    score: 78
    deterministic_score: 0.85
    rubric_score: 78
    model: gpt-4o-2024-05-13
    eval_case_version: v2.3
    tokens_in: 1200
    tokens_out: 280
    duration_ms: 980
    date: 2026-04-26
    by: bob
eval_stats:
  gpt-4o-2024-05-13:
    count: 2
    mean: 81.5
    m2: 24.5
    min: 78
    max: 85
    last_run: 2026-04-26
---
# Prompt Content

你是一位 Go 开发专家...
```

### 3.4 EvalCase（评测集）

文件：`prompts/{case_id}.md`，`category: eval`

```yaml
---
id: code-review-case
name: Code Review Eval Case
version: v2.3
content_hash: sha256:xyz789...
state: active
category: eval
tags: [go, review]
model: gpt-4o-2024-05-13
metric_refs:
  - metric_id: code-quality-metric
    version: v1.0
test_cases:
  - id: tc-001
    name: 简单函数评审
    input:
      language: go
      code: |
        func Add(a, b int) int {
            return a + b
        }
    # 确定性任务的参考答案（可选）
    expected:
      score: 90
      content: 应正确实现加法运算
    # 开放式任务的评分标准（可选，可同时存在）
    rubric:
      - check: 正确性
        weight: 0.6
        criteria: 函数返回值是否正确
      - check: 风格
        weight: 0.4
        criteria: 代码风格是否符合规范
---
# Eval Case Content

## Test Case 1: 简单函数评审

### Input
```go
func Add(a, b int) int {
    return a + b
}
```

**评判规则**：
- `expected`：存在时用比对打分
- `rubric`：存在时用 Metric 评判
- 两者都有时综合评分
- 至少配置一个

### 3.5 Metric（评价标准）

文件：`prompts/{metric_id}.md`，`category: metric`

```yaml
---
id: code-quality-metric
name: Code Quality Metric
version: v1.0
content_hash: sha256:abc123...
state: active
category: metric
description: 代码质量评分标准
rubric:
  - check: correctness
    weight: 0.4
    description: 代码逻辑是否正确
    criteria: |
      - 函数返回值是否正确
      - 边界条件是否处理
  - check: style
    weight: 0.3
    description: 代码风格是否符合规范
    criteria: |
      - 命名规范
      - 注释完整
  - check: security
    weight: 0.3
    description: 安全性检查
    criteria: |
      - SQL 注入风险
      - XSS 风险
---
```

---

## 四、存储服务设计

> **注**：Asset 文件的读写通过 `AssetFileManager`（见 INDEX-ARCHITECTURE.md）统一管理。eval_history / eval_stats 追加到 Asset frontmatter。

### 4.1 ExecutionFileStore

```go
type ExecutionFileStore struct {
    baseDir string
    mu      sync.RWMutex  // 保护并发写入
}

// 保存 execution 元数据
func (s *ExecutionFileStore) Save(ctx context.Context, exec *domain.EvalExecution) error

// 读取单个 execution
func (s *ExecutionFileStore) Get(ctx context.Context, id string) (*domain.EvalExecution, error)

// 按 asset 列出所有 execution
func (s *ExecutionFileStore) ListByAsset(ctx context.Context, assetID string) ([]*domain.EvalExecution, error)

// 列出所有 execution（分页）
func (s *ExecutionFileStore) List(ctx context.Context, offset, limit int) ([]*domain.EvalExecution, int, error)

// 更新 execution 状态
func (s *ExecutionFileStore) UpdateStatus(ctx context.Context, id string, status domain.ExecutionStatus) error

// 更新进度
func (s *ExecutionFileStore) UpdateProgress(ctx context.Context, id string, completed, failed, cancelled int) error

// 归档 execution（预留接口，具体实现见 long-shot）
func (s *ExecutionFileStore) Archive(ctx context.Context, id string) error

// 检查是否已归档
func (s *ExecutionFileStore) IsArchived(ctx context.Context, id string) bool
```

### 4.2 LLMCallFileStore

```go
type LLMCallFileStore struct {
    baseDir string
    mu      sync.Mutex  // 保护同一 execution 的并发写入
}

// 追加单条调用记录
func (s *LLMCallFileStore) Append(ctx context.Context, executionID string, call *LLMCall) error

// 批量追加（用于恢复时）
func (s *LLMCallFileStore) AppendBatch(ctx context.Context, executionID string, calls []*LLMCall) error

// 读取某个 execution 的所有调用
func (s *LLMCallFileStore) ListByExecution(ctx context.Context, executionID string) ([]*LLMCall, error)

// 读取某个 execution 的调用（分页）
func (s *LLMCallFileStore) ListByExecutionPaginated(ctx context.Context, executionID string, offset, limit int) ([]*LLMCall, int, error)

// 获取已完成 run_id 列表（用于中断恢复）
func (s *LLMCallFileStore) GetCompletedRunIDs(ctx context.Context, executionID string) (map[string]bool, error)
```

### 4.3 Asset frontmatter 操作（通过 AssetFileManager）

> **注**：`AssetFileManager`（见 INDEX-ARCHITECTURE.md）提供 `UpdateFrontmatter` 方法用于追加 eval_history / eval_stats。

```go
// 追加 eval 结果到 asset 的 frontmatter
// 使用 AssetFileManager.UpdateFrontmatter() 实现
// entry 中的 eval_history 和 eval_stats 字段会被合并更新
```

**追加流程**：
1. 调用 `AssetFileManager.GetFrontmatter(assetID)` 读取当前 frontmatter
2. 构造新的 `EvalHistoryEntry` 并追加到 `eval_history`
3. 调用 `AssetFileManager.UpdateFrontmatter(assetID, func(fm *FrontMatter) { fm.EvalHistory = newHistory })`
4. Welford 统计在步骤 2 中一并更新（Mean, M2, Min, Max）

---

## 五、EvalService 修改

### 5.1 修改后的 RunEval 流程

```go
func (s *EvalService) RunEval(ctx context.Context, req *RunEvalRequest) (*domain.EvalExecution, error) {
    // 1. 创建 Execution
    exec := &domain.EvalExecution{
        ID:          ulid.New().String(),
        AssetID:     req.AssetID,
        Status:      domain.ExecutionStatusRunning,
        CreatedAt:   time.Now(),
        // ... 其他字段
    }

    // 2. 保存 Execution 元数据到文件
    if err := s.executionStore.Save(ctx, exec); err != nil {
        return nil, err
    }

    // 3. 初始化 worker pool
    // ...

    // 4. 执行循环
    for _, workItem := range workItems {
        // 4.1 LLM 调用
        resp, err := s.llmInvoker.Invoke(ctx, prompt, req.Model, req.Temperature)

        // 4.2 记录调用
        call := &LLMCall{
            RunID:     workItem.RunID,
            CaseID:    workItem.CaseID,
            Status:    "completed",
            TokensIn:  resp.Usage.InputTokens,
            TokensOut: resp.Usage.OutputTokens,
            LatencyMs: resp.Latency,
            // ...
        }
        s.callStore.Append(ctx, exec.ID, call)

        // 4.3 更新 frontmatter eval_history
        // 使用 AssetFileManager.UpdateFrontmatter() 追加到 eval_history
        // 同时更新 eval_stats（Welford 算法）
    }

    // 5. 更新 Execution 状态
    s.executionStore.UpdateStatus(ctx, exec.ID, domain.ExecutionStatusCompleted)

    return exec, nil
}
```

### 5.2 中断恢复

```go
func (s *EvalService) ResumeExecution(ctx context.Context, executionID string) error {
    // 1. 读取 Execution 状态
    exec, err := s.executionStore.Get(ctx, executionID)
    if err != nil {
        return err
    }

    // 2. 获取已完成的 run_id
    completed, err := s.callStore.GetCompletedRunIDs(ctx, executionID)
    if err != nil {
        return err
    }

    // 3. 过滤出未完成的 work items
    remainingWorkItems := filterByNotCompleted(workItems, completed)

    // 4. 继续执行剩余 work items
    // ...
}
```

---

## 六、需要删除的代码

> **注**：当前无用户，无需考虑迁移。可直接删除废弃代码。

### 6.1 需要删除的文件

| 文件 | 废弃原因 | 替代方案 |
|------|---------|---------|
| `internal/storage/eval_run_repository.go` | EvalRun 改为存 .md frontmatter | 通过 AssetFileManager 读取 frontmatter |
| `internal/storage/eval_case_repository.go` | EvalCase 改为存 .md 文件 | 通过 AssetFileManager 读取 |
| `internal/storage/eval_work_item_repository.go` | WorkItem 改为存 .evals/calls/ | LLMCallFileStore |
| `internal/storage/eval_execution_repository.go` | Execution 改为存 .evals/executions/ | ExecutionFileStore |
| `internal/storage/ent/evalrun*.go` | Ent schema，EvalRun 废弃 | - |
| `internal/storage/ent/evalworkitem*.go` | Ent schema，WorkItem 废弃 | - |
| `internal/storage/ent/evalexecution*.go` | Ent schema，Execution 废弃 | - |
| `internal/storage/ent/schema/evalrun.go` | Ent schema | - |
| `internal/storage/ent/schema/evalworkitem.go` | Ent schema | - |
| `internal/storage/ent/schema/evalexecution.go` | Ent schema | - |

### 6.2 当前废弃标注情况

| 文件 | @Deprecated 标注 |
|------|-----------------|
| `EvalRunRepository` | ✅ 有 |
| `EvalCaseRepository` | ✅ 有 |
| `EvalExecutionRepository` | ❌ 需补充 |
| `EvalWorkItemRepository` | ❌ 需补充 |

### 6.3 删除后需实现的替代

| 废弃 | 替代 |
|------|------|
| EvalRunRepository | 通过 AssetFileManager 读取 frontmatter eval_history |
| EvalExecutionRepository | ExecutionFileStore |
| EvalWorkItemRepository | LLMCallFileStore |

---

## 八、待讨论

1. **~~trace_path 是否需要？~~** → **结论：不需要，SQLite 本地私有存储原始 trace，Git 只存整理后的数据（含 response_content）**
2. **~~JSONL 文件大小控制？~~** → **结论：先不加限制，出问题再分片**
3. **~~Git 体积？~~** → **结论：预留 Archive 接口（IsArchived/Archive），具体归档逻辑放 long-shot**
4. **~~eval_history 追加时机？~~** → **结论：execution 结束时批量写入，calls.jsonl 是完整数据源，中断可重建**
5. **~~Dataset 概念？~~** → **结论：未来需要（复用、版本、权限），当前用 Asset 数组即可，长期放 long-shot**
6. **~~Ground Truth？~~** → **结论：test_cases 同时支持 expected 和 rubric_ref，两者可选但至少配置一个**

---

## 九、文件系统数据分析

> **核心原则**：Git 同步的文件必须包含索引关系字段，方便其他人 clone 后重建本地索引。

### 9.1 文件分类总览

| 文件 | 存储位置 | Git 同步 | 索引关系字段 |
|------|----------|---------|-------------|
| Asset (prompt) | prompts/*.md | ✅ | - |
| Execution | .evals/executions/*.json | ✅ | asset_id |
| LLM Call | .evals/calls/*/calls.jsonl | ✅ | asset_id, snapshot_id, execution_id |
| eval_history | Asset frontmatter | ✅ | - |
| Raw Trace | SQLite | ❌ | - |

### 9.2 Asset (prompts/*.md)

**Git 同步**：✅

**索引关系**：
- 无需指向其他文件的索引字段
- eval_history 追加到 frontmatter

**本地私有（SQLite 索引）**：
- id, name, description, tags, content_hash 等

### 9.3 Execution (.evals/executions/*.json)

**Git 同步**：✅

**索引关系字段**：
```json
{
  "id": "exec_01AR3C",
  "asset_id": "code-review",
  "snapshot_id": "snap_01B2K9",
  "status": "completed"
}
```

**为什么需要 asset_id**：
- 团队成员可以查询某个 Asset 的所有 Execution
- 重建本地索引时知道这次 eval 是对哪个 Asset 做的

**本地私有（SQLite）**：Execution 的详细状态、中断恢复进度

### 9.4 LLM Call (.evals/calls/*/calls.jsonl)

**Git 同步**：✅

**索引关系字段**：
```jsonl
{
  "run_id": "run_001",
  "execution_id": "exec_01AR3C",
  "asset_id": "code-review",
  "snapshot_id": "snap_01B2K9",
  "case_id": "case-001",
  "response_content": "模型的原始回答..."
}
```

**为什么需要这些字段**：
- `asset_id` - 关联被测 Asset
- `snapshot_id` - 关联具体版本
- `execution_id` - 关联所属批次
- `case_id` - 关联使用的 EvalCase
- `response_content` - **Git 同步的 LLM 回答原文**

**本地私有（SQLite）**：
- 完整的原始 response（含 API 元数据、usage、id）
- 中间过程 trace

---

## 十、相关文档

| 文档 | 内容 |
|------|------|
| [INDEX-ARCHITECTURE.md](./INDEX-ARCHITECTURE.md) | 核心架构：数据库是索引，文件系统是存储 |
| [reconcile-design.md](./reconcile-design.md) | Reconcile 算法设计 |
| 本文档 | Eval 执行记录存储设计 |

---

**状态**：待评审
