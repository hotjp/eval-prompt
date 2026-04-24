# task_016

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_016.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

P06 ModelAdapter 接口 + 实现 — 跨模型格式转换、参数调整、规则库

## 需求 (requirements)

- service/interfaces.go: ModelAdapter 接口定义
- plugins/modeladapter/adapter.go: 实现
- Adapt: Prompt 跨模型转换
- RecommendParams: 参数建议
- EstimateScore: 得分预估
- GetModelProfile: 模型特性

## 验收标准 (acceptance)

- [ ] ModelAdapter 接口定义完整
- [ ] Claude ↔ GPT 转换实现
- [ ] 参数调整实现

## 交付物 (deliverables)

- `plugins/modeladapter/adapter.go`
- `plugins/modeladapter/rules.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 模型适配逻辑
- [ ] **测试验证**: 格式转换测试
- [ ] **影响范围**: ep adapt 命令

### 测试步骤
1. Adapt 调用测试
2. 参数调整验证
