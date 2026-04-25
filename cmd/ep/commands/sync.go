package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/eval-prompt/plugins/search"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "同步操作",
}

func init() {
	syncCmd.AddCommand(syncReconcileCmd)
	syncCmd.AddCommand(syncExportCmd)
}

// resolveWorkDir resolves the working directory from --repo or --dir flag.
// If --repo is specified, it switches to that repo in lock.json first.
// If --dir is specified, uses that directory directly without touching lock.
// If neither is specified, uses the current repo from lock.json.
// Returns an error if no repo is set and neither flag is provided.
func resolveWorkDir(cmd *cobra.Command) (string, error) {
	repoPath, _ := cmd.Flags().GetString("repo")
	dirPath, _ := cmd.Flags().GetString("dir")

	if repoPath != "" {
		// --repo specified: add to lock and switch to it
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return "", fmt.Errorf("无法获取绝对路径: %w", err)
		}

		repoLock, err := lock.ReadLock()
		if err != nil {
			return "", fmt.Errorf("读取 lock 文件失败: %w", err)
		}

		repoLock.AddRepo(absPath)
		repoLock.SetCurrent(absPath)
		if err := lock.WriteLock(repoLock); err != nil {
			return "", fmt.Errorf("写入 lock 文件失败: %w", err)
		}

		fmt.Printf("已切换到仓库: %s\n", absPath)
		return absPath, nil
	}

	if dirPath != "" {
		// --dir specified: use it directly
		return dirPath, nil
	}

	// Neither --repo nor --dir specified: require current repo from lock
	repoLock, err := lock.ReadLock()
	if err != nil {
		return "", fmt.Errorf("读取 lock 文件失败: %w", err)
	}

	current := repoLock.GetCurrent()
	if current == "" {
		return "", fmt.Errorf("未设置仓库，请先运行 ep init 或使用 --repo 指定")
	}
	return current, nil
}

var syncReconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "对账",
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := resolveWorkDir(cmd)
		if err != nil {
			return err
		}

		indexer := search.Default()
		indexer.SetPersistDir(filepath.Join(wd, ".eval-prompt"))
		if err := indexer.Load(); err != nil {
			fmt.Printf("警告: 加载索引失败: %v\n", err)
		}
		gitBridge := gitbridge.NewBridge()
		if err := gitBridge.Open(wd); err != nil {
			return fmt.Errorf("打开git仓库失败: %w", err)
		}
		indexer.SetGitBridge(gitBridge)

		syncService := service.NewSyncService(indexer, gitBridge)
		report, err := syncService.Reconcile(context.Background())
		if err != nil {
			return fmt.Errorf("对账失败: %w", err)
		}

		fmt.Printf("对账完成\n")
		fmt.Printf("新增: %d\n", report.Added)
		fmt.Printf("更新: %d\n", report.Updated)
		fmt.Printf("删除: %d\n", report.Deleted)
		if len(report.Errors) > 0 {
			fmt.Printf("错误: %v\n", report.Errors)
		}
		return nil
	},
}

func init() {
	syncReconcileCmd.Flags().String("repo", "", "仓库路径（默认为当前仓库）")
	syncReconcileCmd.Flags().String("dir", "", "项目目录路径（仅影响 --dir 模式，不写入 lock）")
}

var syncExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出备份",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		wd, err := resolveWorkDir(cmd)
		if err != nil {
			return err
		}

		indexer := search.Default()
		gitBridge := gitbridge.NewBridge()
		if err := gitBridge.Open(wd); err != nil {
			return fmt.Errorf("打开git仓库失败: %w", err)
		}
		indexer.SetGitBridge(gitBridge)

		syncService := service.NewSyncService(indexer, gitBridge)

		// Run reconcile first to populate the index from git
		if _, err := syncService.Reconcile(context.Background()); err != nil {
			return fmt.Errorf("对账失败: %w", err)
		}

		data, err := syncService.Export(context.Background(), format)
		if err != nil {
			return fmt.Errorf("导出失败: %w", err)
		}

		if output != "" {
			return os.WriteFile(output, data, 0644)
		}

		fmt.Print(string(data))
		return nil
	},
}

func init() {
	syncExportCmd.Flags().String("format", "json", "导出格式: json|yaml")
	syncExportCmd.Flags().String("output", "", "输出文件路径")
	syncExportCmd.Flags().String("repo", "", "仓库路径（默认为当前仓库）")
	syncExportCmd.Flags().String("dir", "", "项目目录路径（仅影响 --dir 模式，不写入 lock）")
}
