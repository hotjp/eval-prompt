# Eval 存储实现方案

**日期**: 2026-04-26
**状态**: 待实现
**基于**: `EVAL-STORAGE-DESIGN.md`

---

## Context

`EVAL-STORAGE-DESIGN.md` 定义了三种 Asset category: `content`(被测Prompt)、`eval`(评测集)、`metric`(评价标准)。代码实现存在以下问题:

1. `category` 字段未在代码中实现
2. `eval_history`、`eval_stats` 等字段已存在于 `FrontMatter`，但 API 不返回
3. `RunEval` 执行后没有写 eval_history 到 frontmatter
4. 前端没有根据 category 区分展示和操作

---

## 一、后端修改

### 1.1 domain/frontmatter.go

添加 `Category` 字段:

```go
type FrontMatter struct {
    // ... existing fields ...
    Category string `yaml:"category,omitempty"` // content/eval/metric
}
```

### 1.2 service/interfaces.go

扩展 `AssetDetail`:

```go
type AssetDetail struct {
    // ... existing fields ...
    Category              string                    `json:"category,omitempty"`
    EvalHistory          []domain.EvalHistoryEntry `json:"eval_history,omitempty"`
    EvalStats            domain.EvalStats         `json:"eval_stats,omitempty"`
    Triggers             []domain.TriggerEntry     `json:"triggers,omitempty"`
    TestCases            []domain.TestCase         `json:"test_cases,omitempty"`
    RecommendedSnapshotID string                   `json:"recommended_snapshot_id,omitempty"`
    Labels               []LabelInfo              `json:"labels,omitempty"` // 修复: 原来设为 nil
}
```

扩展 `AssetResponse` (在 handlers 中定义):

```go
type AssetResponse struct {
    // ... existing fields ...
    Category              string             `json:"category,omitempty"`
    EvalHistory           []EvalHistoryEntry `json:"eval_history,omitempty"`
    EvalStats             EvalStats         `json:"eval_stats,omitempty"`
    Triggers              []TriggerEntry    `json:"triggers,omitempty"`
    TestCases             []TestCase        `json:"test_cases,omitempty"`
    RecommendedSnapshotID string            `json:"recommended_snapshot_id,omitempty"`
}
```

修改 `GetAsset` handler，将上述字段从 detail 填充到 response。

### 1.3 plugins/search/search.go

#### reconcileFile() - 填充新字段

```go
i.assets[fm.ID] = &assetEntry{
    detail: &service.AssetDetail{
        // ... existing fields ...
        Category:              fm.Category,
        EvalHistory:          fm.EvalHistory,
        EvalStats:            fm.EvalStats,
        Triggers:             fm.Triggers,
        TestCases:            fm.TestCases,
        RecommendedSnapshotID: fm.RecommendedSnapshotID,
        Labels:               parseLabels(fm.Labels), // 新增 parseLabels 函数
    },
}
```

需要新增 `parseLabels()` 辅助函数，将 `[]domain.LabelEntry` 转换为 `[]service.LabelInfo`。

#### persist()/Load() - 持久化新字段

`persistEntry` 结构体需要添加 `Category`、`EvalHistory`、`EvalStats`、`Triggers`、`TestCases`、`RecommendedSnapshotID` 字段，并在 Load 时恢复。

#### CreatePlaceholder() - 添加 category

```go
func (i *Indexer) CreatePlaceholder(ctx context.Context, id, name, bizLine string, tags []string, category string) error {
    fm := &domain.FrontMatter{
        // ...
        Category: category,
    }
}
```

### 1.4 service/eval_service.go

#### 添加 AssetFileManager 依赖

```go
type EvalService struct {
    // ... existing fields ...
    fileManager AssetFileManager
}

func (s *EvalService) WithFileManager(fileManager AssetFileManager) *EvalService {
    s.fileManager = fileManager
    return s
}
```

注: 需要在 `service/interfaces.go` 中定义 `AssetFileManager` 接口（已存在于 `asset_file.go`，需提升到 interfaces.go）。

