# task_052

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_052.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

B01 Makefile 构建 — npm build + go build + embed

## 需求 (requirements)

- Makefile
- make build: npm run build → 放入 web/dist/ → go build -o bin/ep
- make install: cp bin/ep /usr/local/bin/
- 参考 docs/DESIGN.md Section 12.1

## 验收标准 (acceptance)

- [ ] make build 成功
- [ ] ./bin/ep 可执行
- [ ] make install 成功

## 交付物 (deliverables)

- `Makefile`

## 验证证据（完成前必填）

- [ ] **实现证明**: 构建脚本
- [ ] **测试验证**: make build 测试
- [ ] **影响范围**: 构建流程

### 测试步骤
1. make build 测试
2. ./bin/ep --help 测试
