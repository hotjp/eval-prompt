# task_036

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_036.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C07 ep trigger 子命令 — match

## 需求 (requirements)

- cmd/ep/commands/trigger_match.go
- ep trigger match <input> [--top N] [--json]

## 验收标准 (acceptance)

- [ ] ep trigger match 可用
- [ ] --top 选项可用
- [ ] --json 选项可用

## 交付物 (deliverables)

- `cmd/ep/commands/trigger_match.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: CLI 子命令
- [ ] **测试验证**: match 测试
- [ ] **影响范围**: Agent 触发

### 测试步骤
1. `ep trigger match "检查 Go 代码"` 测试
2. `ep trigger match "xxx" --top 3 --json` 测试
