# task_018

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_018.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

S02 EvalService — Eval 编排、A/B 矩阵、报告生成

## 需求 (requirements)

- service/eval_service.go: EvalService 实现
- RunEval: 执行 Eval
- CompareEval: A/B 比对
- GenerateReport: 报告生成
- DiagnoseEval: 失败归因

## 验收标准 (acceptance)

- [ ] EvalService 完整实现
- [ ] Eval 执行正常
- [ ] A/B 比对功能
- [ ] 报告生成正常

## 交付物 (deliverables)

- `internal/service/eval_service.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: Eval 编排逻辑
- [ ] **测试验证**: Eval 执行测试
- [ ] **影响范围**: 核心服务层

### 测试步骤
1. Eval 执行测试
2. A/B 比对测试
