package commands

import "github.com/spf13/cobra"

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "资产操作",
}

func init() {
	assetCmd.AddCommand(assetListCmd)
	assetCmd.AddCommand(assetShowCmd)
	assetCmd.AddCommand(assetCatCmd)
	assetCmd.AddCommand(assetCreateCmd)
	assetCmd.AddCommand(assetEditCmd)
	assetCmd.AddCommand(assetRmCmd)
}

var assetListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有资产",
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var assetShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "显示资产详情",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var assetCatCmd = &cobra.Command{
	Use:   "cat <id>",
	Short: "纯文本输出（管道首选）",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var assetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建资产",
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var assetEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "编辑资产",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var assetRmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "删除资产",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}
