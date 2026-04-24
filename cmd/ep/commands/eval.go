package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/eval-prompt/internal/service"
	"github.com/spf13/cobra"
)

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
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		snapshot, _ := cmd.Flags().GetString("snapshot")
		caseIDs, _ := cmd.Flags().GetStringSlice("case")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if snapshot == "" {
			snapshot = "latest"
		}

		evalService := service.NewEvalService()
		run, err := evalService.RunEval(context.Background(), assetID, snapshot, caseIDs)
		if err != nil {
			if service.ErrNotImplemented != nil {
				return fmt.Errorf("Eval 执行失败: %w", err)
			}
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(run)
		}

		fmt.Printf("Eval 运行已启动: %s\n", run.ID)
		fmt.Printf("状态: %s\n", run.Status)
		return nil
	},
}

var evalCasesCmd = &cobra.Command{
	Use:   "cases <id>",
	Short: "列出测试用例",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args[0] // assetID
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// TODO: Get eval cases from storage
		var cases []service.AssetSummary

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(cases)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tNAME\tSHOULD_TRIGGER\n")
		for _, c := range cases {
			fmt.Fprintf(w, "%s\t%s\t\n", c.ID, c.Name)
		}
		return w.Flush()
	},
}

var evalCompareCmd = &cobra.Command{
	Use:   "compare <id> <v1> <v2>",
	Short: "A/B 比对",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		v1 := args[1]
		v2 := args[2]
		format, _ := cmd.Flags().GetString("format")

		evalService := service.NewEvalService()
		result, err := evalService.CompareEval(context.Background(), assetID, v1, v2)
		if err != nil {
			if service.ErrNotImplemented != nil {
				return fmt.Errorf("比对失败: %w", err)
			}
		}

		switch format {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(result)
		case "markdown":
			fmt.Printf("## %s: %s vs %s\n\n", assetID, v1, v2)
			if result.Run1 != nil {
				fmt.Printf("| 版本 | 得分 | 状态 |\n")
				fmt.Printf("|------|------|------|\n")
				fmt.Printf("| %s | %d | %s |\n", v1, result.Run1.RubricScore, result.Run1.Status)
				fmt.Printf("| %s | %d | %s |\n", v2, result.Run2.RubricScore, result.Run2.Status)
				fmt.Printf("\n**得分差: %+d**\n", result.ScoreDelta)
			}
		default:
			fmt.Printf("%s: %s vs %s\n", assetID, v1, v2)
			fmt.Printf("得分差: %+d\n", result.ScoreDelta)
		}

		return nil
	},
}

var evalReportCmd = &cobra.Command{
	Use:   "report <run-id>",
	Short: "Eval 报告",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		evalService := service.NewEvalService()
		report, err := evalService.GenerateReport(context.Background(), runID)
		if err != nil {
			if service.ErrNotImplemented != nil {
				return fmt.Errorf("生成报告失败: %w", err)
			}
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(report)
		}

		fmt.Printf("Eval 报告: %s\n", runID)
		fmt.Printf("得分: %d/%d\n", report.RubricScore, 100)
		fmt.Printf("状态: %s\n", report.Status)
		return nil
	},
}

var evalDiagnoseCmd = &cobra.Command{
	Use:   "diagnose <run-id>",
	Short: "失败归因",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]
		format, _ := cmd.Flags().GetString("format")

		evalService := service.NewEvalService()
		diagnosis, err := evalService.DiagnoseEval(context.Background(), runID)
		if err != nil {
			if service.ErrNotImplemented != nil {
				return fmt.Errorf("诊断失败: %w", err)
			}
		}

		switch format {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(diagnosis)
		default:
			fmt.Printf("诊断报告: %s\n\n", runID)
			fmt.Printf("严重程度: %s\n", diagnosis.OverallSeverity)
			fmt.Printf("推荐策略: %s\n\n", diagnosis.RecommendedStrategy)
			for _, f := range diagnosis.Findings {
				fmt.Printf("## [%s] %s\n", f.Severity, f.Category)
				fmt.Printf("位置: %s\n", f.Location)
				fmt.Printf("问题: %s\n", f.Problem)
				fmt.Printf("建议: %s\n\n", f.Suggestion)
			}
		}

		return nil
	},
}

func init() {
	evalRunCmd.Flags().String("snapshot", "", "快照版本")
	evalRunCmd.Flags().StringSlice("case", []string{}, "指定测试用例 ID")
	evalRunCmd.Flags().Bool("json", false, "JSON 输出")

	evalCasesCmd.Flags().Bool("json", false, "JSON 输出")

	evalCompareCmd.Flags().String("format", "table", "输出格式: table|json|markdown")

	evalReportCmd.Flags().Bool("json", false, "JSON 输出")

	evalDiagnoseCmd.Flags().String("format", "markdown", "输出格式: json|markdown")
}
