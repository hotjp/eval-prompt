package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <path>",
	Short: "初始化仓库 + SQLite",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		fmt.Printf("初始化 prompt assets 仓库: %s\n", path)

		// 创建目录结构
		dirs := []string{
			"prompts",
			".evals",
			".traces",
		}
		for _, dir := range dirs {
			if err := os.MkdirAll(fmt.Sprintf("%s/%s", path, dir), 0755); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
		}

		// Git init
		if err := os.Chdir(path); err != nil {
			return fmt.Errorf("切换目录失败: %w", err)
		}
		gitCmd := exec.Command("git", "init")
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			fmt.Printf("警告: git init 失败: %v\n", err)
		}

		// 创建 .gitignore
		gitignore := `.eval-prompt/
*.db
.traces/
*.jsonl
`
		if err := os.WriteFile(fmt.Sprintf("%s/.gitignore", path), []byte(gitignore), 0644); err != nil {
			return fmt.Errorf("写入 .gitignore 失败: %w", err)
		}

		// 创建 SQLite 数据库
		dbPath := fmt.Sprintf("%s/.eval-prompt/index.db", os.Getenv("HOME"))
		if err := os.MkdirAll(fmt.Sprintf("%s/.eval-prompt", os.Getenv("HOME")), 0755); err != nil {
			return fmt.Errorf("创建数据库目录失败: %w", err)
		}
		f, err := os.Create(dbPath)
		if err != nil {
			return fmt.Errorf("创建数据库失败: %w", err)
		}
		f.Close()

		fmt.Printf("\n✅ 初始化完成!\n")
		fmt.Printf("   Git 仓库: %s/.git\n", path)
		fmt.Printf("   SQLite: %s\n", dbPath)
		fmt.Printf("\n运行 'ep serve' 启动服务\n")
		return nil
	},
}
