# Eval Orchestrator 架构设计：可编排、可组合、可注入的评估引擎

> 核心思想：评估不是"跑一个固定脚本"，而是**编排多个评估 Agent 的并行执行**，支持提示词注入和人工量表的动态组合。

---

## 一、问题定义

传统 eval 工具的问题：
- **固化**：写死了一套评估逻辑，想加新指标要改代码
- **串行**：BERTScore 跑完跑 G-Eval，再跑 FACTScore——10 个指标跑 10 分钟
- **封闭**：用户想加自己的评估标准（"我们业务要求回答必须带emoji"），做不到
- **单一**：只测模型输出，不测模型在"被干扰"时的表现

我们要的：
- **Plugin 化**：每个评估方式是一个独立 Agent，可插拔
- **并行化**：10 个 Agent 同时跑，10 秒出结果
- **可注入**：在评估过程中插入对抗性提示、模糊指令、位置交换等干扰
- **可定义**：用户用自然语言或 YAML 定义自己的评估标准

---

## 二、核心抽象：四个概念

### 2.1 EvalPlugin（评估插件）

一个 EvalPlugin 是一个**独立的评估 Agent**，有统一的输入输出接口。

```go
// internal/service/eval/plugin.go

// EvalInput：所有插件的统一输入
type EvalInput struct {
    AssetID         string            // 被评估的 Prompt Asset
    SnapshotVersion string            // 版本号
    Prompt          string            // Prompt 内容
    TestCases       []TestCase        // 测试用例（input + expected）
    Variables       map[string]string // 注入变量
    
    // 执行上下文
    Embedder  Embedder  // 嵌入模型
    Judge     Judge     // 评估模型
    Generator Generator // 生成模型（被测模型）
}

// EvalResult：所有插件的统一输出
type EvalResult struct {
    PluginName string                 // 插件标识
    Score      float64                // 总分（0-1 或 0-100，插件自定）
    Confidence float64                // 置信度（0-1）
    Dimensions map[string]Dimension   // 多维度分解
    Details    []EvalDetail           // 逐 case 详情
    Metadata   map[string]interface{} // 插件自定义元数据
    Paper      string                 // 论文引用
}

type Dimension struct {
    Score      float64
    Weight     float64
    CI         [2]float64            // 置信区间
    Samples    int                   // 采样次数
}

type EvalDetail struct {
    CaseID    string
    Input     string
    Expected  string
    Actual    string
    Score     float64
    Reasoning string                // 评估理由（Judge 的 CoT）
    Pass      bool
}

// EvalPlugin 接口
type EvalPlugin interface {
    Name() string
    Description() string
    RequiredCapabilities() []Capability // 声明依赖（需要 Embedder / Judge / Generator）
    
    // Evaluate 执行评估
    Evaluate(ctx context.Context, input EvalInput) (*EvalResult, error)
}
```

### 2.2 EvalOrchestrator（编排器）

编排器负责：**选择插件 → 注入策略 → 并行执行 → 统计后处理**。

```go
// internal/service/eval/orchestrator.go

type EvalConfig struct {
    AssetID         string
    SnapshotVersion string
    BaselineVersion string            // 用于对比的基线版本
    
    // 插件选择
    Plugins []string                   // ["bertscore", "geval", "beliefrevision", "custom:my-rubric"]
    
    // 注入策略
    Injections []InjectionStrategy    // 提示词注入策略列表
    
    // 统计配置
    BootstrapN      int                // Bootstrap 采样次数，默认 1000
    ConfidenceLevel float64            // 默认 0.95
    
    // 执行配置
    Parallelism     int                // 并行度，默认 5
    Timeout         time.Duration      // 单个插件超时
}

// Run 执行一次完整的评估编排
func (o *Orchestrator) Run(ctx context.Context, config EvalConfig) (*OrchestratorResult, error) {
    // 1. 加载 Asset 和 Test Cases
    asset, cases := o.loadAsset(config.AssetID, config.SnapshotVersion)
    
    // 2. 构建执行计划（哪些插件可以并行）
    plan := o.buildExecutionPlan(config.Plugins)
    
    // 3. 应用注入策略，生成变体 Cases
    variantCases := o.applyInjections(cases, config.Injections)
    
    // 4. 并行执行所有插件
    results := o.runParallel(ctx, plan, variantCases, config.Parallelism)
    
    // 5. 如果有 Baseline，执行对比统计
    if config.BaselineVersion != "" {
        baselineResults := o.loadBaseline(config.AssetID, config.BaselineVersion)
        results = o.computeComparisons(results, baselineResults, config.ConfidenceLevel)
    }
    
    // 6. Bootstrap 置信区间
    results = o.computeConfidenceIntervals(results, config.BootstrapN, config.ConfidenceLevel)
    
    // 7. ELO 更新（如果启用排名）
    o.updateELO(config.AssetID, config.SnapshotVersion, results)
    
    return &OrchestratorResult{
        Config:  config,
        Results: results,
        Summary: o.generateSummary(results),
    }, nil
}
```

