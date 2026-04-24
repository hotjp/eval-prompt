# task_037

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_037.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C08 ep sync 子命令 — reconcile/export

## 需求 (requirements)

- cmd/ep/commands/sync_reconcile.go
- cmd/ep/commands/sync_export.go

## 验收标准 (acceptance)

- [ ] ep sync reconcile 可用
- [ ] ep sync export 可用

## 交付物 (deliverables)

- `cmd/ep/commands/sync_*.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: CLI 子命令
- [ ] **测试验证**: sync 测试
- [ ] **影响范围**: 数据同步

### 测试步骤
1. `ep sync reconcile` 测试
2. `ep sync export` 测试
