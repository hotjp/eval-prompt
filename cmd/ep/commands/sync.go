package commands

import "github.com/spf13/cobra"

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
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var syncExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出备份",
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}
