package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

		// Create directory structure
		dirs := []string{
			"prompts",
			".evals",
			".traces",
		}
		for _, dir := range dirs {
			fullPath := filepath.Join(path, dir)
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
		}

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
