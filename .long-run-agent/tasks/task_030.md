# task_030

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_030.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C01 ep init — 仓库初始化、SQLite 创建、Git init

## 需求 (requirements)

- cmd/ep/commands/init.go: ep init 命令
- 创建 Git 仓库
- 创建 SQLite 数据库
- 创建目录结构 (prompts/, .evals/, .traces/)
- .gitignore 配置

## 验收标准 (acceptance)

- [ ] ep init 命令可用
- [ ] Git 仓库创建成功
- [ ] SQLite 数据库创建成功
- [ ] 目录结构正确

## 交付物 (deliverables)

- `cmd/ep/commands/init.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 初始化逻辑
- [ ] **测试验证**: ep init 测试
- [ ] **影响范围**: CLI 入口

### 测试步骤
1. `ep init ~/test-repo`
2. 验证 Git/SQLite/目录创建