### 2.3 InjectionStrategy（注入策略）

注入策略不是"攻击"，而是**在评估过程中插入干扰，测试模型的鲁棒性**。

```go
// internal/service/eval/injection.go

type InjectionType string

const (
    // PositionSwap：交换 A/B 答案顺序，检测位置偏见
    InjectionPositionSwap InjectionType = "position_swap"
    
    // AdversarialPrefix：在输入前添加对抗性前缀
    InjectionAdversarialPrefix InjectionType = "adversarial_prefix"
    
    // VagueInstruction：模糊化指令，测试模型对模糊输入的处理
    InjectionVagueInstruction InjectionType = "vague_instruction"
    
    // MultilingualMix：多语言混合输入
    InjectionMultilingualMix InjectionType = "multilingual_mix"
    
    // ContextOverload：超长上下文，测试注意力分散
    InjectionContextOverload InjectionType = "context_overload"
    
    // ConstraintConflict：插入与先验假设冲突的约束（明星航班场景）
    InjectionConstraintConflict InjectionType = "constraint_conflict"
)

type InjectionStrategy struct {
    Type   InjectionType
    Params map[string]interface{} // 策略参数
}

// 应用示例
func applyInjection(case TestCase, strategy InjectionStrategy) []TestCase {
    switch strategy.Type {
    case InjectionPositionSwap:
        // 把 case 拆成 A/B 两个变体，交换顺序
        return []TestCase{
            {ID: case.ID + "_ab", Input: case.Input + "\nA: " + case.Expected + "\nB: " + case.Actual},
            {ID: case.ID + "_ba", Input: case.Input + "\nA: " + case.Actual + "\nB: " + case.Expected},
        }
        
    case InjectionConstraintConflict:
        // 在第二轮提问中插入冲突约束
        return []TestCase{
            {ID: case.ID + "_round1", Input: case.Input, Round: 1},
            {ID: case.ID + "_round2", Input: case.Input + "\n但注意：" + strategy.Params["conflict"].(string), Round: 2},
        }
        
    case InjectionAdversarialPrefix:
        prefix := strategy.Params["prefix"].(string)
        return []TestCase{{ID: case.ID + "_adv", Input: prefix + "\n" + case.Input}}
    }
}
```

### 2.4 CustomRubric（人工定义量表）

用户用自然语言或 YAML 定义评估标准，系统动态生成 Custom Plugin。

```go
// internal/service/eval/custom_rubric.go

// CustomRubric 定义
type CustomRubric struct {
    ID          string
    Name        string
    Description string
    Criteria    []Criterion
    ScoringMode ScoringMode // absolute / pairwise
}

type Criterion struct {
    ID          string
    Name        string
    Description string        // 自然语言描述，Judge 据此评分
    Weight      float64       // 权重
    Scale       int           // 分数范围（如 1-5）
    Required    bool          // 是否必须满足（不满足直接不及格）
}

type ScoringMode string

const (
    ScoringAbsolute ScoringMode = "absolute"   // G-Eval 模式：直接打分
    ScoringPairwise ScoringMode = "pairwise"   // 成对比较模式
)
```

**用户定义示例（YAML）**：

```yaml
# .evals/rubrics/customer-service.yaml
id: customer-service-v1
name: 客服回复质量评估
description: 评估客服 Prompt 的回复是否符合业务标准

scoring_mode: absolute

criteria:
  - id: empathy
    name: 共情能力
    description: |
      回复是否表达了理解和同情？
      5分：真诚道歉并表达理解用户情绪
      3分：机械性道歉
      1分：没有共情表达
    weight: 0.25
    scale: 5
    required: false

  - id: solution
    name: 解决方案完整性
    description: |
      是否提供了具体的解决步骤？
      5分：步骤清晰、可执行
      3分：有步骤但不够详细
      1分：没有实际解决方案
    weight: 0.35
    scale: 5
    required: true   # 必须有解决方案，否则整体不及格

  - id: brand_voice
    name: 品牌语调一致性
    description: |
      回复是否符合品牌设定的友好、专业但不失轻松的语调？
      是否使用了品牌规定的emoji和话术？
    weight: 0.20
    scale: 5
    required: false

  - id: safety
    name: 安全合规
    description: |
      是否泄露了用户隐私或公司敏感信息？
      是否给出了可能造成伤害的建议？
      5分：完全安全
      1分：有严重安全隐患
    weight: 0.20
    scale: 5
    required: true   # 安全必须达标
```

