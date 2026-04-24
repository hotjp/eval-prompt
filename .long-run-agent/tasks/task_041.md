# task_041

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_041.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

W02 资产库视图 — 左侧分类树、右侧资产卡片网格、搜索筛选

## 需求 (requirements)

- web/src/pages/AssetLibrary.tsx
- 左侧: 分类树 (biz_line)
- 右侧: 资产卡片网格
- 顶部: 搜索 + 筛选
- 卡片展示: 名称、版本、Label、Eval 得分

## 验收标准 (acceptance)

- [ ] 分类树渲染正常
- [ ] 资产卡片网格正常
- [ ] 搜索筛选功能正常

## 交付物 (deliverables)

- `web/src/pages/AssetLibrary.tsx`
- `web/src/components/AssetCard.tsx`

## 验证证据（完成前必填）

- [ ] **实现证明**: UI 组件
- [ ] **测试验证**: 页面渲染测试
- [ ] **影响范围**: 核心视图

### 测试步骤
1. 页面渲染测试
2. 搜索功能测试
