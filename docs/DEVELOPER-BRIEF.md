# 给开发 Agent 的任务简报

## 一、他应该看哪几个文档

| 优先级 | 文档 | 看多久 | 看什么 |
|--------|------|--------|--------|
| **必看** | `docs/EVAL-ORCHESTRATOR-ARCHITECTURE.md` | 15 分钟 | 核心架构：Plugin 接口、Orchestrator、Injection、Custom Rubric |
| **必看** | `docs/EVAL-ENGINE-GO.md` | 10 分钟 | 每个指标的具体 Go 实现方案（BERTScore 怎么做、统计层怎么写） |
| **必看** | 本文件 (`docs/DEVELOPER-BRIEF.md`) | 5 分钟 | 现有代码怎么改、边界在哪 |
| **选看** | `docs/INFERENCE-PATTERN-ANALYSIS.md` | 5 分钟 | 理解"明星航班"场景为什么催生 BeliefRevision 指标 |
| **不看** | `docs/EVAL-RESEARCH-GRADE.md` | — | 这是给你老板看的论文引用，开发不用看 |
| **不看** | `docs/EVAL-UI-REDESIGN.md` | — | 前端已做完，后端不用关心 |

**> 给开发 Agent 的原话：**
> "读 `EVAL-ORCHESTRATOR-ARCHITECTURE.md` 了解要做什么，读 `EVAL-ENGINE-GO.md` 了解怎么做，读 `DEVELOPER-BRIEF.md` 了解在现有代码上怎么改。其他文档不用看。"

---

## 二、现有代码能复用多少？

**结论：改造可行性极高。现有架构和我们的设计天然契合。**

### 2.1 现有架构优势（不用改，直接复用）

| 现有组件 | 位置 | 怎么复用 |
|---------|------|---------|
| **Plugin 框架** | `plugins/plugins.go` + `internal/service/interfaces.go` | 已经有 `Plugin` 接口和 clean architecture。我们的 EvalPlugin 直接接入这套体系。 |
| **LLM 调用抽象** | `plugins/llm/invoker.go` | 已经有 `OpenAIProvider` / `ClaudeProvider` / `OllamaProvider`。只需要给每个 provider 加 `Embed(texts []string)` 方法就能做 BERTScore。 |
| **Worker Pool** | `internal/service/eval_executor.go` | 已经有 `Coordinator` + `Worker` 的并发模式。Orchestrator 的并行执行可以直接复用。 |
| **领域模型** | `internal/domain/eval_*.go` | `EvalRun`, `EvalExecution`, `EvalCase`, `EvalWorkItem` 基本够用，少量扩展即可。 |
| **Ent Schema** | `internal/storage/ent/schema/*.go` | 数据库 schema 已定义，只需要加字段（`metric_results` JSON 字段），不用新建表。 |
| **API Handler** | `internal/gateway/handlers/eval_handler.go` | 已有 `/api/v1/evals/execute` 等端点。新增 `/api/v1/evals/orchestrate` 端点即可。 |

### 2.2 必须新增的部分

| 新增组件 | 工作量 | 说明 |
|---------|--------|------|
| `Embedder` 接口 + 实现 | 小 | 复用现有 LLM provider，加 `Embed()` 方法。OpenAI 调用 `/embeddings`，Ollama 调用 `/api/embeddings`。 |
| `Judge` 接口 | 极小 | 其实就是现有 `LLMInvoker`，加 Temperature=0 的封装。 |
| `EvalPlugin` 接口 | 小 | 定义统一输入输出结构（见架构文档）。 |
| `Orchestrator` | 中 | 编排器：并行调度 Plugin、收集结果、调用统计层。 |
| `InjectionStrategy` | 小 | 注入策略框架 + 具体策略实现。 |
| `CustomRubric` 解析器 | 小 | YAML → Plugin 的动态生成。 |
| 统计层 (`stats/`) | 小 | Bootstrap CI、t-test、Cohen's d、ELO。纯数学，~500 行。 |
| 各 Plugin 实现 | 中 | bertscore、geval、beliefrevision、constraint、factscore、selfcheck。 |

### 2.3 必须修改的现有代码

