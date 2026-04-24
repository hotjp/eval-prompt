# task_011

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_011.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

P01 GitBridger 接口 + 实现 — go-git: Init/Add/Commit/Diff/Log/Status

## 需求 (requirements)

- service/interfaces.go: GitBridger 接口定义
- plugins/gitbridge/bridge.go: go-git 实现
- 支持: Init, StageAndCommit, Diff, Log, Status
- .gitignore 自动管理

## 验收标准 (acceptance)

- [ ] GitBridger 接口定义完整
- [ ] go-git 实现可用
- [ ] Init/Commit/Diff/Log/Status 工作正常

## 交付物 (deliverables)

- `internal/service/interfaces.go`
- `plugins/gitbridge/bridge.go`
- `plugins/gitbridge/gitignore.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: go-git 纯 Go 实现
- [ ] **测试验证**: Git 操作测试
- [ ] **影响范围**: Asset 版本管理

### 测试步骤
1. `ep init` 测试 Git 仓库创建
2. Commit/Diff 操作测试
