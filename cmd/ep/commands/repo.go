package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "管理多个仓库",
	Long:  `管理多个 prompt assets 仓库，支持 list 和 switch 操作`,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有仓库",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoLock, err := lock.ReadLock()
		if err != nil {
			return fmt.Errorf("读取 lock 文件失败: %w", err)
		}

		if len(repoLock.Repos) == 0 {
			fmt.Println("尚未初始化任何仓库，请先运行 'ep init <path>'")
			return nil
		}

		current := repoLock.GetCurrent()
		for _, entry := range repoLock.Repos {
			status := lock.ValidatePath(entry.Path)
			marker := "  "
			statusText := ""

			if entry.Path == current {
				marker = "● "
				switch status {
				case lock.PathValid:
					statusText = "← 当前"
				case lock.PathNotFound:
					statusText = "← 当前（未找到）"
				case lock.PathNotGit:
					statusText = "← 当前（无效）"
				}
			} else {
				switch status {
				case lock.PathValid:
					statusText = ""
				case lock.PathNotFound:
					statusText = "← 未找到"
				case lock.PathNotGit:
					statusText = "← 无效（不是git仓库）"
				}
			}

			if statusText != "" {
				fmt.Printf("%s%s %s\n", marker, entry.Path, statusText)
			} else {
				fmt.Printf("%s%s\n", marker, entry.Path)
			}
		}

		fmt.Printf("\n当前仓库: %s\n", current)
		return nil
	},
}

var repoSwitchCmd = &cobra.Command{
	Use:   "switch <path>",
	Short: "切换当前仓库",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("无法获取绝对路径: %w", err)
		}

		// Reject path traversal
		if strings.Contains(absPath, "..") {
			return fmt.Errorf("path traversal not allowed")
		}

		repoLock, err := lock.ReadLock()
		if err != nil {
			return fmt.Errorf("读取 lock 文件失败: %w", err)
		}

		// Check if path exists in repos
		found := false
		for _, entry := range repoLock.Repos {
			if entry.Path == absPath {
				found = true
				break
			}
		}

		if !found {
			// Path not in lock, ask to add it
			fmt.Printf("路径 %s 不在已管理的仓库中\n", absPath)
			fmt.Print("是否将其作为新仓库添加并切换? (y/N): ")
			var confirm string
			yes, _ := cmd.Flags().GetBool("yes")
			if yes {
				confirm = "y"
			} else {
				fmt.Scanln(&confirm)
			}
			if strings.ToLower(confirm) != "y" {
				fmt.Println("取消操作")
				return nil
			}
			repoLock.AddRepo(absPath)
		}

		// Validate path
		status := lock.ValidatePath(absPath)
		switch status {
		case lock.PathValid:
			// Good, switch to it
		case lock.PathNotFound:
			fmt.Printf("警告: 路径 %s 不存在\n", absPath)
			fmt.Print("是否创建并初始化? (y/N): ")
			var confirm string
			yes, _ := cmd.Flags().GetBool("yes")
			if yes {
				confirm = "y"
			} else {
				fmt.Scanln(&confirm)
			}
			if strings.ToLower(confirm) != "y" {
				fmt.Println("取消操作")
				return nil
			}
			// Create directory structure
			dirs := []string{"prompts", ".evals", ".traces"}
			for _, dir := range dirs {
				fullPath := filepath.Join(absPath, dir)
				if err := os.MkdirAll(fullPath, 0755); err != nil {
					return fmt.Errorf("创建目录失败: %w", err)
				}
			}
			// Init git
			bridge := gitbridge.NewBridge()
			if err := bridge.InitRepo(context.Background(), absPath); err != nil {
				fmt.Printf("警告: git init 失败: %v\n", err)
				gitignore := `.eval-prompt/
*.db
.traces/
*.jsonl
`
				os.WriteFile(filepath.Join(absPath, ".gitignore"), []byte(gitignore), 0644)
			}
		case lock.PathNotGit:
			fmt.Printf("警告: 路径 %s 不是 git 仓库\n", absPath)
			fmt.Print("是否初始化为 git 仓库? (y/N): ")
			var confirm string
			yes, _ := cmd.Flags().GetBool("yes")
			if yes {
				confirm = "y"
			} else {
				fmt.Scanln(&confirm)
			}
			if strings.ToLower(confirm) != "y" {
				fmt.Println("取消操作")
				return nil
			}
			bridge := gitbridge.NewBridge()
			if err := bridge.InitRepo(context.Background(), absPath); err != nil {
				return fmt.Errorf("git init 失败: %w", err)
			}
		}

		// Update lock
		repoLock.SetCurrent(absPath)
		if err := lock.WriteLock(repoLock); err != nil {
			return fmt.Errorf("写入 lock 文件失败: %w", err)
		}

		fmt.Printf("✅ 已切换到仓库: %s\n", absPath)

		// Note: config.PromptAssets.RepoPath update requires server restart or config reload
		// For CLI, we just update the lock file. The serve command will read from config.

		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoSwitchCmd)
	repoSwitchCmd.Flags().BoolP("yes", "y", false, "自动确认")
	rootCmd.AddCommand(repoCmd)
}
