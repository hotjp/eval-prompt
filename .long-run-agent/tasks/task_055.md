# task_055

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_055.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

sync/Git E2E tests: init → create prompts/.md → sync reconcile → sync export (yaml/json) → modify .md → sync reconcile detect Updated

## 需求 (requirements)

- 创建 cmd/ep/commands/sync_e2e_test.go
- 测试流程: init → create prompts/.md → sync reconcile → sync export (yaml/json) → modify .md → sync reconcile detect Updated

## 验收标准 (acceptance)

- [ ] E2E test file created
- [ ] Test init creates repo and prompts directory
- [ ] Test reconcile detects added files
- [ ] Test export outputs valid YAML and JSON
- [ ] Test reconcile detects Updated after file modification

## 交付物 (deliverables)

- `cmd/ep/commands/sync_e2e_test.go`

## 设计方案 (design)

- 使用 os.MkdirTemp 创建临时测试目录
- 使用 exec.Command 调用 ep init, ep sync reconcile, ep sync export
- 使用 testify/assert 进行断言
- 清理测试目录 defer

## 验证证据（完成前必填）

- [ ] **实现证明**: Go E2E test using exec.Command
- [ ] **测试验证**: go test -v ./cmd/ep/commands/... -run Sync
- [ ] **影响范围**: None - test only file
