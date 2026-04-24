package commands

import "github.com/spf13/cobra"

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "版本管理",
}

func init() {
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotDiffCmd)
	snapshotCmd.AddCommand(snapshotCheckoutCmd)
}

var snapshotListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "列出版本",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var snapshotDiffCmd = &cobra.Command{
	Use:   "diff <id> <v1> <v2>",
	Short: "版本对比",
	Args:  cobra.ExactArgs(3),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var snapshotCheckoutCmd = &cobra.Command{
	Use:   "checkout <id> <v>",
	Short: "切换版本",
	Args:  cobra.ExactArgs(2),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}
