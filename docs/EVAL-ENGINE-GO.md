# Eval Engine Go：轻量级科研级评估引擎

> 原则：借鉴论文思想，全部用 Go 实现，不依赖 Python 生态，复用现有 LLM Provider 基础设施。

---

## 一、架构总览

```
┌─────────────────────────────────────────────────────────────┐
│                    Eval Engine (Go)                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │  Embedder   │  │    Judge    │  │   KnowledgeBase     │ │
│  │  Interface  │  │  Interface  │  │     Interface       │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
│         │                │                    │            │
│         ▼                ▼                    ▼            │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              Metric Plugins (统一接口)               │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐ │  │
│  │  │BERTScore│ │ G-Eval  │ │FACTScore│ │SelfCheck │ │  │
│  │  │  (嵌入)  │ │ (LLM)   │ │ (检索)  │ │  (采样)  │ │  │
│  │  └─────────┘ └─────────┘ └─────────┘ └──────────┘ │  │
│  └─────────────────────────────────────────────────────┘  │
│                           │                                │
│                           ▼                                │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              Statistics Layer (纯 Go)                │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐ │  │
│  │  │Bootstrap│ │  t-test │ │Cohen's d│ │  ELO    │ │  │
│  │  │   CI    │ │/Wilcoxon│ │ Effect  │ │ Ranking │ │  │
│  │  └─────────┘ └─────────┘ └─────────┘ └──────────┘ │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 二、基础设施层：复用现有能力

### 2.1 Embedder：获取文本嵌入向量

**不装 Python，怎么做 BERT 嵌入？**

方案：**复用现有 LLM Provider 的 Embedding API**。

```go
// internal/service/eval/embedder.go

type Embedder interface {
    // Embed 获取文本的嵌入向量
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    // Dimension 返回向量维度（用于预分配）
    Dimension() int
}

// OpenAIEmbedder：复用现有的 OpenAI provider 配置
type OpenAIEmbedder struct {
    client *openai.Client  // 复用 internal/llm 里的 client
    model  string           // "text-embedding-3-small" 或用户配置的
}

// OllamaEmbedder：本地模型，零额外成本
type OllamaEmbedder struct {
    endpoint string
    model    string  // "nomic-embed-text" 或 "mxbai-embed-large"
}
```

**为什么可行？**
- OpenAI Embedding API：`text-embedding-3-small` 1536 维，便宜到几乎免费
- Ollama 本地跑：`nomic-embed-text` 是 137M 参数，CPU 都能跑，完全离线
- 我们自己的 LLM 配置里已经有 provider + api_key + endpoint，直接复用

**Fallback 策略**：
- 如果用户没有配置 Embedding，用基于字符 n-gram 的简化语义相似度（不是 BERTScore，但比 BLEU 好）
- 或者干脆报错："请配置 Embedding provider 以启用 BERTScore"

### 2.2 Judge：评估模型

```go
// internal/service/eval/judge.go

type Judge interface {
    // Compare 成对比较：返回 A 是否优于 B，以及理由
    Compare(ctx context.Context, criteria string, answerA, answerB string) (*ComparisonResult, error)
    // Score 绝对打分（G-Eval 用）：返回分数 + 理由
    Score(ctx context.Context, criteria string, answer string) (*ScoreResult, error)
}

type ComparisonResult struct {
    Winner     string  // "A", "B", "tie"
    Confidence float64 // 0-1
    Reasoning  string  // CoT 理由
}
```

**复用现有 LLM 基础设施**：
- 复用 `internal/llm` 里的 client
- Temperature 强制设为 0
- 支持多次采样（n=3 或 n=5）

### 2.3 KnowledgeBase：事实验证来源

```go
// internal/service/eval/knowledge.go

type KnowledgeBase interface {
    // Verify 验证一个原子事实是否成立
    Verify(ctx context.Context, atomicFact string) (*VerificationResult, error)
}

// 实现1：Wikipedia API（HTTP 请求，零依赖）
type WikipediaKB struct{}

