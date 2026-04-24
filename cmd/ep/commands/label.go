package commands

import "github.com/spf13/cobra"

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "标签操作",
}

func init() {
	labelCmd.AddCommand(labelListCmd)
	labelCmd.AddCommand(labelSetCmd)
	labelCmd.AddCommand(labelUnsetCmd)
}

var labelListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "列出标签",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var labelSetCmd = &cobra.Command{
	Use:   "set <id> <name> <v>",
	Short: "设置标签",
	Args:  cobra.ExactArgs(3),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var labelUnsetCmd = &cobra.Command{
	Use:   "unset <id> <name>",
	Short: "取消标签",
	Args:  cobra.ExactArgs(2),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}
