# task_046

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_046.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

H01 路径 Sanitize — 清理 ..、校验沙箱目录

## 需求 (requirements)

- internal/authz/sanitize.go
- filepath.Clean 清理 ..
- 检测 .. 路径遍历
- 校验路径在沙箱根目录内

## 验收标准 (acceptance)

- [ ] 路径 Sanitize 实现完整
- [ ] .. 遍历防护正常
- [ ] 沙箱边界校验正常

## 交付物 (deliverables)

- `internal/authz/sanitize.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 路径校验逻辑
- [ ] **测试验证**: 路径遍历测试
- [ ] **影响范围**: 文件系统安全

### 测试步骤
1. 正常路径放行
2. .. 路径拦截
3. 沙箱外路径拦截
