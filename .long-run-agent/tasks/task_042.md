# task_042

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_042.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

W03 编辑器视图 — Monaco Editor + 实时预览 + 保存提交

## 需求 (requirements)

- web/src/pages/Editor.tsx
- Monaco Editor 编辑 PROMPT.md
- 左侧: 编辑区
- 右侧: 实时预览 (变量注入后)
- 底部: 保存并提交按钮

## 验收标准 (acceptance)

- [ ] Monaco Editor 正常
- [ ] YAML/Markdown 高亮
- [ ] 实时预览正常
- [ ] 保存提交功能正常

## 交付物 (deliverables)

- `web/src/pages/Editor.tsx`
- `web/src/components/PromptPreview.tsx`

## 验证证据（完成前必填）

- [ ] **实现证明**: Monaco 集成
- [ ] **测试验证**: 编辑功能测试
- [ ] **影响范围**: 核心视图

### 测试步骤
1. Monaco 编辑测试
2. 保存提交测试
