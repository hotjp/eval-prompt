# task_022

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_022.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

A02 SandboxGuard — 文件系统路径校验、防路径遍历

## 需求 (requirements)

- authz/sandbox_guard.go: SandboxGuard 中间件
- filepath.Clean 清理 ..
- 路径必须在沙箱根目录内
- 参考 docs/DESIGN.md Section 5.2

## 验收标准 (acceptance)

- [ ] SandboxGuard 实现完整
- [ ] 路径遍历防护
- [ ] 沙箱目录校验

## 交付物 (deliverables)

- `internal/authz/sandbox_guard.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 路径校验逻辑
- [ ] **测试验证**: 路径遍历测试
- [ ] **影响范围**: 文件操作安全

### 测试步骤
1. 正常路径放行
2. .. 路径遍历拦截
3. 沙箱外路径拦截
