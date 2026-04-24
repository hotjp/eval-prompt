# task_034

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_034.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C05 ep label 子命令 — list/set/unset

## 需求 (requirements)

- cmd/ep/commands/label_list.go
- cmd/ep/commands/label_set.go
- cmd/ep/commands/label_unset.go

## 验收标准 (acceptance)

- [ ] ep label list 可用
- [ ] ep label set 可用
- [ ] ep label unset 可用

## 交付物 (deliverables)

- `cmd/ep/commands/label_*.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: CLI 子命令
- [ ] **测试验证**: 各子命令测试
- [ ] **影响范围**: Label 操作

### 测试步骤
1. `ep label list <id>` 测试
2. `ep label set <id> prod v1.2.3` 测试
