# task_001

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_001.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

F01 项目初始化 — go mod init、目录结构、ep CLI 根命令脚手架

## 需求 (requirements)

- go mod init eval-prompt
- 创建完整目录结构（参考 docs/DESIGN.md Section 13.1）：
  - cmd/server/
  - internal/{config,gateway,authz,service,domain,storage,telemetry}/
  - plugins/{gitbridge,llm,eval,mcp,modeladapter}/
  - api/
  - web/ (React 项目占位)
  - scripts/
  - migrations/
- 创建 ep CLI 根命令脚手架（cobra 或clicmd）
- 创建 Makefile 基础结构

## 验收标准 (acceptance)

- [ ] go mod init 成功
- [ ] 目录结构完整
- [ ] `ep --help` 可执行
- [ ] Makefile 存在基础 target

## 交付物 (deliverables)

- `go.mod`
- `cmd/server/main.go`
- `cmd/ep/main.go` (CLI 入口)
- `Makefile`
- 完整目录结构

## 设计方案 (design)

按照 docs/DESIGN.md 的技术栈：
- 连接协议: connect-go
- ORM: ent
- Git: go-git/v6
- 配置: koanf
- 日志: log/slog

## 验证证据（完成前必填）

<!-- 标记完成前，请提供以下证据： -->

- [ ] **实现证明**: 简要说明如何实现
- [ ] **测试验证**: 如何验证功能正常（测试步骤/截图/命令输出）
- [ ] **影响范围**: 是否影响其他功能

### 测试步骤
1. `go mod tidy` 无错误
2. `go build -o bin/ep ./cmd/ep/` 成功
3. `./bin/ep --help` 输出帮助

### 验证结果
<!-- 粘贴验证截图、命令输出或测试结果 -->