**系统动态生成 Plugin**：

```go
// 读取 YAML → 生成 CustomRubricPlugin → 注册到 Registry
func LoadCustomRubric(path string) (EvalPlugin, error) {
    rubric := parseYAML(path)
    return &customRubricPlugin{rubric: rubric}, nil
}

type customRubricPlugin struct {
    rubric CustomRubric
}

func (p *customRubricPlugin) Evaluate(ctx context.Context, input EvalInput) (*EvalResult, error) {
    results := make(map[string]Dimension)
    details := []EvalDetail{}
    
    for _, criterion := range p.rubric.Criteria {
        // 对每个 criterion，调用 Judge 评分
        scores := []float64{}
        for _, tc := range input.TestCases {
            score, reasoning := p.scoreWithJudge(ctx, input.Judge, criterion, tc)
            scores = append(scores, score)
            details = append(details, EvalDetail{
                CaseID:    tc.ID,
                Score:     score,
                Reasoning: reasoning,
            })
        }
        
        mean := mean(scores)
        ci := bootstrapCI(scores, 0.95)
        
        results[criterion.ID] = Dimension{
            Score:   mean / float64(criterion.Scale), // 归一化到 0-1
            Weight:  criterion.Weight,
            CI:      [2]float64{ci[0] / float64(criterion.Scale), ci[1] / float64(criterion.Scale)},
            Samples: len(scores),
        }
    }
    
    // 加权总分
    totalScore := 0.0
    for _, dim := range results {
        totalScore += dim.Score * dim.Weight
    }
    
    // 检查 required criteria
    passed := true
    for _, criterion := range p.rubric.Criteria {
        if criterion.Required && results[criterion.ID].Score < 0.6 {
            passed = false
            break
        }
    }
    
    return &EvalResult{
        PluginName: "custom:" + p.rubric.ID,
        Score:      totalScore,
        Confidence: 0.9, // 基于采样次数计算
        Dimensions: results,
        Details:    details,
        Metadata: map[string]interface{}{
            "rubric_name": p.rubric.Name,
            "passed_required": passed,
        },
    }, nil
}
```

---

## 三、排列组合：一次 Eval 跑什么

用户的核心诉求：**不是选 A 或 B，而是 A + B + C 的组合**。

### 3.1 组合示例

```yaml
# config/eval-profiles/standard.yaml
name: 标准评估套件
description: 发布前的完整评估

plugins:
  - bertscore                    # 语义相似度
  - geval:instruction_following  # 指令遵循
  - geval:factual_accuracy      # 事实准确
  - beliefrevision              # 信念修正（明星航班类场景）
  - constraint                  # 约束满足
  - custom:customer-service     # 业务自定义量表

injections:
  - type: position_swap          # 检测位置偏见
    params: {}
  - type: constraint_conflict    # 检测信念修正能力
    params:
      conflict: "但注意：用户刚表示他不喜欢这个方案"

baseline: v1.4                   # 与 v1.4 对比

statistics:
  bootstrap_n: 1000
  confidence_level: 0.95

parallelism: 5
```

### 3.2 组合后的执行流程

```
1. 加载 Prompt Asset + Test Cases（20 cases）
   ↓
2. 应用注入策略
   - position_swap: 20 → 40 cases（每 case 生成 ab/ba 两个变体）
   - constraint_conflict: 20 → 40 cases（每 case 生成 round1/round2）
   总计：20 + 40 + 40 = 100 个评估单元
   ↓
3. 并行分发到 6 个 Plugin
   ├─ bertscore: 评估 100 个单元
   ├─ geval_instruction: 评估 100 个单元
   ├─ geval_factual: 评估 100 个单元
   ├─ beliefrevision: 只评估 20 个有 round2 的单元
   ├─ constraint: 评估 100 个单元
   └─ custom:customer-service: 评估 100 个单元
   
   6 个 Plugin × 100 单元 = 600 次评估调用
   并行度 5 → 约 120 批次 → 预计 2-3 分钟
   ↓
4. 收集所有结果
   ↓
5. 统计后处理
   - 每个 Plugin：Bootstrap CI
   - vs Baseline：成对 t 检验 + Cohen's d
   - ELO 排名更新
   ↓
6. 生成报告
```

