# LLM 语义能力 — 底层基础设施

## 背景

eval-prompt 的 LLM 集成已跑通。现在需要把 LLM 能力建设为底层基础设施，供多个服务复用：

1. **语义 Eval 评分** — 消除硬编码模型，使用配置的默认模型
2. **Trigger 自动生成** — 保存资产时自动从内容提取触发词，写入 frontmatter
3. **自然语言搜索** — trigger pattern 匹配优先，不足时用 LLM 语义重排补充
4. **语义 Diff** — 解释两个版本的语义变化

> 自动打标签暂不实现

## 设计原则

- `SemanticAnalyzer` 接口放在 service 层（不是 plugin 层）
- 使用 `InvokeWithSchema` 保证结构化 JSON 输出
- 不硬编码模型 — 使用 LLM 配置中的 `DefaultModel`
- 遵循现有 builder pattern（如 `WithLLMInvoker`）

---

## 第一阶段：核心基础设施

### 1.1 接口定义 — `internal/service/interfaces.go`

```go
// SemanticAnalyzer 提供基于 LLM 的语义能力。
type SemanticAnalyzer interface {
    AnalyzeContent(ctx context.Context, req AnalyzeContentRequest) (*AnalyzeContentResult, error)
    ExplainDiff(ctx context.Context, req ExplainDiffRequest) (*ExplainDiffResult, error)
}

// AnalyzeContentRequest 是 AnalyzeContent 的输入。
type AnalyzeContentRequest struct {
    Content     string // prompt 文本内容
    Description string // 可选，资产描述
    AssetType    string // 可选，业务线提示
}

// AnalyzeContentResult 是 AnalyzeContent 的输出。
type AnalyzeContentResult struct {
    Triggers []TriggerEntry        // 提取的触发词
    Issues   []ContentIssue        // 发现的问题
    Score    ContentScore          // 质量评分
}

// TriggerEntry 是从 prompt 中提取的触发词条目。
type TriggerEntry struct {
    Pattern    string   `yaml:"pattern"`    // 正则 pattern，如 "我要投诉|东西坏了"
    Examples   []string `yaml:"examples"`   // 示例输入
    Confidence float64  `yaml:"confidence"`  // 置信度 0-1
}

// ContentIssue 是 prompt 中发现的问题。
type ContentIssue struct {
    Severity   string `json:"severity"` // critical | high | medium | low
    Location   string `json:"location"` // 问题位置描述
    Problem    string `json:"problem"`  // 问题描述
    Suggestion string `json:"suggestion"` // 建议
}

// ContentScore 是 prompt 的质量评分。
type ContentScore struct {
    Overall      float64 `json:"overall"`       // 总体质量 0-1
    Clarity      float64 `json:"clarity"`       // 清晰度 0-1
    Completeness float64 `json:"completeness"`  // 完整性 0-1
}

// ExplainDiffRequest 是 ExplainDiff 的输入。
type ExplainDiffRequest struct {
    OldContent string // 旧版本内容
    NewContent string // 新版本内容
    OldVersion string // 旧版本号
    NewVersion string // 新版本号
}

// ExplainDiffResult 是 ExplainDiff 的输出。
type ExplainDiffResult struct {
    Summary string           `json:"summary"`  // 变更摘要
    Changes []SemanticChange `json:"changes"` // 语义变更列表
    Impact  string           `json:"impact"`  // low | medium | high
}

// SemanticChange 是一条语义变更。
type SemanticChange struct {
    Type        string `json:"type"`         // added | removed | modified
    Location    string `json:"location"`     // 变更位置
    Description string `json:"description"`  // 变更描述
    Significance string `json:"significance"` // low | medium | high
}
```

### 1.2 实现 — `internal/service/semantic_service.go`（新建）

用 `LLMInvoker.InvokeWithSchema` 实现 `SemanticAnalyzer`。

- 存储 `invoker LLMInvoker` 和 `model string`（从配置传入）
- 每个方法构建 prompt，调用 `InvokeWithSchema`（带 JSON schema），解析 JSON 结果
- 结构化任务用 temperature 0.3
- 工厂方法：`NewSemanticService(invoker LLMInvoker, model string) *SemanticService`

---

## 第二阶段：消除 Eval 硬编码模型

### 2.1 `plugins/eval/runner.go`

`RunRubric` 第 91 行当前硬编码 `"gpt-4o"`，改为接受 `model string` 参数：

```go
func (r *Runner) RunRubric(ctx context.Context, output string, rubric service.Rubric, invoker service.LLMInvoker, model string) (service.RubricResult, error)
```

### 2.2 `internal/service/interfaces.go` — `EvalRunner` 接口

