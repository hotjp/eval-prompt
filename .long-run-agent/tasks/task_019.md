# task_019

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_019.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

S03 SyncService — Reconcile、索引重建、导出备份

## 需求 (requirements)

- service/sync_service.go: SyncService 实现
- Reconcile: Git 仓库与数据库对账
- RebuildIndex: 索引重建
- Export: 导出备份

## 验收标准 (acceptance)

- [ ] SyncService 完整实现
- [ ] Reconcile 功能正常
- [ ] 导出功能正常

## 交付物 (deliverables)

- `internal/service/sync_service.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 对账逻辑
- [ ] **测试验证**: Reconcile 测试
- [ ] **影响范围**: 数据一致性

### 测试步骤
1. Reconcile 测试
2. Export 测试
