# task_053

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_053.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

B02 Homebrew 分发 — brew tap 配置

## 需求 (requirements)

- homebrew-ep Formula
- brew install eval-prompt/tap/ep
- 参考 docs/DESIGN.md Section 12.2

## 验收标准 (acceptance)

- [ ] Homebrew Formula 可用
- [ ] brew install 成功
- [ ] ep 命令可用

## 交付物 (deliverables)

- `homebrew-ep.rb` (或 homebrew/ directory)

## 验证证据（完成前必填）

- [ ] **实现证明**: Homebrew Formula
- [ ] **测试验证**: brew install 测试
- [ ] **影响范围**: 分发

### 测试步骤
1. Homebrew Formula 验证
2. brew install 测试 (可选)
