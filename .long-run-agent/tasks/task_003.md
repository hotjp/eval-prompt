# task_003

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_003.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

F03 共享内核 — domain/types.go (ULID)、errors.go (错误码)、events.go (领域事件协议)、state_machine.go

## 需求 (requirements)

- domain/types.go: ULID 生成器、通用类型定义
- domain/errors.go: 错误码格式 L{层号}{3位序号}，DomainError 结构
- domain/events.go: 领域事件协议（参考 docs/DESIGN.md Section 三 Domain 领域事件）
- domain/state_machine.go: 通用状态机框架

## 验收标准 (acceptance)

- [ ] ULID 生成器可用
- [ ] 错误码格式符合规范
- [ ] 领域事件结构完整
- [ ] 状态机框架可用

## 交付物 (deliverables)

- `internal/domain/types.go`
- `internal/domain/errors.go`
- `internal/domain/events.go`
- `internal/domain/state_machine.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 各模块实现说明
- [ ] **测试验证**: 单元测试通过
- [ ] **影响范围**: L2-Domain 层核心依赖

### 测试步骤
1. ULID 唯一性验证
2. 错误码格式验证
3. 状态机转换测试
