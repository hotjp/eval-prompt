# Eval 界面体验优化方案：基于场景的视图拆分

## 一、Eval 生命周期与场景模型

Prompt eval 不是单点操作，而是一个**设计 → 运行 → 分析 → 迭代**的闭环。我们把用户旅程拆为 5 个阶段：

```
┌─────────┐    ┌──────────┐    ┌─────────┐    ┌──────────┐    ┌─────────┐
│  Design │ → │ Configure│ → │   Run   │ → │ Analyze  │ → │ Compare │
│  (设计)  │    │  (配置)   │    │  (执行)  │    │  (分析)   │    │ (演进)  │
└─────────┘    └──────────┘    └─────────┘    └──────────┘    └─────────┘
     ↑___________________________________________________________|
```

### 阶段定义与场景

| 阶段 | 核心问题 | 典型场景 | 当前痛点 |
|------|---------|---------|---------|
| **Design** | Cases 是否覆盖全面？ | 新增/修改 test case；调整 metric 引用 | EvalCasesView 只读，Cases 和 Prompt 编辑分离 |
| **Configure** | 用什么参数跑？ | 选择模型、温度、并发、模式 | 配置项和结果混在一起，没有参数模板 |
| **Run** | 跑得怎么样了？ | 点击运行后等待，想随时看进度 | 进度条只在 `/eval` 页可见，切走就丢失上下文 |
| **Analyze** | 为什么挂了？ | 看 rubric 失败详情；下钻到具体 LLM 调用 | Rubric details 被 ellipsis 截断；无法直接看 prompt/response |
| **Compare** | 这次改好了吗？ | 对比两次 eval 结果；对比两个版本 | CompareView 只有总分数，没有 rubric 级逐条对比 |

---

## 二、视图拆分方案

保留 `/assets/:id/eval` 作为主入口，内部用 **子路由 + 阶段导航** 拆分为 4 个独立视图：

```
/assets/:id/eval
├── /design        → EvalDesignView    (原 EvalCasesView 增强)
├── /run           → EvalRunView       (原 EvalPanelView 执行部分)
├── /report/:runId → EvalReportView    (新增：单轮深度报告)
└── /history       → EvalHistoryView   (原 EvalPanelView 趋势+历史)
```

### 2.1 EvalDesignView — 测试用例与指标设计

**解决场景**：Cases 维护、Metrics 关联

**布局**：
```
┌─────────────────────────────────────────────────────────────┐
│  [Prompt 信息卡片]                                           │
├──────────────────────────────┬──────────────────────────────┤
│  Test Cases 列表              │  关联 Metrics                │
│  ┌────────────────────────┐  │  ┌────────────────────────┐  │
│  │ 🔍 Search/filter cases │  │  │ + Add Metric Ref       │  │
│  ├────────────────────────┤  │  ├────────────────────────┤  │
│  │ ☑ Case 1: 简短描述      │  │  │ 📎 metric-001          │  │
│  │   input: xxx            │  │  │ 📎 metric-002          │  │
│  │   expected: xxx         │  │  └────────────────────────┘  │
│  │   [Run this case ▶]     │  │                              │
│  ├────────────────────────┤  │                              │
│  │ ☑ Case 2: ...           │  │                              │
│  └────────────────────────┘  │                              │
│  [+ Add Case]  [Run Selected]│                              │
└──────────────────────────────┴──────────────────────────────┘
```

**关键改进**：
- Cases 可编辑（inline edit 或抽屉编辑），不再只读
- 每个 case 增加复选框，支持"选中部分 cases 运行"
- 右侧常驻显示关联的 metrics，点击可跳转 metric 详情
- 增加 `Run this case` 快捷按钮，支持单 case 调试

### 2.2 EvalRunView — 执行配置与实时状态

**解决场景**：配置参数、启动执行、实时监控

**布局**：
```
┌─────────────────────────────────────────────────────────────┐
│  [Prompt 信息卡片]                                           │
├─────────────────────────────────────────────────────────────┤
│  执行配置卡片                                                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │  Model   │ │   Temp   │ │  Mode    │ │Concurrent│       │
│  │ ▼ gpt-4o │ │  0.7     │ │ ▼ single│ │    4     │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│  [💾 Save as Preset]    [▶ Start Eval]  [⏹ Cancel]          │
├─────────────────────────────────────────────────────────────┤
│  实时状态区（有执行时展开）                                    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ ████████░░░░ 67%  (8/12 cases)  running...          │    │
│  │ Model: gpt-4o | Temp: 0.7 | Started: 14:32          │    │
│  │ [实时日志滚动] ...                                  │    │
│  └─────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│  最近一轮结果快照（执行完成后自动展示）                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ Overall  │ │Determined│ │  Rubric  │ │ Pass Rate│       │
│  │   0.82   │ │   0.90   │ │   0.74   │ │   75%    │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│  [查看完整报告 →]  [保存为 Baseline]                          │
└─────────────────────────────────────────────────────────────┘
```

