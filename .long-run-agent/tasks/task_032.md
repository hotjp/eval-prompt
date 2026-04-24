# task_032

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_032.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C03 ep asset 子命令 — list/show/cat/create/edit/rm

## 需求 (requirements)

- cmd/ep/commands/asset_list.go
- cmd/ep/commands/asset_show.go
- cmd/ep/commands/asset_cat.go
- cmd/ep/commands/asset_create.go
- cmd/ep/commands/asset_edit.go
- cmd/ep/commands/asset_rm.go

## 验收标准 (acceptance)

- [ ] ep asset list 可用
- [ ] ep asset show 可用
- [ ] ep asset cat 可用 (管道首选)
- [ ] ep asset create 可用
- [ ] ep asset edit 可用
- [ ] ep asset rm 可用

## 交付物 (deliverables)

- `cmd/ep/commands/asset_*.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: CLI 子命令
- [ ] **测试验证**: 各子命令测试
- [ ] **影响范围**: Asset 操作

### 测试步骤
1. `ep asset list` 测试
2. `ep asset cat <id> | head` 测试