| 现有代码 | 怎么改 |
|---------|--------|
| `EvalService.RunEval()` | 目前评分逻辑是硬编码的（甚至有个 TODO 注释说 score 没算）。需要替换为：调用 Orchestrator → 并行执行 Plugins → 汇总结果。 |
| `EvalRun` domain model | 加 `AssetID` 字段（目前 service 层有，domain 层没有）。加 `MetricResults` 字段存多指标结果。 |
| `EvalExecution` schema | 加 `plugin_results` JSON 字段，存各 Plugin 的输出。 |
| API Handler | 新增 `POST /api/v1/evals/orchestrate` 端点。 |

---

## 三、给开发 Agent 的具体指令

### 3.1 第一阶段任务（2 周）

**任务 1.1：基础设施（第 1 周前半）**

```
在 internal/service/eval/ 下创建：

1. plugin.go
   - 定义 EvalPlugin 接口（Name/Description/RequiredCapabilities/Evaluate）
   - 定义 EvalInput / EvalResult / Dimension / EvalDetail 结构体

2. embedder.go
   - 定义 Embedder 接口（Embed/Dimension）
   - 实现 OpenAIEmbedder：复用 plugins/llm 里的 HTTP client，调用 /embeddings
   - 实现 OllamaEmbedder：复用 plugins/llm 里的 HTTP client，调用 /api/embeddings
   - 注意：不要新建 HTTP client，复用现有 provider 的 client

3. judge.go
   - 定义 Judge 接口（Compare/Score）
   - 实现 LLMJudge：封装现有 LLMInvoker，强制 Temperature=0

4. registry.go
   - Plugin 注册表，支持内置 Plugin 自动注册 + 自定义 Rubric 动态注册
```

**任务 1.2：统计层（第 1 周后半）**

```
在 internal/service/eval/stats/ 下创建：

1. bootstrap.go
   - BootstrapCI(values []float64, confidence float64, n int) (low, high float64)
   
2. ttest.go
   - PairedTTest(before, after []float64) (tStat, pValue float64)
   
3. effect_size.go
   - CohensD(groupA, groupB []float64) float64
   
4. elo.go
   - UpdateELO(ratingA, ratingB, outcome float64) (newA, newB float64)
   
依赖：只用 Go 标准库（math, sort, math/rand）。
不要用 gonum，保持零依赖。
```

**任务 1.3：BERTScore Plugin（第 2 周）**

```
在 internal/service/eval/plugins/bertscore/ 下创建 plugin.go：

1. 实现 EvalPlugin 接口
2. Evaluate 逻辑：
   - 从 EvalInput 获取 candidate/reference 文本
   - 调用 Embedder.Embed() 获取向量（批量，一次请求）
   - 计算余弦相似度矩阵
   - 计算 Precision / Recall / F1
   - 返回 EvalResult（带 ConfidenceInterval，用 stats.BootstrapCI）

3. 分词策略：
   - 中文：简单按字符分（BERT 中文模型是字级别）
   - 英文：按空格分词
   
4. 论文合规：在 Metadata 里记录使用的 embedder 名称和维度
```

### 3.2 第二阶段任务（并行，各 1 周）

**任务 2.1：Orchestrator + 并行执行**

```
在 internal/service/eval/orchestrator.go：

1. EvalConfig 结构体（插件列表、注入策略、统计配置、并行度）
2. Orchestrator.Run() 方法：
   - 加载 Asset + TestCases
   - 应用 InjectionStrategy 生成变体
   - 用 errgroup 并行执行所有 Plugin（限制并发数）
   - 收集结果 → 计算 CI → vs Baseline 对比 → 更新 ELO
   - 返回 OrchestratorResult

3. 超时控制：每个 Plugin 单独 context.WithTimeout
```

**任务 2.2：G-Eval Plugin**

```
在 internal/service/eval/plugins/geval/：

1. 定义 criteria 模板（prompt template）
2. Evaluate：
   - 对每个 case，调用 Judge.Score()，Temperature=0
   - 多次采样（n=3 或 5）
   - 取平均，计算 Bootstrap CI
   - 返回 EvalResult
```

**任务 2.3：BeliefRevision + Constraint Plugins**

```
在 internal/service/eval/plugins/beliefrevision/：

1. 两阶段测试：
   - Round 1：诱导模型给出初始答案
   - Round 2：插入冲突约束，看模型是否修正
   
2. 评分：
   - 1.0：明确修正并给出替代
   - 0.5：承认冲突但没有替代
   - 0.0：忽视冲突

在 internal/service/eval/plugins/constraint/：

1. 检查模型输出是否满足所有约束条件
2. 用正则/关键词 + Judge 验证
```

### 3.3 第三阶段任务（2 周）

**任务 3.1：自定义 Rubric**

