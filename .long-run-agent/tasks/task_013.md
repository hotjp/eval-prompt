# task_013

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_013.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

P03 EvalRunner 接口 + 实现 — Deterministic Checker + Rubric Grader

## 需求 (requirements)

- service/interfaces.go: EvalRunner 接口定义
- plugins/eval/runner.go: EvalRunner 实现
- Deterministic Checker: JSONL Trace 断言规则
- Rubric Grader: LLM 评审
- 内置断言库: command_executed, file_exists, json_valid, content_contains, json_path

## 验收标准 (acceptance)

- [ ] EvalRunner 接口定义完整
- [ ] Deterministic Checker 实现
- [ ] Rubric Grader 实现
- [ ] 断言库完整

## 交付物 (deliverables)

- `plugins/eval/runner.go`
- `plugins/eval/deterministic_checker.go`
- `plugins/eval/rubric_grader.go`
- `plugins/eval/assertions.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 两种 Eval 方式
- [ ] **测试验证**: 断言执行测试
- [ ] **影响范围**: 核心 Eval 功能

### 测试步骤
1. Deterministic Eval 测试
2. Rubric Eval 测试
