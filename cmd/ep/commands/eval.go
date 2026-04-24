package commands

import "github.com/spf13/cobra"

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Eval 操作",
}

func init() {
	evalCmd.AddCommand(evalRunCmd)
	evalCmd.AddCommand(evalCasesCmd)
	evalCmd.AddCommand(evalCompareCmd)
	evalCmd.AddCommand(evalReportCmd)
	evalCmd.AddCommand(evalDiagnoseCmd)
}

var evalRunCmd = &cobra.Command{
	Use:   "run <id>",
	Short: "执行 Eval",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var evalCasesCmd = &cobra.Command{
	Use:   "cases <id>",
	Short: "列出测试用例",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var evalCompareCmd = &cobra.Command{
	Use:   "compare <id> <v1> <v2>",
	Short: "A/B 比对",
	Args:  cobra.ExactArgs(3),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var evalReportCmd = &cobra.Command{
	Use:   "report <run-id>",
	Short: "Eval 报告",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}

var evalDiagnoseCmd = &cobra.Command{
	Use:   "diagnose <run-id>",
	Short: "失败归因",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}