#### RunEval 结束时写 eval_history

在 `return exec, nil` 之前添加:

```go
if err := s.updateAssetEvalHistory(ctx, exec); err != nil {
    slog.Warn("failed to update asset eval_history", "error", err)
}
```

#### 新增 updateAssetEvalHistory 方法

```go
func (s *EvalService) updateAssetEvalHistory(ctx context.Context, exec *domain.EvalExecution) error {
    // 1. 从 callStore.ListByExecution() 获取所有 call 记录
    calls, err := s.callStore.ListByExecution(ctx, exec.ID)
    if err != nil {
        return err
    }

    // 2. 按 case 分组，计算每 case 的平均分
    // 注意: 当前 RunEval 没有评分逻辑，LLMCall 只有 response_content
    // 评分需要后续 eval_runner 补充

    // 3. 读取 asset frontmatter
    fm, err := s.fileManager.GetFrontmatter(ctx, exec.AssetID)
    if err != nil {
        return err
    }

    // 4. 构造 EvalHistoryEntry（评分逻辑待补充）
    // entry := domain.EvalHistoryEntry{
    //     RunID:     exec.ID,
    //     SnapshotID: exec.SnapshotID,
    //     Score:     calculatedScore,
    //     ...
    // }
    // fm.EvalHistory = append(fm.EvalHistory, entry)

    // 5. 更新 eval_stats (Welford)
    // for model, score := range caseScores {
    //     stat := fm.EvalStats[model]
    //     stat.Update(float64(avgScore))
    //     fm.EvalStats[model] = stat
    // }

    // 6. 写回 frontmatter
    // commitMsg := fmt.Sprintf("Update eval_history for %s", exec.AssetID)
    // if _, err := s.fileManager.UpdateFrontmatter(ctx, exec.AssetID, updater, commitMsg); err != nil {
    //     return err
    // }

    // 7. Git commit
    // if s.gitBridger != nil {
    //     if err := s.gitBridger.Commit(ctx, commitMsg); err != nil {
    //         slog.Warn("failed to commit eval_history", "error", err)
    //     }
    // }

    return nil
}
```

**注意**: 当前 `RunEval` 只记录 LLM 调用，没有评分逻辑。评分需要 `eval_runner` 补充后才能真正写入 `eval_history`。

#### 通知 indexer 刷新

Eval 执行完成后，通过 `ConfigManager.Notify(ctx, "repo", []string{exec.AssetID})` 触发 indexer 重新扫描。

---

## 二、API 修改

### 2.1 GET /api/v1/assets - 支持 category 筛选

添加 `category` 查询参数:

```go
// In ListAssets handler
category := r.URL.Query().Get("category") // content/eval/metric
if category != "" {
    filters.Category = category
}
```

### 2.2 GET /api/v1/assets/{id} - 返回完整字段

返回 category, eval_history, eval_stats, triggers, test_cases, recommended_snapshot_id 等完整字段。

### 2.3 GET /api/v1/assets/{id}/metrics (新增)

返回某 eval asset 引用的 metric 列表（通过 frontmatter 的 metric_refs）。

### 2.4 GET /api/v1/metrics/{id}/used-by (新增)

返回引用某 metric 的 eval asset 列表（反向引用）。

### 2.5 GET /api/v1/executions (新增)

返回所有 execution 记录，路径: `.evals/executions/`。

### 2.6 GET /api/v1/executions/{id} (新增)

返回单个 execution 详情。

### 2.7 GET /api/v1/executions/{id}/calls (新增)

返回某次 execution 的所有 LLM call，路径: `.evals/calls/{execution_id}/calls.jsonl`。

---

## 三、前端修改

### 3.1 导航结构 - 下拉选择器

**文件**: `web/src/App.tsx`, `web/src/components/Sidebar.tsx`

