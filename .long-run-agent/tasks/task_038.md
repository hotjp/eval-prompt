# task_038

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_038.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C09 ep adapt 子命令 — 跨模型 Prompt 适配

## 需求 (requirements)

- cmd/ep/commands/adapt.go
- ep adapt <id> <version> [--from] [--to] [--save-as] [--auto-eval]

## 验收标准 (acceptance)

- [ ] ep adapt 命令可用
- [ ] --from/--to 选项可用
- [ ] --save-as 选项可用
- [ ] --auto-eval 选项可用

## 交付物 (deliverables)

- `cmd/ep/commands/adapt.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 跨模型适配
- [ ] **测试验证**: adapt 测试
- [ ] **影响范围**: ModelAdapter

### 测试步骤
1. `ep adapt common/code-review v1 --from claude-3-5-sonnet --to gpt-4o` 测试
