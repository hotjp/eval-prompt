# task_043

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_043.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

W04 版本树视图 — Snapshot 时间轴、Diff 展示

## 需求 (requirements)

- web/src/pages/VersionTree.tsx
- 垂直时间轴展示 Snapshot 历史
- 节点显示: 版本号、提交信息、Eval 得分
- 点击节点查看内容
- 点击连线查看 Diff

## 验收标准 (acceptance)

- [ ] 版本树渲染正常
- [ ] 时间轴展示正确
- [ ] Diff 展示功能正常

## 交付物 (deliverables)

- `web/src/pages/VersionTree.tsx`

## 验证证据（完成前必填）

- [ ] **实现证明**: 时间轴组件
- [ ] **测试验证**: 版本历史测试
- [ ] **影响范围**: 版本管理视图

### 测试步骤
1. 版本树渲染测试
2. Diff 查看测试
