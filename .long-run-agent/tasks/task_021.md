# task_021

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_021.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

A01 EvalGateGuard — prod 晋升门禁（Eval 得分 ≥ 80）

## 需求 (requirements)

- authz/eval_gate_guard.go: EvalGateGuard 实现
- 拦截 Label 移动到 prod 的请求
- 校验目标 Snapshot 的 Eval 得分 ≥ 阈值（默认 80）

## 验收标准 (acceptance)

- [ ] EvalGateGuard 实现完整
- [ ] 门禁逻辑正确
- [ ] 得分校验正常

## 交付物 (deliverables)

- `internal/authz/eval_gate_guard.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 门禁检查逻辑
- [ ] **测试验证**: 门禁测试
- [ ] **影响范围**: Label 晋升

### 测试步骤
1. 得分达标时晋升放行
2. 得分不达标时晋升拦截