// 实现2：本地向量检索（如果用户有文档库）
type VectorRetrievalKB struct {
    embedder Embedder
    store    VectorStore  // 复用 internal/storage
}

// 实现3：搜索引擎（Serper/Bing API）
type WebSearchKB struct {
    apiKey string
}
```

---

## 三、Metric 插件：每个指标的 Go 实现

### 3.1 BERTScore（Zhang et al., ICLR 2020）

**核心公式**：
```
Precision = (1/|x|) * Σ max(cos(x_i, y_j))   // 生成文本的每个token找参考文本最相似的
Recall    = (1/|y|) * Σ max(cos(x_i, y_j))   // 参考文本的每个token找生成文本最相似的
F1        = 2 * P * R / (P + R)
```

**Go 实现**（不依赖 PyTorch）：

```go
// internal/service/eval/metrics/bertscore.go

func BERTScore(candidate, reference string, embedder Embedder) (*BERTScoreResult, error) {
    // 1. 分词（简单空格分词，中文用 gojieba 或简单字符分词）
    candTokens := tokenize(candidate)
    refTokens := tokenize(reference)

    // 2. 获取嵌入向量（批量调用 Embedder）
    allTokens := append(candTokens, refTokens...)
    embeddings, err := embedder.Embed(ctx, allTokens)
    if err != nil { return nil, err }

    candEmbeds := embeddings[:len(candTokens)]
    refEmbeds := embeddings[len(candTokens):]

    // 3. 计算余弦相似度矩阵
    similarities := cosineSimilarityMatrix(candEmbeds, refEmbeds)

    // 4. 计算 Precision / Recall / F1
    precision := maxPerRow(similarities).Mean()
    recall := maxPerCol(similarities).Mean()
    f1 := 2 * precision * recall / (precision + recall)

    return &BERTScoreResult{Precision: precision, Recall: recall, F1: f1}, nil
}

