package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/pathutil"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/yamlutil"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/eval-prompt/plugins/search"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Eval operations",
}

func init() {
	evalCmd.AddCommand(evalRunCmd)
	evalCmd.AddCommand(evalCasesCmd)
	evalCmd.AddCommand(evalCompareCmd)
	evalCmd.AddCommand(evalReportCmd)
	evalCmd.AddCommand(evalDiagnoseCmd)
	evalCmd.AddCommand(evalSetupCmd)
	evalCmd.AddCommand(evalListCmd)
	evalCmd.AddCommand(evalCancelCmd)
}

var evalRunCmd = &cobra.Command{
	Use:   "run <id>",
	Short: "Run Eval",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		snapshot, _ := cmd.Flags().GetString("snapshot")
		caseIDs, _ := cmd.Flags().GetStringSlice("case")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		model, _ := cmd.Flags().GetString("model")
		temperature, _ := cmd.Flags().GetFloat64("temperature")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		noSync, _ := cmd.Flags().GetBool("no-sync")

		if snapshot == "" {
			snapshot = "latest"
		}

		evalService := service.NewEvalService()
		svcReq := &service.RunEvalRequest{
			AssetID:         assetID,
			SnapshotVersion: snapshot,
			EvalCaseIDs:     caseIDs,
			Concurrency:     concurrency,
			Model:           model,
			Temperature:     temperature,
		}
		execution, err := evalService.RunEval(context.Background(), svcReq)
		if err != nil {
			return fmt.Errorf("Eval execution failed: %w", err)
		}

		if !noSync {
			// Auto-reconcile to update index with new eval results
			wd, _ := cmd.Flags().GetString("dir")
			if wd == "" {
				wd, _ = os.Getwd()
			}
			indexer := search.Default()
			indexer.SetPersistDir(filepath.Join(wd, ".eval-prompt"))
			if err := indexer.Load(); err != nil {
				fmt.Printf("Warning: failed to load index: %v\n", err)
			}
			gitBridge := gitbridge.NewBridge()
			if err := gitBridge.Open(wd); err == nil {
				indexer.SetGitBridge(gitBridge)
				report, err := indexer.Reconcile(context.Background())
				if err == nil {
					fmt.Printf("Index synced: 新增 %d, 更新 %d, 删除 %d\n", report.Added, report.Updated, report.Deleted)
				}
			}
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(execution)
		}

		fmt.Printf("Eval run started: %s\n", execution.ID)
		fmt.Printf("Status: %s\n", execution.Status)
		if concurrency > 0 {
			fmt.Printf("Concurrency: %d\n", concurrency)
		}
		if model != "" {
			fmt.Printf("Model: %s\n", model)
		}
		if temperature > 0 {
			fmt.Printf("Temperature: %.2f\n", temperature)
		}
		return nil
	},
}

var evalCasesCmd = &cobra.Command{
	Use:   "cases <id>",
	Short: "List test cases",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		evalService := service.NewEvalService()

		cases, err := evalService.ListEvalCases(context.Background(), assetID)
		if err != nil {
			return fmt.Errorf("Failed to get eval cases: %w", err)
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(cases)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tNAME\tSHOULD_TRIGGER\n")
		for _, c := range cases {
			triggerStr := "false"
			if c.ShouldTrigger {
				triggerStr = "true"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, c.Name, triggerStr)
		}
		return w.Flush()
	},
}

var evalCompareCmd = &cobra.Command{
	Use:   "compare <id> <v1> <v2>",
	Short: "A/B Compare",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		v1 := args[1]
		v2 := args[2]
		format, _ := cmd.Flags().GetString("format")

		evalService := service.NewEvalService()
		result, err := evalService.CompareEval(context.Background(), assetID, v1, v2)
		if err != nil {
			return fmt.Errorf("Compare failed: %w", err)
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
	Short: "Eval Report",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		evalService := service.NewEvalService()
		report, err := evalService.GenerateReport(context.Background(), runID)
		if err != nil {
			return fmt.Errorf("Failed to generate report: %w", err)
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(report)
		}

		fmt.Printf("Eval Report: %s\n", runID)
		fmt.Printf("Score: %d/%d\n", report.RubricScore, 100)
		fmt.Printf("状态: %s\n", report.Status)
		return nil
	},
}

var evalDiagnoseCmd = &cobra.Command{
	Use:   "diagnose <run-id>",
	Short: "Failure Diagnosis",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]
		format, _ := cmd.Flags().GetString("format")

		evalService := service.NewEvalService()
		diagnosis, err := evalService.DiagnoseEval(context.Background(), runID)
		if err != nil {
			return fmt.Errorf("Failed to diagnose: %w", err)
		}

		switch format {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(diagnosis)
		default:
			fmt.Printf("Diagnosis Report: %s\n\n", runID)
			fmt.Printf("Severity: %s\n", diagnosis.OverallSeverity)
			fmt.Printf("Recommended Strategy: %s\n\n", diagnosis.RecommendedStrategy)
			for _, f := range diagnosis.Findings {
				fmt.Printf("## [%s] %s\n", f.Severity, f.Category)
				fmt.Printf("Location: %s\n", f.Location)
				fmt.Printf("Problem: %s\n", f.Problem)
				fmt.Printf("Suggestion: %s\n\n", f.Suggestion)
			}
		}

		return nil
	},
}

