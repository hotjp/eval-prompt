# 分支策略推荐

## 个人使用

- 使用 main 分支即可
- 本地改动直接 commit

## 团队协作

1. 每个人 fork 自己的分支
2. 改动在个人分支开发
3. eval 分数验证后发起 PR 到 main
4. review 后合并

## 跨设备同步

1. 使用 Git remote（如 GitHub）
2. 设备 A: `git push`
3. 设备 B: `git pull`