**关键改进**：
- 模型选择从后端 `/api/v1/llm-config` 动态拉取，不再 hardcode 4 个模型
- 增加 **Preset（预设模板）** 功能，常用参数组合可保存/复用
- 实时状态区常驻，执行期间显示进度条 + 实时日志流（SSE 或轮询）
- 执行完成后，本页直接展示结果快照，并提供"查看完整报告"入口跳转 `/report/:runId`
- 增加全局 **Eval 执行悬浮指示器**（详见 3.1）

### 2.3 EvalReportView — 单轮深度报告

**解决场景**：分析一次 eval 为何成功/失败

**布局**：
```
┌─────────────────────────────────────────────────────────────┐
│  [Prompt 信息卡片]  [Run: abc123] [Status: ✅ Passed]        │
├─────────────────────────────────────────────────────────────┤
│  分数概览                                                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ Overall  │ │Determined│ │  Rubric  │ │ Pass Rate│       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
├─────────────────────────────────────────────────────────────┤
│  Rubric 逐条诊断                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ ✅ Check 1: 格式合规     Score: 1.0                 │    │
│  │    └─ Details...                                    │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ ❌ Check 2: 事实准确性   Score: 0.3  [展开 ▼]        │    │
│  │    ┌─ Details: 模型输出包含错误信息 xxx...            │    │
│  │    ├─ 关联 Cases: case-1, case-3                     │    │
│  │    └─ [🔍 查看 LLM Call 详情 →]                      │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ ❌ Check 3: ...                                      │    │
│  └─────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│  Case 执行矩阵                                               │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐  │
│  │ Case     │ Status   │Expected  │ Actual   │ Action   │  │
│  ├──────────┼──────────┼──────────┼──────────┼──────────┤  │
│  │ case-1   │ ✅ pass  │ ...      │ ...      │ [查看 →]  │  │
│  │ case-2   │ ❌ fail  │ ...      │ ...      │ [查看 →]  │  │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**关键改进**：
- Rubric details 改为**可展开的诊断卡片**，失败项默认展开
- 每个 rubric check 增加"关联 Cases"和"查看 LLM Call"下钻入口
- 新增 **Case 执行矩阵** 表格，一眼看出哪些 case 挂了
- 从 case 或 rubric check 可直接跳转到 CallLogView 的对应记录

### 2.4 EvalHistoryView — 历史趋势与版本对比

**解决场景**：长期趋势跟踪、A/B 对比

**布局**：
```
┌─────────────────────────────────────────────────────────────┐
│  [Prompt 信息卡片]                                           │
├─────────────────────────────────────────────────────────────┤
│  分数趋势图（Echarts）                                        │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Overall │──╱╲──╱╲───  [切换: Overall/Det/Rubric]   │    │
│  └─────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│  Runs 列表（可勾选对比）                                       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ ☑ │ Run ID │ Time    │ Model   │ Overall │ Det │ Rub │ ▶ │
│  ├───┼────────┼─────────┼─────────┼─────────┼─────┼─────┼───┤
│  │ ☑ │ abc123 │ 04-27   │ gpt-4o  │ 0.82    │0.90 │0.74 │报告│
│  │ ☑ │ def456 │ 04-26   │ gpt-4o  │ 0.75    │0.85 │0.65 │报告│
│  └───┴────────┴─────────┴─────────┴─────────┴─────┴─────┴───┘
│  [勾选 2 项后显示: 🔍 Compare Selected Runs]                  │
└─────────────────────────────────────────────────────────────┘
```

**关键改进**：
- Runs 列表增加**复选框**，勾选 2 项后弹出/显示对比入口
- 对比结果页（或 Drawer）展示 rubric 级别的逐条变化（哪条 check 改善了、哪条 regress 了）
- 趋势图支持切换 Overall / Deterministic / Rubric 三条线

---

## 三、跨视图全局优化

### 3.1 全局 Eval 执行状态指示器

**问题**：
1. 用户启动 eval 后如果切到 Editor 或其他页面，就不知道跑完了没有。
2. **同时优化多个提示词时，当前单 eval 状态会互相覆盖。** 当前 `store.runningEval` 是单例对象（`{ id, assetId, assetName } | null`），无法表达并行执行。

**方案**：

#### Store 层改造

将单 eval 状态升级为**多 eval 集合**：

```ts
interface RunningEval {
  id: string
  assetId: string
  assetName: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelling'
  progress?: { completed: number; total: number }
  startedAt: number
}

