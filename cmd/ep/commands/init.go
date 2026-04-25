package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <path>",
	Short: "初始化仓库 + SQLite",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		fmt.Printf("初始化 prompt assets 仓库: %s\n", path)

			// Git init using go-git bridge
		bridge := gitbridge.NewBridge()
		if err := bridge.InitRepo(context.Background(), path); err != nil {
			fmt.Printf("警告: git init 失败: %v\n", err)
			// Fall back to manual gitignore
			gitignore := `.eval-prompt/
*.db
.traces/
*.jsonl
`
			os.WriteFile(filepath.Join(path, ".gitignore"), []byte(gitignore), 0644)
		} else {
			fmt.Printf("✅ Git 仓库初始化完成\n")
		}

		// Update repo lock file
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("警告: 无法获取绝对路径: %v\n", err)
		} else {
			repoLock, err := lock.ReadLock()
			if err != nil {
				fmt.Printf("警告: 读取 lock 文件失败: %v\n", err)
			} else {
				repoLock.AddRepo(absPath)
				repoLock.SetCurrent(absPath)
				if err := lock.WriteLock(repoLock); err != nil {
					fmt.Printf("警告: 写入 lock 文件失败: %v\n", err)
				} else {
					fmt.Printf("✅ 仓库已添加到锁文件\n")
				}
			}
		}

		// Create SQLite database path
		dbDir := filepath.Join(os.Getenv("HOME"), ".eval-prompt")
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("创建数据库目录失败: %w", err)
		}
		dbPath := filepath.Join(dbDir, "index.db")

		fmt.Printf("\n初始化完成!\n")
		fmt.Printf("   Git 仓库: %s/.git\n", path)
		fmt.Printf("   SQLite: %s\n", dbPath)
		fmt.Printf("\n运行 'ep serve' 启动服务\n")
		return nil
	},
}
