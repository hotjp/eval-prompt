# task_020

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_020.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

S04 TriggerService — 触发匹配、Anti-Pattern Guard、变量注入

## 需求 (requirements)

- service/trigger_service.go: TriggerService 实现
- MatchTrigger: 输入匹配 Prompt
- ValidateAntiPatterns: Anti-Pattern 检查
- InjectVariables: 变量注入

## 验收标准 (acceptance)

- [ ] TriggerService 完整实现
- [ ] 触发匹配正常
- [ ] Anti-Pattern 检查
- [ ] 变量注入正常

## 交付物 (deliverables)

- `internal/service/trigger_service.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 触发匹配逻辑
- [ ] **测试验证**: Match 测试
- [ ] **影响范围**: MCP 协议

### 测试步骤
1. Match 触发测试
2. Anti-Pattern 验证
