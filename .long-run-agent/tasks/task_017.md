# task_017

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_017.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

S01 AssetService — Asset CRUD、版本管理、Label 操作

## 需求 (requirements)

- service/asset_service.go: AssetService 实现
- CreateAsset: 创建 Asset + 初始 Snapshot
- UpdateAsset: 更新 + 新 Snapshot
- GetAsset: 获取详情
- ListAssets: 列表 + 过滤
- SetLabel/UnsetLabel: Label 操作

## 验收标准 (acceptance)

- [ ] AssetService 完整实现
- [ ] CRUD + 版本管理
- [ ] Label 操作正常

## 交付物 (deliverables)

- `internal/service/asset_service.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 业务逻辑
- [ ] **测试验证**: CRUD + Label 测试
- [ ] **影响范围**: 核心服务层

### 测试步骤
1. Asset CRUD 测试
2. Label 操作测试