```
在 internal/service/eval/custom_rubric.go：

1. YAML 解析（用 gopkg.in/yaml.v3，项目已有）
2. 从 CustomRubric 动态生成 EvalPlugin
3. 启动时扫描 configs/rubrics/*.yaml 自动注册
4. API：上传/列出/预览自定义量表
```

**任务 3.2：Injection 策略**

```
在 internal/service/eval/injection.go：

1. InjectionStrategy 接口 + 具体实现
2. PositionSwap：生成 A/B 交换变体
3. ConstraintConflict：插入冲突约束（明星航班场景）
4. AdversarialPrefix：添加对抗性前缀
```

**任务 3.3：API Handler 适配**

```
在 internal/gateway/handlers/eval_handler.go：

1. 新增 POST /api/v1/evals/orchestrate
2. 请求体：EvalConfig（插件列表、注入策略、统计配置）
3. 返回：OrchestratorResult（多指标结果 + 统计显著性）
```

---

## 四、边界和约束

### 4.1 不许做的

- **不许引入 Python 依赖**：所有指标用 Go 实现，Embedding 通过 HTTP API 调用
- **不许引入重型 ML 库**：不用 TensorFlow Go、ONNX Runtime，保持零 CGO
- **不许改现有数据库表结构**：只加 JSON 字段，不改列、不删表
- **不许破坏现有 API**：`/api/v1/evals/execute` 继续工作，新增 `/orchestrate` 端点

### 4.2 必须复用的

- **复用 `plugins/llm` 的 HTTP client**：不要新建 client，给 provider 加 `Embed()` 方法
- **复用 `eval_executor.go` 的 Coordinator**：并行执行直接用这个模式
- **复用 Ent 的 schema 定义**：在现有 schema 上加字段，用 Ent 的 migration

### 4.3 代码规范

- 所有 Plugin 放在 `internal/service/eval/plugins/{name}/`
- 所有统计函数放在 `internal/service/eval/stats/`
- 接口定义在 `internal/service/eval/plugin.go`
- 论文引用用常量定义在各自 Plugin 文件顶部

---

## 五、验收标准

### 5.1 功能验收

```bash
# 1. 能跑通 BERTScore
curl -X POST /api/v1/evals/orchestrate \
  -d '{"asset_id":"test-001","plugins":["bertscore"]}'
# 返回结果包含 precision/recall/f1 + 95% CI

# 2. 能跑多插件组合
curl -X POST /api/v1/evals/orchestrate \
  -d '{"asset_id":"test-001","plugins":["bertscore","geval","beliefrevision"]}'
# 返回包含 3 个插件的结果

# 3. 能加载自定义 Rubric
curl -X POST /api/v1/evals/rubrics -F file=@my-rubric.yaml
curl -X POST /api/v1/evals/orchestrate \
  -d '{"plugins":["custom:my-rubric"]}'

# 4. 有统计显著性
# 返回结果包含 p-value、Cohen's d、置信区间
```

### 5.2 性能验收

- 单个 Plugin 评估 20 个 cases：BERTScore < 3 秒，G-Eval < 10 秒
- 5 个 Plugin 并行评估 20 个 cases：总时间 < 15 秒
- 内存占用：评估过程中不加载本地模型（Embedding 走 API）

---

## 六、常见坑（提前告知开发 Agent）

| 坑 | 原因 | 解法 |
|---|------|------|
| `coordinators sync.Map` 是空的 | `RunEval` 从来没往里面存 | 不用修，我们的 Orchestrator 用自己的并发控制（errgroup） |
| `EvalRun` domain 没有 `AssetID` | 历史遗留 | 加字段，不影响现有代码 |
| 文件存储 vs 数据库存储 | `RunEval` 用文件，`Ent schema` 没真正用 | 新架构继续用文件存储（简单），或可选迁移到数据库 |
| Embedding API 速率限制 | OpenAI Embedding 有 TPM 限制 | 批量请求 + 指数退避重试 |
| Judge 位置偏见 | GPT-4 偏爱前面的答案 | 必须做 Swap Test，不一致则标记为 uncertain |

---

## 七、一句话总结给开发 Agent

> **你在做一个"实验编排器"：定义 Plugin 接口 → 实现几个评估 Agent（BERTScore、G-Eval、信念修正）→ 用 errgroup 并行调度 → 最后套一层统计检验（Bootstrap CI + t-test）。全部在现有代码上改，不引入 Python，不破坏现有 API。**
