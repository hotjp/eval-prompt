package commands

import "github.com/spf13/cobra"

var optimizeCmd = &cobra.Command{
	Use:   "optimize <id>",
	Short: "Agent 自主优化",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

func init() {
	optimizeCmd.Flags().String("strategy", "failure_driven", "优化策略: failure_driven | score_max | compact")
	optimizeCmd.Flags().Int("iterations", 3, "最大迭代次数")
	optimizeCmd.Flags().Float64("threshold-delta", 5, "得分提升阈值")
	optimizeCmd.Flags().Bool("auto-promote", false, "优化通过后自动申请 Label 晋升")
}