```
┌─────────────────────────────────────────────────────────────┐
│  Logo   [Assets ▼]   [Settings]          [Create Asset +]  │
└─────────────────────────────────────────────────────────────┘
                │
                ▼
        ┌───────────────┐
        │ All Assets   │  ← 默认，显示所有 category
        │ Prompts      │  ← content 类型
        │ Eval Cases   │  ← eval 类型
        │ Metrics      │  ← metric 类型
        └───────────────┘
```

**实现方式**:
- 顶部导航栏下拉选择器（Select dropdown）
- 选择后筛选列表页显示对应 category 的 assets
- URL 保持 `/assets`，通过 query param `?category=content` 记录状态
- 下拉显示 category 统计数量: "Prompts (12)"

### 3.2 列表页 - Category 筛选和展示

**文件**: `web/src/views/AssetListView.tsx`

**卡片展示**:
- 左上角: Category 标签 (Prompts/Eval Cases/Metrics，不同颜色图标)
- 卡片底部根据 category 显示不同快捷信息:
  - content: "Latest: 85% (GPT-4)" 或 "No eval yet"
  - eval: "X test cases"
  - metric: "X rubric checks"

### 3.3 详情页 - 根据 Category 显示不同内容

**文件**: `web/src/views/ContentDetailView.tsx` (content)
**文件**: `web/src/views/EvalCasesView.tsx` (eval)
**文件**: `web/src/views/MetricDetailView.tsx` (metric)

**共用路由**: `/assets/:id` 根据 category 渲染不同视图

**Content 详情 (content)**:
- **Overview**: 基本信息、tags、triggers、recommended_snapshot_id
- **Editor**: Monaco editor 编辑内容
- **Versions**: 版本历史时间轴
- **Eval History**: 组合形式时间轴 + 详情卡片
- 可执行评测（跳转 EvalPanelView）

**Eval Cases 详情 (eval)**:
- **Overview**: 基本信息、tags、引用关系（metric_refs）
- **Cases Editor**: test_cases 列表，每个 case 可展开编辑 input/expected/rubric
- **Versions**: 版本历史
- 左侧: 引用关系面板（展示引用的 metrics，可跳转）

**Metric 详情 (metric)**:
- **Overview**: 基本信息、description
- **Rubric Editor**: rubric 列表（check、weight、criteria），可编辑
- **Versions**: 版本历史
- **Used By**: 反向引用列表（展示引用此 metric 的 eval assets，可跳转）

### 3.4 Eval History - 组合形式（时间轴 + 详情卡片）

```
┌──────────────────────────────────────────────────────────────────┐
│  [时间轴]                      │  [详情卡片]                     │
├────────────────────────────────┤                                 │
│ ● Exec #123                    │  Execution #123                 │
│   2026-04-26 10:15  85% ✓    │  Status: ✓ Passed              │
│                                │  Model: GPT-4o                  │
│ ● Exec #122                    │  Deterministic: 0.92            │
│   2026-04-25 14:30  72% ✗    │  Rubric Score: 85/100          │
│                                │  Tokens: 1500 in / 350 out      │
│ ● Exec #121                    │  Duration: 1200ms              │
│   2026-04-24 09:00  90% ✓    │                                │
│                                │  ─────────────────────────     │
│                                │  Snapshots: v1.2.3             │
│                                │  By: alice                     │
└────────────────────────────────┴────────────────────────────────┘
```

**左侧时间轴**:
- 按时间倒序排列（最新在上）
- 每个节点: execution_id、日期时间、分数、状态图标
- 可点击选中高亮

**右侧详情卡片**:
- 选中 execution 的详细信息
- 统计: status, model, deterministic_score, rubric_score, tokens, latency
- Eval Stats: 当前 asset 的 Welford 统计（count, mean, stddev, min, max）

### 3.5 引用关系展示

**eval → metric 引用** (eval 类型详情页):

```
┌──────────────────────────────────────────────────────────────────┐
│  Test Cases                              │  Referenced Metrics   │
├─────────────────────────────────────────┤                       │
│  ┌──────────────────────────────────┐   │  code-quality-v1     │
│  │ Case 1: 简单函数评审              │   │  security-check-v1   │
│  │ input: func Add(a, b int) int    │   │                      │
│  │ expected: score: 90              │   │  [跳转到 Metric 详情]│
│  └──────────────────────────────────┘   │                       │
└─────────────────────────────────────────┴───────────────────────┘
```

