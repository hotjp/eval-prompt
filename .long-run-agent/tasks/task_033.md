# task_033

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_033.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C04 ep snapshot 子命令 — list/diff/checkout

## 需求 (requirements)

- cmd/ep/commands/snapshot_list.go
- cmd/ep/commands/snapshot_diff.go
- cmd/ep/commands/snapshot_checkout.go

## 验收标准 (acceptance)

- [ ] ep snapshot list 可用
- [ ] ep snapshot diff 可用
- [ ] ep snapshot checkout 可用

## 交付物 (deliverables)

- `cmd/ep/commands/snapshot_*.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: CLI 子命令
- [ ] **测试验证**: 各子命令测试
- [ ] **影响范围**: 版本管理

### 测试步骤
1. `ep snapshot list <id>` 测试
2. `ep snapshot diff <id> v1 v2` 测试
