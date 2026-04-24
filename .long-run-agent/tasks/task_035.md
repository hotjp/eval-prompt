# task_035

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_035.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C06 ep eval 子命令 — run/cases/compare/report/diagnose

## 需求 (requirements)

- cmd/ep/commands/eval_run.go
- cmd/ep/commands/eval_cases.go
- cmd/ep/commands/eval_compare.go
- cmd/ep/commands/eval_report.go
- cmd/ep/commands/eval_diagnose.go

## 验收标准 (acceptance)

- [ ] ep eval run 可用
- [ ] ep eval cases 可用
- [ ] ep eval compare 可用
- [ ] ep eval report 可用
- [ ] ep eval diagnose 可用

## 交付物 (deliverables)

- `cmd/ep/commands/eval_*.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: CLI 子命令
- [ ] **测试验证**: 各子命令测试
- [ ] **影响范围**: Eval 操作

### 测试步骤
1. `ep eval run <id>` 测试
2. `ep eval report <run-id>` 测试
