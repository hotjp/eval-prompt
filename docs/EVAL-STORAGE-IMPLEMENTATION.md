# Eval 存储实现待办

**日期**: 2026-04-26
**状态**: 待实现

---

## 一、数据模型变更

### 1.1 domain/frontmatter.go - 添加 category

```go
type FrontMatter struct {
    // ... existing fields ...
    Category string `yaml:"category,omitempty"` // 新增: content/eval/metric
}
```

### 1.2 service/interfaces.go - 扩展 AssetDetail

```go
type AssetDetail struct {
    // ... existing fields ...

    // 新增字段
    Category              string                    `json:"category,omitempty"`
    EvalHistory          []domain.EvalHistoryEntry `json:"eval_history,omitempty"`
    EvalStats            domain.EvalStats         `json:"eval_stats,omitempty"`
    Triggers             []domain.TriggerEntry    `json:"triggers,omitempty"`
    TestCases            []domain.TestCase        `json:"test_cases,omitempty"`
    RecommendedSnapshotID string                   `json:"recommended_snapshot_id,omitempty"`
    Labels               []LabelInfo              `json:"labels,omitempty"` // 修复: 原来是 nil
}
```

### 1.3 gateway/handlers/asset_handler.go - 扩展 AssetResponse

```go
type AssetResponse struct {
    // ... existing fields ...

    // 新增字段
    Category              string             `json:"category,omitempty"`
    EvalHistory           []EvalHistoryEntry `json:"eval_history,omitempty"`
    EvalStats             EvalStats         `json:"eval_stats,omitempty"`
    Triggers              []TriggerEntry    `json:"triggers,omitempty"`
    TestCases             []TestCase        `json:"test_cases,omitempty"`
    RecommendedSnapshotID string            `json:"recommended_snapshot_id,omitempty"`
}
```

并修改 `GetAsset` 方法，填充以上新字段。

---

## 二、plugins/search/search.go 修改

### 2.1 reconcileFile() - 填充新字段

位置: line 289-297

```go
i.assets[fm.ID] = &assetEntry{
    asset: asset,
    detail: &service.AssetDetail{
        // ... existing fields ...
        Category:              fm.Category,
        EvalHistory:          fm.EvalHistory,
        EvalStats:            fm.EvalStats,
        Triggers:             fm.Triggers,
        TestCases:            fm.TestCases,
        RecommendedSnapshotID: fm.RecommendedSnapshotID,
        Labels:               parseLabels(fm.Labels), // 修复: 从 nil 改为实际值
    },
}
```

需要新增 `parseLabels()` 辅助函数，将 `[]domain.LabelEntry` 转换为 `[]service.LabelInfo`。

### 2.2 persist()/Load() - 持久化新字段

persist() 的 `persistEntry` 结构体需要添加:
```go
type persistEntry struct {
    // ... existing fields ...
    Category              string                    `json:"category"`
    EvalHistory          []domain.EvalHistoryEntry `json:"eval_history"`
    EvalStats            domain.EvalStats         `json:"eval_stats"`
    Triggers             []domain.TriggerEntry    `json:"triggers"`
    TestCases            []domain.TestCase        `json:"test_cases"`
    RecommendedSnapshotID string                   `json:"recommended_snapshot_id"`
}
```

Load() 需要同步恢复这些字段。

### 2.3 CreatePlaceholder() - 添加 category 参数

```go
func (i *Indexer) CreatePlaceholder(ctx context.Context, id, name, bizLine string, tags []string, category string) error {
    // ...
    fm := &domain.FrontMatter{
        // ...
        Category: category,
    }
}
```

### 2.4 新增 NotifyIndexChange() 方法

```go
// NotifyIndexChange 当外部（如 eval 执行）修改了 asset 文件后，通知 indexer 刷新
func (i *Indexer) NotifyIndexChange(ctx context.Context, assetID string) error {
    // 读取该 asset 的 frontmatter
    // 更新 i.assets[assetID]
}
```

---

## 三、service/eval_service.go 修改

### 3.1 添加 AssetFileManager 依赖

```go
type EvalService struct {
    // ... existing fields ...
    fileManager AssetFileManager // 新增
}

func (s *EvalService) WithFileManager(fileManager AssetFileManager) *EvalService {
    s.fileManager = fileManager
    return s
}
```

注: 需要在 `service/interfaces.go` 中定义 `AssetFileManager` 接口（已存在于 `asset_file.go`，需提升到 interfaces.go）。

### 3.2 RunEval 结束时写 eval_history

在 `RunEval` 方法的 `return exec, nil` 之前添加:

```go
// 更新 asset frontmatter 的 eval_history 和 eval_stats
if err := s.updateAssetEvalHistory(ctx, exec); err != nil {
    slog.Warn("failed to update asset eval_history", "error", err)
}
```

### 3.3 新增 updateAssetEvalHistory 方法

```go
func (s *EvalService) updateAssetEvalHistory(ctx context.Context, exec *domain.EvalExecution) error {
    // 1. 读取 .evals/calls/{exec.ID}/calls.jsonl 获取所有 call 记录
    calls, err := s.callStore.ListByExecution(ctx, exec.ID)
    if err != nil {
        return err
    }

    // 2. 按 case 分组，计算每 case 的平均分
    caseScores := make(map[string][]int)
    for _, call := range calls {
        if call.Status == "completed" {
            // 注意: 当前 RunEval 没有评分逻辑，LLMCall 只有 response_content
            // 评分需要后续 eval_runner 补充
        }
    }

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

### 3.4 通知 indexer 刷新

Eval 执行完成后，需要通知 indexer 重新扫描该 asset 文件:

```go
// 方案1: 直接调用 indexer.NotifyIndexChange()
// 需要 EvalService 持有 Indexer 引用（通过接口）

// 方案2: 通过 ConfigManager
// s.configManager.Notify(ctx, "repo", []string{exec.AssetID})
```

---

## 四、待讨论: UI 是否需要修改

### 4.1 前端现状

根据代码分析，前端（`web/src/views/AssetListView.tsx`）目前只显示:
- 列表页: id, name, description, tags
- 详情页: id, name, description, asset_type, tags, state, labels, snapshots

### 4.2 建议的 UI 改动

| 页面 | 建议新增显示 |
|------|------------|
| Asset 列表页 | category 筛选/显示 |
| Asset 详情页 | category 标签页 |
| Asset 详情页 | Eval History 标签页（显示评测历史、分数趋势） |
| Asset 详情页 | Test Cases 标签页（如果是 eval category） |
| Asset 详情页 | Triggers 展示 |
| Asset 详情页 | Metrics 展示（如果是 metric category） |

### 4.3 评测相关页面

如果存在评测执行 UI:
- Execution 列表/详情
- LLM Call 调用记录查看
- 评分详情（rubric check 结果）

---

## 五、废弃代码清理（待定）

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

---

## 六、优先级建议

1. **P0**: 核心数据模型 + API 返回（本文第一、二、三章）
2. **P1**: Eval 执行后写 eval_history（第三章 3.2-3.4）
3. **P2**: UI 展示新字段（第四章）
4. **P3**: 废弃代码清理（第五章）

---

## 七、依赖关系图

```
domain/frontmatter.go (+Category)
         ↓
service/interfaces.go (AssetDetail +新字段)
         ↓
gateway/handlers/asset_handler.go (AssetResponse +新字段)
         ↓
plugins/search/search.go (reconcileFile/persist/Load)
         ↓
service/eval_service.go (RunEval 结束时写 eval_history)
         ↓
ConfigManager 或 Indexer.NotifyIndexChange (通知刷新)
```
