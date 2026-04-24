package commands

import (
	"context"
	"fmt"
	"os"

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

var syncReconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "对账",
	RunE: func(cmd *cobra.Command, args []string) error {
		indexer := search.NewIndexer()
		gitBridge := gitbridge.NewBridge()

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

var syncExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出备份",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		indexer := search.NewIndexer()
		gitBridge := gitbridge.NewBridge()

		syncService := service.NewSyncService(indexer, gitBridge)
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
}