**metric → eval 反向引用** (metric 类型详情页):

```
┌──────────────────────────────────────────────────────────────────┐
│  Rubric                                  │  Used By             │
├─────────────────────────────────────────┤                       │
│  ┌──────────────────────────────────┐   │  code-review-eval    │
│  │ correctness (40%)                │   │  refactoring-eval   │
│  │   - 函数返回值正确                │   │                      │
│  │   - 边界条件处理                  │   │  [跳转到 Eval 详情]  │
│  └──────────────────────────────────┘   │                       │
└─────────────────────────────────────────┴───────────────────────┘
```

**实现**: 扫描所有 eval 类型 asset 的 frontmatter，检查 `metric_refs` 字段构建引用关系。

### 3.6 Execution 列表页

**路由**: `/executions`

```
┌──────────────────────────────────────────────────────────────────────┐
│  Executions                                           [Filter ▼]    │
├──────────────────────────────────────────────────────────────────────┤
│  ID          │ Asset        │ Status     │ Progress │ Model  │ Time│
├──────────────────────────────────────────────────────────────────────┤
│  exec_01AR3C │ code-review  │ Completed  │ 10/10   │ GPT-4o│ 10m │
│  exec_01AR3D │ refactoring  │ Running    │ 3/10    │ GPT-4o│ 3m  │
│  exec_01AR3E │ api-design   │ Failed     │ 0/10    │ GPT-4o│ 1m  │
└──────────────────────────────────────────────────────────────────────┘
                                    [View Calls] 按钮 →
```

**功能**:
- 列表分页
- Status 筛选（All / Completed / Running / Failed）
- 点击行查看 Execution 详情
- "View Calls" 按钮跳转 Call Log 查看器

### 3.7 LLM Call 查看器

**路由**: `/executions/:id/calls`

```
┌────────────────────────────────────────────────────────────────────────┐
│  Execution: exec_01AR3C                              [Back to Executions]│
├────────────────────────────────────────────────────────────────────────┤
│  [Call 列表]                    │  [Call 详情]                         │
├─────────────────────────────────┤                                      │
│  run_001  ✓  completed  10:00  │  Model: GPT-4o                      │
│  run_002  ✓  completed  10:05  │  Temperature: 0.7                    │
│  run_003  ✗  failed     10:10  │  Tokens: 1500 in / 350 out          │
│  run_004  ✓  completed  10:15  │  Latency: 1200ms                     │
│  ...                            │                                      │
│                                 │  ───────────────────────────────     │
│                                 │  [Prompt]  [Response]  [Raw JSON]    │
│                                 │  ───────────────────────────────     │
│                                 │                                      │
│                                 │  Prompt:                             │
│                                 │  ┌────────────────────────────────┐   │
│                                 │  │ # System                       │   │
│                                 │  │ You are a code reviewer...     │   │
│                                 │  │ # User                         │   │
│                                 │  │ func Add(a, b int) int {...}   │   │
│                                 │  └────────────────────────────────┘   │
│                                 │                                      │
│                                 │  Response:                           │
│                                 │  ┌────────────────────────────────┐   │
│                                 │  │ LGTM! The function correctly   │   │
│                                 │  │ implements addition. No issues │   │
│                                 │  └────────────────────────────────┘   │
└─────────────────────────────────┴──────────────────────────────────────┘
```

**功能**:
- 左侧: call 列表（run_id、状态图标、timestamp）
- 右侧 Tabs:
  - **Prompt**: 显示发送给 LLM 的完整 prompt
  - **Response**: 显示 LLM 返回内容
  - **Raw JSON**: 显示原始 JSON 结构
- 底部: metadata (Model、Temperature、Tokens、Latency)

### 3.8 CreateAsset 页面 - 支持 category（创建时必选，之后锁定）