### 3.3 不同场景的组合模板

| 场景 | 插件组合 | 注入策略 |
|------|---------|---------|
| **快速迭代** | `bertscore` + `geval:instruction` | 无 |
| **发布前检查** | 全部标准插件 | `position_swap` + `constraint_conflict` |
| **安全审查** | `factscore` + `selfcheckgpt` + `custom:safety-rubric` | `adversarial_prefix` |
| **Agent 评估** | `task_success` + `tool_accuracy` + `trajectory` | `context_overload` |
| **多语言测试** | `bertscore` + `geval` | `multilingual_mix` |

---

## 四、Plugin 注册与发现机制

### 4.1 内置 Plugin（Go 代码）

```go
// internal/service/eval/registry.go

var defaultRegistry = NewRegistry()

func init() {
    // 内置插件自动注册
    defaultRegistry.Register(&bertscore.Plugin{})
    defaultRegistry.Register(&geval.Plugin{})
    defaultRegistry.Register(&beliefrevision.Plugin{})
    defaultRegistry.Register(&constraint.Plugin{})
    defaultRegistry.Register(&factscore.Plugin{})
    defaultRegistry.Register(&selfcheck.Plugin{})
    defaultRegistry.Register(&pairwise.Plugin{})
}

// 从文件系统加载自定义 Rubric（启动时扫描）
func LoadCustomRubrics(dir string) error {
    files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
    for _, f := range files {
        plugin, err := LoadCustomRubric(f)
        if err != nil { continue }
        defaultRegistry.Register(plugin)
    }
    return nil
}
```

### 4.2 Plugin 目录约定

```
internal/service/eval/
├── engine.go              # Orchestrator
├── plugin.go              # 接口定义
├── registry.go            # 注册表
├── injection.go           # 注入策略
├── custom_rubric.go       # 自定义量表
├── stats/                 # 统计层
│   ├── bootstrap.go
│   ├── ttest.go
│   └── elo.go
└── plugins/               # 内置插件
    ├── bertscore/
    │   └── plugin.go
    ├── geval/
    │   └── plugin.go
    ├── beliefrevision/
    │   └── plugin.go
    ├── constraint/
    │   └── plugin.go
    ├── factscore/
    │   └── plugin.go
    ├── selfcheck/
    │   └── plugin.go
    └── pairwise/
        └── plugin.go

configs/rubrics/           # 用户自定义量表
├── customer-service.yaml
├── safety-check.yaml
└── brand-voice.yaml
```

---

## 五、API 设计

### 5.1 执行编排

```http
POST /api/v1/evals/orchestrate
Content-Type: application/json

{
  "asset_id": "prompt-001",
  "snapshot_version": "v2.0",
  "baseline_version": "v1.4",
  "profile": "standard",           // 引用预设组合
  // 或 inline 定义：
  "plugins": [
    "bertscore",
    "geval:instruction_following",
    "beliefrevision",
    "custom:customer-service"
  ],
  "injections": [
    {"type": "position_swap"},
    {"type": "constraint_conflict", "params": {"conflict": "用户表示不喜欢这个方案"}}
  ],
  "statistics": {
    "bootstrap_n": 1000,
    "confidence_level": 0.95
  },
  "parallelism": 5
}
```

### 5.2 管理自定义量表

```http
# 上传自定义量表
POST /api/v1/evals/rubrics
Content-Type: multipart/form-data
file: customer-service.yaml

# 列出所有可用量表
GET /api/v1/evals/rubrics

# 预览量表效果（不执行完整评估）
POST /api/v1/evals/rubrics/{id}/preview
{
  "case_id": "case-001",
  "response": "用户回复内容"
}
```

### 5.3 返回结果

```json
{
  "execution_id": "exec_xyz",
  "status": "completed",
  "duration_ms": 125000,
  
  "summary": {
    "vs_baseline": {
      "conclusion": "significantly_better",
      "p_value": 0.003,
      "cohens_d": 0.68,
      "elo_delta": 65
    },
    "injection_results": {
      "position_swap": {
        "consistency_rate": 0.94,
        "position_bias_detected": false
      },
      "constraint_conflict": {
        "belief_revision_rate": 0.78,
        "failure_cases": ["case-003", "case-017"]
      }
    }
  },
  
  "plugin_results": {
    "bertscore": {
      "score": 0.84,
      "ci_95": [0.81, 0.87],
      "paper": "Zhang et al. ICLR 2020"
    },
    "geval:instruction_following": {
      "score": 4.3,
      "scale": 5,
      "ci_95": [4.1, 4.5],
      "paper": "Liu et al. EMNLP 2023"
    },
    "beliefrevision": {
      "score": 0.78,
      "failure_mode": "20% cases 未修正初始信念",
      "paper": "原创"
    },
    "custom:customer-service": {
      "score": 0.82,
      "passed_required": true,
      "dimensions": {
        "empathy": {"score": 0.86, "weight": 0.25},
        "solution": {"score": 0.80, "weight": 0.35},
        "brand_voice": {"score": 0.88, "weight": 0.20},
        "safety": {"score": 0.95, "weight": 0.20}
      }
    }
  }
}
```

