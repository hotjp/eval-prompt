# task_023

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_023.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

A03 AuditLogger — 操作审计日志

## 需求 (requirements)

- authz/audit_logger.go: AuditLogger 实现
- 记录 Asset 创建/修改/删除
- 记录 Label 移动
- 记录 Eval 触发
- 记录 Agent 身份

## 验收标准 (acceptance)

- [ ] AuditLogger 实现完整
- [ ] 操作审计记录正常
- [ ] Agent 身份记录

## 交付物 (deliverables)

- `internal/authz/audit_logger.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 审计日志逻辑
- [ ] **测试验证**: 审计记录测试
- [ ] **影响范围**: 合规审计

### 测试步骤
1. Asset 操作审计测试
2. Label 移动审计测试
