# 推理模式分析：从"明星航班"到 LLM 评估

> 用户原话："假设我很了解一个明星，我会推导：按照我这个人的了解，她会飞下午的国航。所以下午2点这班最有可能。但是这一班只剩1张票了，她要和她经纪人一起，所以不可能买这班。不会玩太久，因为他家里养了狗。"

---

## 一、这句话的推理结构拆解

```
Step 1: 先验信念（Prior Belief）
  "我很了解她" → "她会飞下午的国航" → "下午2点这班最有可能"
  
Step 2: 约束发现（Constraint Discovery）
  "这一班只剩1张票" + "她要和她经纪人一起" → 需要2张票
  
Step 3: 信念修正（Belief Revision）
  初始信念（2点航班）与约束冲突 → 排除2点航班
  
Step 4: 常识推理（Commonsense Inference）
  "家里养了狗" → 狗需要照顾 → 不会玩太久 → 需要早点回去
  
Step 5: 最终决策（Decision Under Constraints）
  在满足所有约束的前提下选择最优航班
```

**这不是简单的"问答"，而是五种认知能力的复合：**

| 认知能力 | 在这个场景中的体现 | 现有 LLM 评估是否覆盖 |
|---------|-------------------|---------------------|
| **溯因推理** | 从"了解她"推断行为模式 | ❌ MMLU/GSM8K 不测 |
| **约束满足** | 票数限制、人数限制 | ⚠️ 少数数学题涉及 |
| **信念修正** | 发现冲突后主动排除原选项 | ❌ 几乎不测试 |
| **常识推理** | 养狗→需要照顾→不能玩太久 | ⚠️ CommonsenseQA 测但不系统 |
| **多跳推理** | 养狗→早归→选更早航班（3跳） | ⚠️ 部分数据集覆盖 |

---

## 二、为什么现有 LLM 评估测不到这个

当前主流的 LLM 评估范式：

1. **MMLU / C-Eval**：知识问答，单步事实检索
2. **GSM8K / MATH**：数学推理，但约束是明确的、符号化的
3. **HumanEval / MBPP**：代码生成，确定性输出
4. **MT-Bench / Arena**：对话质量，主观偏好

**共同缺陷**：
- 测试的是**单点能力**，不是**认知流水线**
- 没有**冲突检测**：模型不会遇到"你的第一个答案可能是错的"这种情况
- 没有**常识与约束的交织**：数学题不会突然出现"因为家里养狗所以时间不够"
- 没有**信念修正的验证**：即使模型第一步错了，评估只关心最终答案对不对，不关心它有没有自我纠正

---

## 三、迁移到 Eval 体系："复合推理评估框架"

### 3.1 核心洞察

这句话揭示了一个被忽视的评估维度：

> **LLM 不是不会推理，而是不会在"推理过程中遇到意外约束并修正自己的信念"。**

现有 Prompt 评估假设：
- 给模型一个 task → 模型直接输出答案 → 我们对答案打分

真实世界的要求：
- 给模型一个 task → 模型做初步计划 → **发现约束冲突** → **修正计划** → 输出最终答案

### 3.2 新增评估维度："认知流水线"

基于这个推理模式，我们的评估体系应该增加一个**认知阶段跟踪**（Cognitive Stage Tracking）：

```
评估不是看最终答案对不对，而是看每一步认知是否正确。

Case: 明星航班预测
├─ [Step 1] 先验推理: "2点航班最有可能" → 期望: 正确 / 可接受
├─ [Step 2] 约束识别: "只剩1张票 + 需要2张" → 期望: 必须识别
├─ [Step 3] 信念修正: "排除2点航班" → 期望: 必须排除
├─ [Step 4] 常识推理: "养狗→早归" → 期望: 正确推断
└─ [Step 5] 最终决策: "选择满足所有约束的航班" → 期望: 正确

如果模型 Step 3 没做（仍然推荐2点航班）→ 标记为"信念修正失败"
如果模型 Step 4 没做（推荐晚上航班）→ 标记为"常识推理失败"
```

### 3.3 对应到我们的 Go Eval Engine

这个场景直接催生出三个新的 Metric：

#### Metric A: BeliefRevisionScore（信念修正能力）

