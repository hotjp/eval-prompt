# task_008

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_008.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

D04 EvalCase + EvalRun 领域 + Storage — deterministic_score、rubric_score、Ent schema

## 需求 (requirements)

- domain/eval_case.go: EvalCase 实体
- domain/eval_run.go: EvalRun 实体
- storage/ent/schema/eval_case.go: Ent schema
- storage/ent/schema/eval_run.go: Ent schema
- storage/eval_case_repository.go
- storage/eval_run_repository.go

## 验收标准 (acceptance)

- [ ] EvalCase/EvalRun 实体完整
- [ ] Ent schema 生成成功
- [ ] Repository 可用

## 交付物 (deliverables)

- `internal/domain/eval_case.go`
- `internal/domain/eval_run.go`
- `internal/storage/ent/schema/eval_case.go`
- `internal/storage/ent/schema/eval_run.go`
- `internal/storage/eval_case_repository.go`
- `internal/storage/eval_run_repository.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: EvalRun 状态机
- [ ] **测试验证**: CRUD 测试
- [ ] **影响范围**: Eval 引擎

### 测试步骤
1. EvalCase.Create 测试
2. EvalRun 状态转换测试