// 替换原有的 runningEval + setRunningEval
runningEvals: RunningEval[]
setRunningEvals: (evals: RunningEval[]) => void
addRunningEval: (eval: RunningEval) => void
updateRunningEval: (id: string, patch: Partial<RunningEval>) => void
removeRunningEval: (id: string) => void
```

#### 轮询机制上移

当前 `EvalPanelView` 的轮询是页面级的（离开页面就停止）。要支持全局多 eval 监控，**轮询必须提升到全局层**：
- App 或 Sidebar 层统一轮询所有 `runningEvals` 中状态为 `pending/running` 的执行
- 各视图（EvalRunView、Sidebar、全局指示器）只从 store 订阅，不做独立轮询

#### UI 展示

顶部 Header 增加常驻状态区：

```
┌─────────────────────────────────────────────────────────────┐
│ 🟢 Server │ Assets ▼ │ Compare │ Settings │  🟡 3 running ▼ │
└─────────────────────────────────────────────────────────────┘

点击下拉后:
┌─────────────────────────────┐
│ 🟡 prompt-A    ████░░ 67%   │
│ 🟡 prompt-B    ██░░░░ 33%   │
│ 🟢 prompt-C    ✅ 完成       │
├─────────────────────────────┤
│ [查看全部执行 →]             │
└─────────────────────────────┘
```

- 单执行时：直接显示进度条 + assetName
- 多执行时：显示聚合摘要（如"3 running, 1 completed"），点击下拉展开列表
- 执行完成：对应项绿色脉冲提示，3 秒后自动移除或归档到"最近完成"
- 点击任意项：跳转对应 asset 的 `/eval/run` 页面

### 3.2 Editor → Eval 的快速通道

**问题**：Prompt 工程师的核心循环是"改 prompt → 跑 eval → 看结果 → 再改"。当前需要在 Editor 和 Eval 页面间反复跳转。

**方案**：
1. 在 **EditorViewV2** 的右侧 Chat Panel 下方（或 header 区域）增加 **Quick Eval 区域**：
   - 下拉选择 Preset（默认用上次配置）
   - `[▶ Quick Eval]` 按钮
   - 执行中在 Editor 页面内弹出 mini drawer/overlay 显示进度和结果快照
   - 结果快照包含 Overall Score + Pass/Fail 计数，点击跳转完整报告

2. 或者在 **EditorViewV2** header 的导航按钮中，把"Run Eval"升级为：
   - 点击后不是直接跳转，而是弹出一个快捷执行浮层（选择 model/mode）
   - 执行完成后浮层展示结果，用户可选择"跳转详细报告"或"继续编辑"

### 3.3 AssetListView 的 Eval 状态增强

**问题**：列表上看不出 prompt 的健康度。

**方案**：
- 卡片上的 `latest_score` 改为**迷你趋势图**（Sparkline，最近 5 次 eval 的折线，5px 高）
- 增加**距离上次 eval 的时间**（如"2h ago"），超过 24h 未跑 eval 显示灰色警告
- 增加快捷操作：卡片 hover 时显示 `[▶ Quick Eval]` 按钮

---

## 四、路由与组件调整

### 新增/修改路由

```tsx
// App.tsx 中 Eval 相关路由调整
<Route path="/assets/:id/eval" element={<EvalPanelView />}>  {/* 重定向到 /run */}
<Route path="/assets/:id/eval/design" element={<EvalDesignView />} />
<Route path="/assets/:id/eval/run" element={<EvalRunView />} />
<Route path="/assets/:id/eval/report/:runId" element={<EvalReportView />} />
<Route path="/assets/:id/eval/history" element={<EvalHistoryView />} />
```

### 组件拆分

```
web/src/views/eval/
├── EvalLayout.tsx          # 共享的左侧/顶部阶段导航 + Prompt 信息卡
├── EvalDesignView.tsx      # 原 EvalCasesView 升级 + 可编辑
├── EvalRunView.tsx         # 配置 + 执行 + 实时状态
├── EvalReportView.tsx      # 单轮深度报告（新增）
├── EvalHistoryView.tsx     # 趋势图 + Runs 列表 + 对比
└── components/
    ├── PromptHeader.tsx      # 共享的 Prompt 信息卡片
    ├── ScoreCards.tsx        # 四个分数 Statistic 卡片
    ├── RubricDetailList.tsx  # 可展开的 rubric 诊断列表
    ├── CaseMatrix.tsx        # Case 执行矩阵
    ├── RunConfigForm.tsx     # 执行参数配置表单
    ├── ExecutionProgress.tsx # 实时进度条
    ├── TrendChart.tsx        # Echarts 趋势图
    ├── RunCompareDrawer.tsx  # 对比抽屉
    └── QuickEvalModal.tsx    # Editor 里的快捷 eval 弹层
