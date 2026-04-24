package commands

import "github.com/spf13/cobra"

var adaptCmd = &cobra.Command{
	Use:   "adapt <id> <version>",
	Short: "跨模型 Prompt 适配",
	Args:  cobra.ExactArgs(2),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

func init() {
	adaptCmd.Flags().String("from", "", "源模型")
	adaptCmd.Flags().String("to", "", "目标模型")
	adaptCmd.Flags().String("save-as", "", "保存为新 Asset")
	adaptCmd.Flags().Bool("auto-eval", false, "适配后自动 Eval")
}