---

## 六、前端界面映射

### 6.1 EvalRunView → Orchestrator Config

```
┌─────────────────────────────────────────────────────────────┐
│  评估编排配置                                                │
├─────────────────────────────────────────────────────────────┤
│  预设模板 ▼ [标准套件]  [快速迭代]  [安全审查]  [自定义...]    │
├─────────────────────────────────────────────────────────────┤
│  插件选择（拖拽排列，可多选）                                  │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │
│  │BERTScore│ │G-Eval   │ │信念修正 │ │约束满足 │  [+]      │
│  │   ✅    │ │   ✅    │ │   ✅    │ │   ✅    │           │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘          │
│                                                             │
│  自定义量表                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 📄 customer-service.yaml  [编辑] [预览] [删除]       │   │
│  │ 📄 safety-check.yaml      [编辑] [预览] [删除]       │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  注入策略                                                    │
│  ☑ 位置偏见检测 (Swap Test)                                  │
│  ☑ 约束冲突测试                                              │
│    └─ 冲突内容: "用户表示不喜欢这个方案"                      │
│  ☐ 对抗性前缀                                                │
│  ☐ 多语言混合                                                │
├─────────────────────────────────────────────────────────────┤
│  统计配置                                                    │
│  Bootstrap: 1000次  │  置信水平: 95%  │  并行度: 5         │
├─────────────────────────────────────────────────────────────┤
│  [▶ 开始编排评估]                                           │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 EvalReportView → Multi-Plugin Report

```
┌─────────────────────────────────────────────────────────────┐
│  编排报告: prompt-001 v2.0  vs  Baseline v1.4               │
│  执行时间: 2m 15s  │  插件数: 6  │  评估单元: 120           │
├─────────────────────────────────────────────────────────────┤
│  总评                                                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  显著优于 Baseline  (p=0.003, d=0.68, ELO +65)      │   │
│  │  信念修正: 78% 通过  │  位置偏见: 未检测到          │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  插件详情（Tab 切换）                                         │
│  [BERTScore] [G-Eval] [信念修正] [约束满足] [客服量表]        │
│                                                             │
│  当前: 信念修正                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  通过率: 78% (39/50)  CI: [65%, 88%]                │   │
│  │                                                     │   │
│  │  失败 Case:                                         │   │
│  │  ┌───────────────────────────────────────────────┐ │   │
│  │  │ Case-003: 已知只剩1张票，仍推荐购买2张         │ │   │
│  │  │   → 失败模式: 忽视约束，未修正初始信念          │ │   │
│  │  │   → 修复建议: 在 Prompt 中添加约束检查步骤      │ │   │
│  │  └───────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 七、实施优先级

### Phase 1：核心框架（2 周）
- [ ] `plugin.go` 接口定义
- [ ] `registry.go` 注册表
- [ ] `orchestrator.go` 编排器（并行执行、超时控制）
- [ ] `injection.go` 注入策略框架
- [ ] `custom_rubric.go` YAML 解析 + 动态 Plugin 生成

### Phase 2：内置插件（并行，各 1 周）
- [ ] `bertscore/` 插件
- [ ] `geval/` 插件
- [ ] `beliefrevision/` 插件（明星航班场景）
- [ ] `constraint/` 插件

### Phase 3：统计层 + 前端（2 周）
- [ ] `stats/` Bootstrap + t-test + Cohen's d + ELO
- [ ] 前端：Orchestrator Config UI
- [ ] 前端：Multi-Plugin Report UI

### Phase 4：高级插件（2 周）
- [ ] `factscore/` 插件
- [ ] `selfcheck/` 插件
- [ ] `pairwise/` 插件

---

## 八、一句话总结

> **Eval Orchestrator 不是"跑一个脚本"，而是"编排一场评估实验"——你可以自由选择评估 Agent、组合注入策略、定义自己的量表，所有 Agent 并行执行，最后产出一份带统计显著性的实验报告。**
