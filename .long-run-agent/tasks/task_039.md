# task_039

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_039.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C10 ep optimize 子命令 — Agent 自主优化

## 需求 (requirements)

- cmd/ep/commands/optimize.go
- ep optimize <id> [--strategy] [--iterations] [--threshold-delta] [--auto-promote]
- 策略: failure_driven / score_max / compact

## 验收标准 (acceptance)

- [ ] ep optimize 命令可用
- [ ] --strategy 选项可用
- [ ] --iterations 选项可用
- [ ] --auto-promote 选项可用

## 交付物 (deliverables)

- `cmd/ep/commands/optimize.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 自主优化逻辑
- [ ] **测试验证**: optimize 测试
- [ ] **影响范围**: Auto-Optimization

### 测试步骤
1. `ep optimize common/code-review --strategy failure_driven` 测试