// 余弦相似度（纯 Go，无依赖）
func cosineSimilarity(a, b []float32) float32 {
    var dot, normA, normB float32
    for i := range a {
        dot += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    return dot / (float32(math.Sqrt(float64(normA * normB))))
}
```

**计算量**：
- 假设 candidate 50 tokens，reference 50 tokens
- Embedding API 一次请求最多 2048 tokens，可以一次搞定
- 余弦相似度矩阵 50×50 = 2500 次计算，Go 毫秒级

**论文合规性**：
- 记录使用的 Embedding 模型名称和维度
- 记录分词方式（空格 / jieba）
- 可选：idf rescaling（简单 map 统计 token 频率）

### 3.2 G-Eval（Liu et al., EMNLP 2023）

**核心**：CoT + LLM 评分

**Go 实现**：

```go
// internal/service/eval/metrics/geval.go

var gEvalPromptTemplate = template.Must(template.New("geval").Parse(`
You are an expert evaluator. Evaluate the following response based on the criterion.

Criterion: {{.Criteria}}

Response: {{.Response}}

Provide your evaluation in two steps:
1. Think step by step about the strengths and weaknesses of the response.
2. Give a score from 1 to 5.

Output format:
Reasoning: <your step-by-step reasoning>
Score: <integer from 1 to 5>
`))

func GEval(ctx context.Context, judge Judge, criteria, response string) (*GEvalResult, error) {
    // Temperature 0，多次采样
    scores := make([]int, 0, nSamples)
    reasonings := make([]string, 0, nSamples)

    for i := 0; i < nSamples; i++ {
        result, err := judge.Score(ctx, criteria, response)
        if err != nil { continue }
        scores = append(scores, result.Score)
        reasonings = append(reasonings, result.Reasoning)
    }

    mean := float64(sum(scores)) / float64(len(scores))
    ci := bootstrapCI(scores, 0.95)

    return &GEvalResult{
        Mean:       mean,
        CI:         ci,
        Reasonings: reasonings,
    }, nil
}
```

### 3.3 FACTScore（Min et al., ACL 2023）

**核心**：原子事实分解 + 独立验证

**Go 实现**：

```go
// internal/service/eval/metrics/factscore.go

var atomicFactPrompt = `Extract atomic facts from the following text.
An atomic fact is a single, verifiable claim.

Text: {{.Text}}

Output one fact per line, prefixed with "- ".
`

func FACTScore(ctx context.Context, judge Judge, kb KnowledgeBase, text string) (*FACTScoreResult, error) {
    // 1. LLM 原子化分解
    facts, err := extractAtomicFacts(ctx, judge, text)
    if err != nil { return nil, err }

    // 2. 并行验证每个事实
    var verified, unverified, falseCount int
    var falseFacts []FalseFact

    for _, fact := range facts {
        result, err := kb.Verify(ctx, fact)
        if err != nil {
            unverified++
            continue
        }
        switch result.Status {
        case True:
            verified++
        case False:
            falseCount++
            falseFacts = append(falseFacts, FalseFact{Fact: fact, Correction: result.Correction})
        case Unknown:
            unverified++
        }
    }

    total := len(facts)
    score := float64(verified) / float64(total)

    return &FACTScoreResult{
        Total:        total,
        Verified:     verified,
        Unverified:   unverified,
        False:        falseCount,
        Score:        score,
        FalseFacts:   falseFacts,
    }, nil
}
```

**知识库验证策略（轻量）**：
- 默认用 Wikipedia API（HTTP GET，零依赖）
- 或让用户配置搜索引擎 API（Serper.dev 有免费额度）
- 不做本地 heavy RAG，保持轻量

### 3.4 SelfCheckGPT（Manakul et al., ACL 2023）

**核心**：多次采样 + 一致性检测

**Go 实现**：

```go
// internal/service/eval/metrics/selfcheck.go

func SelfCheckGPT(ctx context.Context, judge Judge, embedder Embedder, question string, n int) (*SelfCheckResult, error) {
    // 1. 对同一个问题采样 N 次
    answers := make([]string, 0, n)
    for i := 0; i < n; i++ {
        ans, err := judge.Generate(ctx, question)
        if err != nil { continue }
        answers = append(answers, ans)
    }

    // 2. 用嵌入相似度检测一致性
    embeddings, err := embedder.Embed(ctx, answers)
    if err != nil { return nil, err }

    // 3. 计算每对答案的相似度
    inconsistencies := []Inconsistency{}
    for i := range answers {
        for j := i + 1; j < len(answers); j++ {
            sim := cosineSimilarity(embeddings[i], embeddings[j])
            if sim < threshold {
                inconsistencies = append(inconsistencies, Inconsistency{
                    AnswerA: answers[i], AnswerB: answers[j], Similarity: sim,
                })
            }
        }
    }

    // 4. 整体幻觉概率 = 不一致对数 / 总对数
    totalPairs := len(answers) * (len(answers) - 1) / 2
    hallucinationProb := float64(len(inconsistencies)) / float64(totalPairs)

    return &SelfCheckResult{
        HallucinationProb: hallucinationProb,
        Inconsistencies:   inconsistencies,
    }, nil
}
```

### 3.5 成对比较 + ELO（Zheng et al., NeurIPS 2023）

**核心**：Judge 做 A vs B，Bradley-Terry 模型推断全局排名

**Go 实现**：

```go
// internal/service/eval/metrics/pairwise.go

func PairwiseCompare(ctx context.Context, judge Judge, criteria, answerA, answerB string) (*PairwiseResult, error) {
    // Swap Test：交换顺序测两次，消除位置偏见
    results := make([]string, 0, 2)

    for _, swapped := range []bool{false, true} {
        var first, second string
        if swapped { first, second = answerB, answerA }
        else { first, second = answerA, answerB }

        result, err := judge.Compare(ctx, criteria, first, second)
        if err != nil { continue }

        // 把结果映射回原始顺序
        winner := result.Winner
        if swapped {
            if winner == "A" { winner = "B" }
            else if winner == "B" { winner = "A" }
        }
        results = append(results, winner)
    }

    // 如果两次结果不一致，说明 Judge 有位置偏见，标记为不确定
    if len(results) == 2 && results[0] != results[1] {
        return &PairwiseResult{Winner: "uncertain", Reason: "position_bias_detected"}, nil
    }

    return &PairwiseResult{Winner: results[0]}, nil
}

// ELO 更新（纯数学）
func UpdateELO(ratingA, ratingB float64, outcome float64) (newA, newB float64) {
    // outcome: 1 = A wins, 0.5 = tie, 0 = B wins
    expectedA := 1 / (1 + math.Pow(10, (ratingB-ratingA)/400))
    expectedB := 1 / (1 + math.Pow(10, (ratingA-ratingB)/400))

    k := 32.0
    newA = ratingA + k*(outcome-expectedA)
    newB = ratingB + k*((1-outcome)-expectedB)
    return
}
```

---

## 四、统计层：纯 Go 实现

### 4.1 Bootstrap 置信区间

```go
// internal/service/eval/stats/bootstrap.go

func BootstrapCI(values []float64, confidence float64, nBootstrap int) (low, high float64) {
    n := len(values)
    means := make([]float64, nBootstrap)

    for i := 0; i < nBootstrap; i++ {
        sum := 0.0
        for j := 0; j < n; j++ {
            idx := rand.Intn(n)
            sum += values[idx]
        }
        means[i] = sum / float64(n)
    }

    sort.Float64s(means)
    alpha := 1 - confidence
    lowIdx := int(float64(nBootstrap) * alpha / 2)
    highIdx := int(float64(nBootstrap) * (1 - alpha/2))

    return means[lowIdx], means[highIdx]
}
```

### 4.2 成对 t 检验

```go
// internal/service/eval/stats/ttest.go

func PairedTTest(before, after []float64) (tStat float64, pValue float64) {
    // 计算差异
    diffs := make([]float64, len(before))
    for i := range before {
        diffs[i] = after[i] - before[i]
    }

    meanDiff := mean(diffs)
    sdDiff := stdDev(diffs)
    n := float64(len(diffs))

    tStat = meanDiff / (sdDiff / math.Sqrt(n))
    // p-value 用 t 分布的 CDF（gonum 有，或自己查表近似）
    pValue = pValueFromT(tStat, len(diffs)-1)
    return
}
```

### 4.3 Cohen's d

```go
func CohensD(groupA, groupB []float64) float64 {
    meanA, meanB := mean(groupA), mean(groupB)
    sdA, sdB := stdDev(groupA), stdDev(groupB)

    // Pooled standard deviation
    nA, nB := float64(len(groupA)), float64(len(groupB))
    pooledSD := math.Sqrt(((nA-1)*sdA*sdA + (nB-1)*sdB*sdB) / (nA + nB - 2))

    return (meanA - meanB) / pooledSD
}
```

**依赖**：
- `math` 标准库
- `sort` 标准库
- `math/rand` 标准库
- 可选：`gonum/stat`（如果要用更完善的统计函数，但也可以自己写）

---

## 五、API 设计

### 5.1 执行评估

```http
POST /api/v1/evals/execute
Content-Type: application/json

{
  "asset_id": "prompt-001",
  "metrics": ["bertscore", "geval", "factscore", "pairwise"],
  "baseline_snapshot": "v1.4",
  "config": {
    "embedder": "openai:text-embedding-3-small",
    "judge": "gpt-4o",
    "judge_temperature": 0,
    "judge_n_samples": 3,
    "bootstrap_n": 1000,
    "confidence_level": 0.95
  }
}
```

### 5.2 返回结果

```json
{
  "execution_id": "exec_xyz",
  "status": "completed",
  "summary": {
    "vs_baseline": {
      "conclusion": "significantly_better",
      "p_value": 0.003,
      "cohens_d": 0.68,
      "elo_delta": 65
    }
  },
  "metrics": {
    "bertscore": {
      "precision": 0.84,
      "recall": 0.79,
      "f1": 0.81,
      "ci_95": [0.79, 0.84],
      "paper": "Zhang et al. ICLR 2020"
    },
    "geval": {
      "criteria": {
        "instruction_following": {"mean": 4.3, "ci_95": [4.1, 4.5]},
        "factual_accuracy": {"mean": 3.8, "ci_95": [3.5, 4.1]}
      },
      "paper": "Liu et al. EMNLP 2023"
    },
    "pairwise": {
      "winner": "current",
      "swap_test_passed": true,
      "judge_consistency": 0.87
    }
  },
  "reproducibility": {
    "seed": 42,
    "embedder": "openai:text-embedding-3-small",
    "judge": "gpt-4o",
    "command": "curl -X POST /api/v1/evals/reproduce -d '{\"execution_id\":\"exec_xyz\"}'"
  }
}
```

---

## 六、前端界面：实验报告风格

前端不需要大改，只需调整数据展示方式：

1. **EvalReportView** 改为"实验报告"布局：
   - 顶部：vs Baseline 结论（显著/不显著）
   - 中部：每个 Metric 的详细结果（带 CI、论文引用）
   - 底部：复现命令

2. **EvalHistoryView** 增加统计列：
   - p-value、Cohen's d、CI
   - ELO 排名变化

3. **新增 Judge 校准页面**：
   - 显示 Judge 自一致性、Swap Test 通过率、人类对齐 Kappa

---

## 七、实施路线图（全都要，但分批）

### Week 1-2：统计基础设施 + BERTScore
- [ ] Bootstrap CI、paired t-test、Cohen's d（纯 Go，~500 行）
- [ ] Embedder 接口 + OpenAI/Ollama 实现
- [ ] BERTScore Go 实现（嵌入 + 余弦相似度矩阵）
- [ ] 前端：所有指标显示 CI

### Week 3-4：成对比较 + ELO
- [ ] Judge 接口 + 成对比较 API
- [ ] Swap Test 位置偏见校正
- [ ] Bradley-Terry ELO 计算
- [ ] 前端：版本对比矩阵 + ELO 排名

### Week 5-6：G-Eval + FACTScore
- [ ] G-Eval（CoT prompt + 多次采样）
- [ ] FACTScore（原子化 + Wikipedia API 验证）
- [ ] 前端：诊断工作台（失败聚类树 + Case 对比 + 修复建议）

### Week 7-8：SelfCheckGPT + Judge 校准
- [ ] SelfCheckGPT（多次采样 + 嵌入一致性）
- [ ] Judge 一致性报告
- [ ] 多 Judge 投票

### Week 9-10：Agent 评估
- [ ] 端到端任务成功率
- [ ] 轨迹偏离度
- [ ] 工具调用链分析

---

## 八、轻量性保证

| 组件 | 依赖 | 大小 |
|------|------|------|
| BERTScore | HTTP 调用 Embedding API | 0 MB 本地模型 |
| G-Eval | HTTP 调用 LLM API | 0 MB 本地模型 |
| FACTScore | HTTP 调用 Wikipedia/搜索 API | 0 MB 本地模型 |
| SelfCheckGPT | HTTP 调用 Embedding + LLM API | 0 MB 本地模型 |
| 统计计算 | Go 标准库 | 0 外部依赖 |
| ELO | Go 标准库 | 0 外部依赖 |

**总外部依赖**：0 个 Python 包，0 个本地模型文件（如果用户用 OpenAI），或 1 个 137M 的 Ollama embedding 模型（如果用户要离线）。

**代码量预估**：
- 统计层：~500 行
- 指标插件（5 个）：~2000 行
- Embedder/Judge 接口：~500 行
- API 层：~500 行
- **总计：~3500 行纯 Go**