**文件**: `web/src/views/CreateAssetView.tsx`

```
┌──────────────────────────────────────────────────────────────┐
│  Create New Asset                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Category:  [Prompt ▼]  ← 必选，创建后不可更改              │
│                                                              │
│  ────────────────────────────────────────────────────────── │
│                                                              │
│  [Prompt 时的表单]                                           │
│  Name:        [________________]                             │
│  Description: [________________]                             │
│  Tags:        [________________]                             │
│  Content:     [____________________________________]        │
│                                                              │
│  [Eval Case 时的表单]                                        │
│  Name:        [________________]                             │
│  Description: [________________]                             │
│  Test Cases:  [Monaco Editor - YAML]                        │
│                                                              │
│  [Metric 时的表单]                                           │
│  Name:        [________________]                             │
│  Description: [________________]                             │
│  Rubric:      [Monaco Editor - YAML]                        │
│                                                              │
│  [Create]  [Cancel]                                         │
└──────────────────────────────────────────────────────────────┘
```

**实现**:
- Category 下拉选择 (Prompt / Eval Case / Metric)
- **创建后 category 不可更改**（category 作为 URL 路径的一部分）
  - Prompt: `/assets/:id` → ContentDetailView
  - Eval Case: `/assets/:id` → EvalCasesView
  - Metric: `/assets/:id` → MetricDetailView
- 根据 category 渲染不同表单组件
- Content 用 Monaco editor
- Eval Case 用 YAML editor（编辑 test_cases）
- Metric 用 YAML editor（编辑 rubric）

---

## 四、涉及文件清单

### 后端

| 文件 | 修改 |
|------|------|
| `internal/domain/frontmatter.go` | +Category |
| `internal/service/interfaces.go` | AssetDetail +新字段, +AssetFileManager接口 |
| `internal/gateway/handlers/asset_handler.go` | AssetResponse +新字段, GetAsset填充 |
| `plugins/search/search.go` | reconcileFile/persist/Load/CreatePlaceholder |
| `internal/service/eval_service.go` | +fileManager, RunEval结束时写eval_history |
| `internal/gateway/handlers/execution_handler.go` | 新增 - execution API handler |
| `internal/gateway/handlers/call_handler.go` | 新增 - call API handler |

### 前端

| 文件 | 修改/新增 |
|------|---------|
| `web/src/App.tsx` | +Executions路由 |
| `web/src/components/Sidebar.tsx` | 下拉选择 category |
| `web/src/views/AssetListView.tsx` | category 筛选、卡片展示 |
| `web/src/views/ContentDetailView.tsx` | 新增 - content 详情（Eval History 组合视图） |
| `web/src/views/EvalCasesView.tsx` | 新增 - eval 详情（Cases Editor + 引用关系） |
| `web/src/views/MetricDetailView.tsx` | 新增 - metric 详情（Rubric Editor + Used By） |
| `web/src/views/CreateAssetView.tsx` | +Category 选择（必选） |
| `web/src/views/ExecutionListView.tsx` | 新增 - execution 列表 |
| `web/src/views/CallLogView.tsx` | 新增 - LLM call 查看器 |
| `web/src/api/client.ts` | +executions API, +calls API |

