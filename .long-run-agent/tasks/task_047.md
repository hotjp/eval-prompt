# task_047

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_047.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

H02 命令白名单 — Eval 执行命令校验

## 需求 (requirements)

- plugins/eval/command_validator.go
- 命令名白名单: npm, go, python, python3, node, git, curl, mkdir, cp, mv, rm
- 禁止模式检测: rm -rf /, | sh, | bash, curl .* | sh
- 执行前校验

## 验收标准 (acceptance)

- [ ] 命令白名单校验正常
- [ ] 禁止模式检测正常
- [ ] 校验失败正确拦截

## 交付物 (deliverables)

- `plugins/eval/command_validator.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 命令校验逻辑
- [ ] **测试验证**: 白名单/黑名单测试
- [ ] **影响范围**: Eval 安全

### 测试步骤
1. 白名单命令放行
2. 非白名单命令拦截
3. 禁止模式拦截