**定义**：模型在推理过程中，当遇到与初始假设冲突的证据时，能否主动修正先前结论。

**测试方法**：
```
Question: "根据你的了解，小明最喜欢吃什么？"
  → 模型回答: "火锅"

Follow-up: "但小明上周刚做了胃切除手术，医生说他不能吃辛辣食物。"
  → 期望模型修正: "那他不能吃火锅了，可能吃粥或清淡的食物"
  → 如果模型坚持"火锅" → 信念修正失败
```

**评分**：
- 1.0: 主动修正并给出合理替代
- 0.5: 承认冲突但没有给出替代
- 0.0: 忽视冲突，坚持原答案

**Go 实现**：
- 两阶段提问（先诱导一个答案，再给出冲突证据）
- Judge LLM 评估第二阶段回答是否包含"修正"

#### Metric B: ConstraintSatisfactionScore（约束满足能力）

**定义**：模型能否在复杂约束条件下找到可行解，而不是只优化单一目标。

**测试方法**：
```
Question: "安排一个3人的晚餐，要求：
  1. 预算不超过500元
  2. 其中1人是素食者
  3. 餐厅距离公司不超过2公里
  4. 周五晚上8点有位置"
  
→ 模型必须同时满足4个约束，遗漏任何一个都是失败
```

**评分**：
- 1.0: 满足所有约束
- 0.0: 遗漏任一约束

**Go 实现**：
- 用正则/关键词检查输出中是否提及每个约束
- Judge LLM 验证最终方案的可行性

#### Metric C: CommonsenseConsistencyScore（常识一致性）

**定义**：模型的推理链中，常识知识的使用是否自洽。

**测试方法**：
```
Question: "为什么这个明星不会玩太久？"
→ 模型需要推断出"养狗→需要照顾→不能久留"的链条
→ 如果模型说"因为她不喜欢玩" → 与已知情境不一致（没有证据）
→ 如果模型说"因为她要回去喂狗" → 正确
```

**评分**：
- 用 SelfCheckGPT 方法：对同一个常识推理问题采样多次，检查一致性
- 或用 BERTScore 比较模型推理链与"标准推理链"的语义相似度

---

## 四、具体迁移：设计一个"明星航班"评估 Case

### 4.1 Eval Case 结构

```yaml
# prompts/eval/cases/reasoning-chain-001.yaml
id: reasoning-chain-001
category: belief_revision
difficulty: medium

description: |
  测试模型在约束冲突情况下的信念修正能力。

steps:
  - id: step-1
    type: prior_inference
    prompt: |
      根据以下信息，推测小李今天会乘坐哪个航班回北京：
      - 小李不喜欢早起
      - 小李偏爱国航
      - 今天下午国航有3个航班：14:00、16:00、18:00
    expected_reasoning: |
      小李不喜欢早起 → 不会选早班机
      偏爱国航 → 从国航下午航班中选
      最可能选 14:00（最符合"不早起"且是最早的下午航班）
    evaluation: |
      不要求最终答案唯一正确，但推理链必须合理。

  - id: step-2
    type: constraint_discovery
    prompt: |
      新信息：
      - 14:00 航班只剩 1 张票
      - 小李要和经纪人一起回北京（需要 2 张票）
      
      结合之前的信息，现在小李最可能乘坐哪个航班？
    expected_reasoning: |
      之前认为 14:00 最可能 → 但只剩1张票，需要2张 → 14:00 不可行
      剩余选项：16:00、18:00
      结合"不喜欢早起"（已满足）和"不会玩太久"（见step-3）做最终选择
    evaluation: |
      必须明确排除 14:00 航班，否则信念修正失败。

  - id: step-3
    type: commonsense_inference
    prompt: |
      额外信息：小李家里养了一只狗，平时都是她自己照顾。
      
      这个信息对你的推测有什么影响？为什么？
    expected_reasoning: |
      养狗需要照顾 → 不会在外面停留太久 → 倾向于选更早回北京的航班
      因此 16:00 比 18:00 更可能
    evaluation: |
      必须建立"养狗→不能久留→选早班"的推理链，不能跳过中间环节。

  - id: step-4
    type: final_decision
    prompt: |
      综合所有信息，小李今天最可能乘坐哪个航班？请给出完整推理过程。
    expected_answer: "16:00"
    evaluation: |
      最终答案必须是 16:00，且推理过程必须包含前三步的关键逻辑。
```