```go
type EvalRunner interface {
    RunDeterministic(ctx context.Context, trace []TraceEvent, checks []DeterministicCheck) (DeterministicResult, error)
    RunRubric(ctx context.Context, output string, rubric Rubric, invoker LLMInvoker, model string) (RubricResult, error)
}
```

### 2.3 调用方更新

- `internal/service/eval_executor.go` — Worker 调用时传入 `model`（从 execution config 获取）
- `internal/service/eval_service.go` — `DiagnoseEval`（约 693 行）硬编码 `"gpt-4o"`，改为使用配置的默认模型

---

## 第三阶段：Trigger 自动生成 + 自然语言搜索

### 3.1 frontmatter 扩展 — `internal/domain/frontmatter.go`

当前 `FrontMatter` 没有 `trigger` 字段，新增：

```go
type FrontMatter struct {
    // ... 现有字段 ...
    Triggers []TriggerEntry `yaml:"triggers,omitempty"`
}
```

### 3.2 保存资产时自动生成 Trigger

在 `AssetIndexer.SaveFileContent` 或 `TriggerService` 相关位置，保存时调用 `SemanticAnalyzer.AnalyzeContent`，把返回的 `Triggers` 写入 frontmatter。

```
用户保存 prompt 资产
    ↓
AnalyzeContent(content)
    ↓
返回 { Triggers: [{pattern, examples, confidence}] }
    ↓
写入 frontmatter Triggers 字段
```

### 3.3 Trigger 匹配 + 自然语言搜索

改进 `TriggerService.MatchTrigger`：

```go
func (s *TriggerService) MatchTrigger(ctx context.Context, input string, top int) ([]*MatchedPrompt, error) {
    // 1. 遍历所有 asset，读 frontmatter.Triggers
    // 2. 用正则匹配 input，收集命中的 asset
    // 3. 如果命中数 < top，进入自然语言搜索
    //    - keyword 匹配拿 20 条候选
    //    - 对每条候选调 AnalyzeContent，按语义相关度打分
    //    - 排序取 top N
}
```

**注意**：遍历所有 asset 做 trigger 正则匹配是 O(N) 的，当 asset 数量大时需优化（如预建倒排索引）。MVP 阶段先做全量遍历。

### 3.4 TriggerService 依赖

`internal/service/trigger_service.go` 新增：

```go
type TriggerService struct {
    indexer         AssetIndexer
    gitBridger      GitBridger
    semanticAnalyzer SemanticAnalyzer  // 新增
    model           string            // 新增，配置的默认模型
}

func NewTriggerService(indexer AssetIndexer, gitBridger GitBridger) *TriggerService {
    return &TriggerService{indexer: indexer, gitBridger: gitBridger}
}

func (s *TriggerService) WithSemanticAnalyzer(sa SemanticAnalyzer, model string) *TriggerService {
    s.semanticAnalyzer = sa
    s.model = model
    return s
}
```

---

## 第四阶段：依赖注入 — `cmd/ep/commands/serve.go`

```go
// 创建 semantic service
semanticService := service.NewSemanticService(llmInvoker, defaultModel)

// 注入 eval service
evalService := service.NewEvalService(...).WithSemanticAnalyzer(semanticService)

// 注入 trigger service
triggerService := service.NewTriggerService(...).WithSemanticAnalyzer(semanticService, defaultModel)
```

---

## 需修改的文件

| 文件 | 改动 |
|------|------|
| `internal/domain/frontmatter.go` | 添加 `Triggers []TriggerEntry` |
| `internal/service/interfaces.go` | 添加 `SemanticAnalyzer` 接口 + 所有类型定义 |
| `internal/service/semantic_service.go` | **新建** — 实现 `SemanticAnalyzer` |
| `plugins/eval/runner.go` | `RunRubric` 增加 `model` 参数 |
| `internal/service/interfaces.go` | `EvalRunner.RunRubric` 签名增加 model |
| `internal/service/eval_executor.go` | 向 `RunRubric` 传入 model |
| `internal/service/eval_service.go` | 添加 `WithSemanticAnalyzer`，更新 `DiagnoseEval` |
| `internal/service/trigger_service.go` | 添加 `SemanticAnalyzer` 依赖和改进的 `MatchTrigger` |
| `cmd/ep/commands/serve.go` | 注入 `SemanticService` |
| `internal/service/mocks/mock.go` | 添加 `MockSemanticAnalyzer` |

## 验证步骤

1. 启动 server，配置 LLM
2. 创建 prompt → 保存后 frontmatter 有 `triggers` 字段
3. 用"客户投诉"搜索 → 命中 trigger 匹配的 asset
4. 用不规则的自然语言搜索 → 走 LLM 语义重排
5. 运行 eval → rubric 评分使用配置的模型（非硬编码）