var evalSetupCmd = &cobra.Command{
	Use:   "setup <asset_id>",
	Short: "Create Eval Prompt Template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		evalsDir, _ := cmd.Flags().GetString("evals-dir")
		model, _ := cmd.Flags().GetString("model")

		if err := pathutil.ValidateID(assetID); err != nil {
			return fmt.Errorf("invalid asset id: %w", err)
		}

		if evalsDir == "" {
			evalsDir = "evals" // default to "evals" directory
		}

		if model == "" {
			model = "gpt-4o" // default model
		}

		// Create evals directory if it doesn't exist
		if err := os.MkdirAll(evalsDir, 0755); err != nil {
			return fmt.Errorf("Failed to create evals directory: %w", err)
		}

		// Generate eval prompt file path
		evalFilePath := filepath.Join(evalsDir, assetID+".md")

		// Check if file already exists
		if _, err := os.Stat(evalFilePath); err == nil {
			return fmt.Errorf("eval prompt file already exists: %s", evalFilePath)
		}

		// Create eval prompt front matter
		fm := &domain.EvalPromptFrontMatter{
			ID:          assetID,
			Name:        fmt.Sprintf("Eval Prompt for Asset %s", assetID),
			Version:     "v1.0.0",
			ContentHash: "", // Will be computed when content is finalized
			State:       "active",
			Tags:        []string{},
			EvalCaseIDs: []string{},
			Model:       model,
		}

		// Create eval prompt content template
		content := `# Eval Prompt

This is the evaluation prompt template for asset ` + assetID + `.

## Instructions
Describe how to evaluate the prompt asset here.

## Evaluation Criteria
- Criterion 1: Describe what to check
- Criterion 2: Describe what to verify

## Expected Output
Describe the expected output format.
`

		// Format the complete markdown file
		mdContent, err := yamlutil.FormatEvalPromptMarkdown(fm, content)
		if err != nil {
			return fmt.Errorf("Failed to format eval prompt: %w", err)
		}

		// Write the file
		if err := os.WriteFile(evalFilePath, []byte(mdContent), 0644); err != nil {
			return fmt.Errorf("Failed to write eval prompt file: %w", err)
		}

		fmt.Printf("Eval Prompt template created: %s\n", evalFilePath)
		fmt.Printf("Model: %s\n", model)
		return nil
	},
}

var evalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Eval Executions",
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID, _ := cmd.Flags().GetString("asset-id")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if assetID == "" {
			return fmt.Errorf("--asset-id is required")
		}

		evalService := service.NewEvalService()
		runs, err := evalService.ListEvalRuns(context.Background(), assetID)
		if err != nil {
			return fmt.Errorf("Failed to list eval executions: %w", err)
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(runs)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tSTATUS\tDETERMINISTIC_SCORE\tRUBRIC_SCORE\tCREATED_AT\n")
		for _, run := range runs {
			fmt.Fprintf(w, "%s\t%s\t%.2f\t%d\t%s\n",
				run.ID, run.Status, run.DeterministicScore, run.RubricScore, run.CreatedAt)
		}
		return w.Flush()
	},
}

var evalCancelCmd = &cobra.Command{
	Use:   "cancel <execution-id>",
	Short: "取消正在执行的 Eval",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		executionID := args[0]
		evalService := service.NewEvalService()

		if err := evalService.CancelExecution(context.Background(), executionID); err != nil {
			return fmt.Errorf("Failed to cancel eval: %w", err)
		}

		fmt.Printf("Eval 已取消: %s\n", executionID)
		return nil
	},
}

func init() {
	evalRunCmd.Flags().String("snapshot", "", "快照版本")
	evalRunCmd.Flags().StringSlice("case", []string{}, "指定测试用例 ID")
	evalRunCmd.Flags().Int("concurrency", 0, "并发数")
	evalRunCmd.Flags().String("model", "", "使用的模型")
	evalRunCmd.Flags().Float64("temperature", 0, "温度参数")
	evalRunCmd.Flags().Bool("json", false, "JSON 输出")
	evalRunCmd.Flags().Bool("no-sync", false, "跳过自动同步索引")
	evalRunCmd.Flags().String("dir", "", "项目目录路径")

	evalCasesCmd.Flags().Bool("json", false, "JSON 输出")

	evalCompareCmd.Flags().String("format", "table", "输出格式: table|json|markdown")

	evalReportCmd.Flags().Bool("json", false, "JSON 输出")

	evalDiagnoseCmd.Flags().String("format", "markdown", "输出格式: json|markdown")

	evalSetupCmd.Flags().String("evals-dir", "evals", "Eval 提示词目录")
	evalSetupCmd.Flags().String("model", "gpt-4o", "使用的模型")

	evalListCmd.Flags().String("asset-id", "", "资产 ID (必需)")
	evalListCmd.Flags().Bool("json", false, "JSON 输出")

	evalCancelCmd.Flags().Bool("json", false, "JSON 输出")
}
