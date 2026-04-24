# LRA Instructions

> **LRA** (Long-Running Agent) 是本项目的任务管理框架。
> agent 必须通过本文件了解 LRA，不要假设其他知识。

## 核心概念

LRA 使用 **Ralph Loop** 7 阶段迭代：
`pending → in_progress → completed → optimizing → truly_completed`

质量门控：后期阶段才强制性能测试和 lint。

## 任务状态

| 状态 | 说明 |
|------|------|
| `pending` | 未开始，可能有依赖阻塞 |
| `in_progress` | 进行中 |
| `completed` | 初始完成，等待质量检查 |
| `optimizing` | 修复质量问题中 |
| `truly_completed` | 全部质量门控通过 |

## 优先级

| 优先级 | 说明 |
|--------|------|
| P0 | 关键（安全、数据丢失、破坏构建）|
| P1 | 高（主要功能、重要 bug）|
| P2 | 中（默认）|
| P3 | 低（优化）|

## 完整命令参考

```bash
# 查看任务
lra list                # 列出所有任务
lra ready              # 列出可认领任务
lra show <id>          # 查看任务详情
lra status             # 查看项目进度

# 认领和操作
lra claim <id>         # 原子性认领任务
lra new "描述"         # 快速创建并认领
lra set <id> <status>  # 更新状态

# 状态流转
lra set <id> in_progress     # 开始
lra set <id> completed       # 初始完成
lra set <id> optimizing      # 优化
lra set <id> truly_completed # 完成

# 检查点
lra checkpoint <id> --note "进度"  # 保存检查点

# 依赖管理
lra deps <id>         # 查看依赖
lra deps add <child> <parent>  # 添加依赖

# 其他
lra doctor             # 健康检查
lra constitution show  # 查看 Constitution 规则
```

## Constitution 质量门控

LRA 通过 Constitution 验证任务质量：

```bash
lra check <id>         # 运行质量检查
lra constitution show  # 查看规则
```

**规则类型**：
- `NON_NEGOTIABLE`: 无法绕过
- `MANDATORY`: 必须通过
- `CONFIGURABLE`: 可选

## 工作流程

```
1. lra ready              # 查看可认领任务
2. lra claim <id>        # 原子性认领
3. lra set <id> in_progress
4. 实现功能
5. lra checkpoint <id> --note "完成核心逻辑"
6. lra set <id> completed
7. lra check <id>        # 运行 Constitution 验证
8. 如果失败: lra set <id> optimizing → 修复 → 回到 step 6
9. 成功后: lra set <id> truly_completed
```

## Session 完成 Checklist

**结束 session 前必须**：

1. `lra checkpoint <id> --note "当前进度"` 保存所有进行中的任务
2. `lra set <id> completed/optimizing` 更新状态
3. Git 推送：
   ```bash
   git add .
   git commit -m "..."
   git push
   ```
4. 为下一个 agent 提供上下文（已完成什么、下一步是什么）

## 禁止规则

- ❌ 不要创建 markdown TODO 列表
- ❌ 不要使用 LRA 以外的追踪系统
- ❌ 不要跳过 `lra ready` 直接问"我该做什么"
- ❌ 不要编辑 task 文件（用 `lra set` 命令）

## 非交互命令

**始终使用非交互标志**，避免命令挂起：

```bash
cp -f source dest      # 不要用: cp source dest
rm -f file            # 不要用: rm file
rm -rf directory      # 不要用: rm -r directory
```

## 外部依赖

本项目的外部服务配置在 `.long-run-agent/config.json`。

<!-- BEGIN LRA INTEGRATION profile:full -->
<!-- 此 section 由 lra init 管理，请勿手动修改 -->
<!-- END LRA INTEGRATION -->