```

### 现有文件处理

| 现有文件 | 处理方式 |
|---------|---------|
| `EvalPanelView.tsx` | 拆分为 `EvalRunView` + `EvalHistoryView` + `EvalReportView`，原文件删除 |
| `EvalCasesView.tsx` | 重命名为 `EvalDesignView.tsx`，增加编辑功能 |
| `ContentDetailView.tsx` | `eval_history` tab 简化，去掉 Timeline 详情（下钻到 EvalHistoryView） |
| `CompareView.tsx` | 保留全局对比入口，但增加从 EvalHistoryView 的快捷对比 |

---

## 五、后端接口需求（待确认）

| 需求 | 是否已有接口 | 备注 |
|------|------------|------|
| 单 case 执行 | ❓ 需确认 | `evalApi.execute` 支持 `case_ids` 参数，似乎已支持 |
| Test case 编辑保存 | ❓ 需确认 | 当前 `assetApi.update` 是否支持更新 `test_cases`？ |
| 实时执行日志流 | ❓ 需确认 | 当前是轮询 `executionApi.get`，是否支持 SSE？ |
| Rubric check → case 关联 | ❓ 需确认 | Report 中是否有 case 级 rubric 结果？ |
| Preset 保存 | ❌ 可能需新增 | 前端 localStorage 先实现，后端可后补 |

---

## 六、后端执行模型限制（代码审查发现）

> 以下结论来自对 `internal/service/eval_service.go` 等后端代码的审查。

| 限制 | 现状 | 影响 |
|------|------|------|
| **执行是同步阻塞的** | `RunEval` 在 HTTP handler 内 `wg.Wait()` 直到所有 case 跑完，不会立即返回 | `POST /evals/execute` 请求会挂住数秒到数分钟，前端需要处理超长请求超时 |
| **Cancel 功能损坏** | `coordinators` sync.Map 从未被填充，`CancelExecution` 永远找不到执行体 | EvalRunView 的 Cancel 按钮无法工作 |
| **GetExecution 不可靠** | 优先查空的 `coordinators` map，fallback 才查 store | 前端轮询可能拿到过期的 `running` 状态，或状态更新延迟 |
| **无全局/单 asset 并发限制** | 没有互斥锁，同一个 asset 可以同时启动任意多个 eval | 前端"同时优化多个提示词"在接口层面可行，但 HTTP 长连接会堆积 |

### 对前端设计的约束

1. **全局多 eval 状态（`runningEvals` 数组）可以纯前端实现**，不需要后端改接口
2. **但每个 eval 的 HTTP 连接都是长连接**，同时跑 5 个以上可能触发浏览器并发连接限制（HTTP/1.1 下通常 6 个/域名）
3. **Cancel 按钮需要后端修复后才能工作**，前端可以先隐藏或做"请求已发送"的乐观提示
4. **执行状态轮询建议增加超时保护**：如果某个 execution 长时间处于 `running` 但后端查询返回异常，前端应标记为"状态未知"而不是无限等待

---

## 七、实施优先级建议

| 阶段 | 内容 | 预估工作量 |
|------|------|-----------|
| **Phase 1** | EvalRunView（配置+执行+实时状态）+ 全局状态指示器 | 中 |
| **Phase 2** | EvalReportView（深度报告+Case 矩阵+Rubric 诊断） | 中 |
| **Phase 3** | EvalHistoryView（趋势+Runs 列表+对比） | 小 |
| **Phase 4** | EvalDesignView（Cases 可编辑+单 case 运行） | 中（依赖后端接口） |
| **Phase 5** | Editor 内 Quick Eval + AssetList 增强 | 小 |

建议先实施 **Phase 1 + Phase 2**，这两个阶段解决的是当前最痛的"执行和结果分析"问题。
