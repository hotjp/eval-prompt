# task_014

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_014.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

P04 AssetIndexer 接口 — Reconcile/Search 接口定义

## 需求 (requirements)

- service/interfaces.go: AssetIndexer 接口定义
- Reconcile: 对账、索引重建
- Search: 全文搜索、过滤
- GetByID/Save/Delete

## 验收标准 (acceptance)

- [ ] AssetIndexer 接口定义完整
- [ ] 接口方法签名正确

## 交付物 (deliverables)

- `internal/service/interfaces.go` (补充 AssetIndexer)

## 验证证据（完成前必填）

- [ ] **实现证明**: 接口设计
- [ ] **测试验证**: 后续插件实现时验证
- [ ] **影响范围**: SyncService、TriggerService

### 测试步骤
1. 接口编译检查