### 4.2 自动评分逻辑（Go）

```go
// 对每个 step 独立评分
func EvaluateReasoningChain(case Case, modelOutput string) *ReasoningChainResult {
    result := &ReasoningChainResult{
        StepScores: make(map[string]float64),
    }

    // Step 1: 先验推理（BERTScore 或 G-Eval）
    result.StepScores["step-1"] = gEval.Evaluate(case.Steps[0].ExpectedReasoning, modelOutput)

    // Step 2: 约束满足 + 信念修正（关键词检查 + Judge）
    hasExcluded := strings.Contains(modelOutput, "14:00") && 
                   strings.Contains(modelOutput, "不可能") ||
                   strings.Contains(modelOutput, "排除") ||
                   strings.Contains(modelOutput, "只剩")
    if hasExcluded {
        result.StepScores["step-2"] = 1.0
    } else {
        result.StepScores["step-2"] = 0.0
        result.FailureMode = "belief_revision_failed"
    }

    // Step 3: 常识推理（SelfCheckGPT 一致性检测）
    // 多次采样同一个问题，检查是否都提到"养狗"和"早归"的关联
    result.StepScores["step-3"] = selfCheckConsistency(modelOutput, "养狗", "早归")

    // Step 4: 最终决策
    if strings.Contains(modelOutput, "16:00") {
        result.StepScores["step-4"] = 1.0
    } else {
        result.StepScores["step-4"] = 0.0
    }

    // 总分：加权平均，但 Step 2（信念修正）是硬性门槛
    if result.StepScores["step-2"] == 0 {
        result.Overall = 0.0  // 信念修正失败，整体不及格
        result.FailureMode = "belief_revision_failed"
    } else {
        result.Overall = weightedAverage(result.StepScores, []float64{0.2, 0.3, 0.2, 0.3})
    }

    return result
}
```

---

## 五、产品价值：这是现有工具都没有的

### 5.1 为什么这个评估场景值钱

**当前市面上的 LLM 评估工具**：
- 告诉用户"你的 Prompt 得了 82 分"
- 用户不知道这 82 分意味着什么
- 用户不知道模型在哪一步开始出错

**我们的"认知流水线评估"**：
- 告诉用户"你的 Prompt 在**信念修正**这一步失败率 40%"
- 告诉用户"模型遇到约束冲突时，有 60% 的概率忽视冲突继续原方案"
- 给出具体 case："在第 17 号测试 case 中，模型已知'只剩1张票'，但仍推荐购买该航班"

**这就是诊断能力**。

### 5.2 迁移到其他领域

这个推理模式不限于"明星航班"，可以泛化到：

| 领域 | 先验信念 | 约束冲突 | 常识推理 | 信念修正 |
|------|---------|---------|---------|---------|
| **客服 Agent** | "用户说要退款" | "但订单已超过7天" | "超时订单通常不能退" | 改为推荐换货 |
| **代码 Agent** | "用 Redis 缓存" | "但环境没有 Redis" | "可以用本地内存缓存替代" | 改为用 map |
| **医疗 Agent** | "开抗生素" | "但患者青霉素过敏" | "过敏不能用青霉素类" | 改为大环内酯类 |
| **招聘 Agent** | "推荐候选人A" | "但A薪资要求超预算" | "超预算需要特批" | 改为推荐候选人B |

**每一个领域都需要"复合推理评估"**。

---

## 六、结论

用户这句话揭示了一个核心需求：

> **我们的 eval 工具不仅要评估"模型答对了没"，还要评估"模型是怎么想的"——特别是在遇到意外约束时，它能不能像人一样修正自己的想法。**

这直接对应到我们的 Go Eval Engine 架构中的三个新 Metric：
1. **BeliefRevisionScore**（信念修正能力）
2. **ConstraintSatisfactionScore**（约束满足能力）
3. **CommonsenseConsistencyScore**（常识推理一致性）

这三个 Metric 在现有 LLM 评估文献中几乎没有系统性的研究，**如果我们先做出来了，这就是论文级的贡献**。
