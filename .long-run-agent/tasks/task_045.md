# task_045

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_045.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

W06 A/B 比对视图 — 内容 Diff、雷达图比对

## 需求 (requirements)

- web/src/pages/ABCompare.tsx
- 左右分栏展示两个版本
- 内容 Diff 展示
- Eval 雷达图比对
- Trace 路径差异

## 验收标准 (acceptance)

- [ ] A/B 分栏渲染正常
- [ ] 内容 Diff 展示正常
- [ ] 雷达图比对展示正常

## 交付物 (deliverables)

- `web/src/pages/ABCompare.tsx`
- `web/src/components/RadarChart.tsx`

## 验证证据（完成前必填）

- [ ] **实现证明**: A/B 比对组件
- [ ] **测试验证**: 比对功能测试
- [ ] **影响范围**: 版本对比视图

### 测试步骤
1. A/B 比对渲染测试
2. 雷达图展示测试