### API 新增

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/assets` | GET | +category 筛选参数 |
| `/api/v1/assets/:id` | GET | 返回 category, eval_history 等完整字段 |
| `/api/v1/assets/:id/metrics` | GET | 某 eval asset 引用的 metric 列表 |
| `/api/v1/metrics/:id/used-by` | GET | 引用某 metric 的 eval asset 列表 |
| `/api/v1/executions` | GET | 所有 execution（分页） |
| `/api/v1/executions/:id` | GET | 单个 execution 详情 |
| `/api/v1/executions/:id/calls` | GET | 某 execution 的 LLM calls |

---

## 五、优先级

| 优先级 | 内容 |
|--------|------|
| P0 | 后端数据模型 + API 返回（第一章 1.1-1.3, 第二章） |
| P1 | Eval 执行后写 eval_history（第一章 1.4） |
| P2 | 前端导航 + 列表页 category 筛选（第三章 3.1） |
| P3 | 前端详情页 - Content + Eval History 组合视图（第三章 3.2-3.4） |
| P4 | 前端详情页 - Eval Cases + Metric + 引用关系（第三章 3.3） |
| P5 | Execution 列表 + Call 查看器（第三章 3.6-3.7） |

---

## 六、验证方式

### 后端 API 测试

1. **Category 筛选**:
   ```bash
   # 创建不同 category 的 assets
   curl -X POST /api/v1/assets -d '{"id":"test-content","name":"Test","category":"content"}'
   curl -X POST /api/v1/assets -d '{"id":"test-eval","name":"TestEval","category":"eval"}'
   curl -X POST /api/v1/assets -d '{"id":"test-metric","name":"TestMetric","category":"metric"}'

   # 验证筛选
   curl /api/v1/assets?category=content   # 只返回 test-content
   curl /api/v1/assets?category=eval       # 只返回 test-eval
   curl /api/v1/assets?category=metric     # 只返回 test-metric
   ```

2. **完整 Asset 响应**:
   ```bash
   curl /api/v1/assets/test-content
   # 验证返回包含: category, eval_history, eval_stats, triggers, test_cases, recommended_snapshot_id
   ```

3. **Executions API**:
   ```bash
   curl /api/v1/executions                    # 返回所有 execution
   curl /api/v1/executions/exec_01AR3C        # 返回单个 execution
   curl /api/v1/executions/exec_01AR3C/calls  # 返回该 execution 的 LLM calls
   ```

4. **Eval 执行后写 frontmatter**:
   - 执行 `POST /api/v1/evals/execute`
   - 检查对应 asset 的 frontmatter 中 `eval_history` 和 `eval_stats` 已更新

### 前端 UI 测试

1. **导航**:
   - [ ] 顶部 "Assets" 下拉选择器正常显示
   - [ ] 选择不同 category 后列表正确筛选
   - [ ] URL 参数 `?category=` 正确更新

2. **Asset 卡片**:
   - [ ] 不同 category 显示不同颜色图标
   - [ ] 卡片底部显示 category 相关信息

3. **Content 详情页**:
   - [ ] Overview / Editor / Versions / Eval History 标签页正常
   - [ ] Eval History 组合视图（时间轴 + 详情卡片）正常

4. **Eval Cases 详情页**:
   - [ ] Cases Editor 显示 test_cases 列表
   - [ ] 左侧显示引用关系面板

5. **Metric 详情页**:
   - [ ] Rubric Editor 显示 rubric 列表
   - [ ] Used By 面板显示反向引用

6. **Execution 列表**:
   - [ ] 显示所有 execution，分页正常
   - [ ] "View Calls" 跳转正常

7. **Call 查看器**:
   - [ ] 左侧 call 列表正常
   - [ ] 右侧 Prompt / Response / Raw JSON tabs 正常

---

## 七、废弃代码清理（待定）

根据 `EVAL-STORAGE-DESIGN.md` 第六章:

| 文件 | 操作 |
|------|------|
| `internal/storage/eval_run_repository.go` | 删除（已废弃标注） |
| `internal/storage/eval_case_repository.go` | 删除（已废弃标注） |
| `internal/storage/eval_work_item_repository.go` | 删除 |
| `internal/storage/eval_execution_repository.go` | 删除（需补充 @Deprecated 标注） |
| `internal/storage/ent/evalrun*.go` | 删除 |
| `internal/storage/ent/evalworkitem*.go` | 删除 |
| `internal/storage/ent/evalexecution*.go` | 删除 |
| `internal/storage/ent/schema/evalrun.go` | 删除 |
| `internal/storage/ent/schema/evalworkitem.go` | 删除 |
| `internal/storage/ent/schema/evalexecution.go` | 删除 |

**注意**: 当前无用户，无需迁移，可直接删除。
