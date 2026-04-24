# task_009

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_009.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

D05 OutboxEvent + AuditLog Storage — 事务内写入、轮询转发

## 需求 (requirements)

- domain/outbox_event.go: OutboxEvent 实体
- domain/audit_log.go: AuditLog 实体
- storage/ent/schema/outbox_event.go
- storage/ent/schema/audit_log.go
- storage/outbox_repository.go
- storage/audit_log_repository.go

## 验收标准 (acceptance)

- [ ] OutboxEvent 实体完整
- [ ] AuditLog 实体完整
- [ ] 事务内写入可用

## 交付物 (deliverables)

- `internal/domain/outbox_event.go`
- `internal/domain/audit_log.go`
- `internal/storage/ent/schema/outbox_event.go`
- `internal/storage/ent/schema/audit_log.go`
- `internal/storage/outbox_repository.go`
- `internal/storage/audit_log_repository.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 事务一致性保证
- [ ] **测试验证**: 事务测试
- [ ] **影响范围**: 事件驱动架构

### 测试步骤
1. OutboxEvent 事务写入测试
2. AuditLog 记录测试
