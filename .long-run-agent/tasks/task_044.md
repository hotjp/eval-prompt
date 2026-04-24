# task_044

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_044.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

W05 Eval 面板 — 得分仪表盘、测试用例、Trace 时间轴

## 需求 (requirements)

- web/src/pages/EvalPanel.tsx
- 总体得分仪表盘
- 测试用例列表
- Trace 时间轴
- Rubric 检查项明细
- 重新执行 Eval 按钮

## 验收标准 (acceptance)

- [ ] 得分仪表盘渲染正常
- [ ] 测试用例列表正常
- [ ] Trace 时间轴展示正常
- [ ] Rubric 明细展示正常

## 交付物 (deliverables)

- `web/src/pages/EvalPanel.tsx`
- `web/src/components/ScoreGauge.tsx`

## 验证证据（完成前必填）

- [ ] **实现证明**: Eval 面板组件
- [ ] **测试验证**: Eval 结果展示测试
- [ ] **影响范围**: Eval 核心视图

### 测试步骤
1. Eval 面板渲染测试
2. 得分仪表盘测试
