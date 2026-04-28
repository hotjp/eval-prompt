package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/internal/pathutil"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/yamlutil"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/eval-prompt/plugins/search"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

// newEvalService creates a configured EvalService for CLI use.
func newEvalService(evalsDir string) *service.EvalService {
	baseDir := ".evals"
	evalsBase := evalsDir
	if evalsBase == "" {
		evalsBase = "evals"
	}
	svc := service.NewEvalService().
		WithExecutionStore(service.NewExecutionFileStore(filepath.Join(baseDir, "executions"))).
		WithCallStore(service.NewLLMCallFileStore(filepath.Join(baseDir, "calls"))).
		WithEvalsDir(evalsBase)
	return svc
}

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: i18n.T(i18n.MsgEvalCmdShort, nil),
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
	Short: i18n.T(i18n.MsgEvalRunShort, nil),
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
		evalsDir, _ := cmd.Flags().GetString("evals-dir")

		if snapshot == "" {
			snapshot = "latest"
		}

		evalService := newEvalService(evalsDir)
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
			return errors.New(i18n.T(i18n.MsgEvalRunFailed, pongo2.Context{"error": err.Error()}))
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
				fmt.Println(i18n.T(i18n.MsgSyncReconcileWarning, pongo2.Context{"error": err.Error()}))
			}
			gitBridge := gitbridge.NewBridge()
			if err := gitBridge.Open(wd); err == nil {
				indexer.SetGitBridge(gitBridge)
				report, err := indexer.Reconcile(context.Background())
				if err == nil {
					fmt.Println(i18n.T(i18n.MsgSyncReconcileDone, pongo2.Context{"count": report.Added + report.Updated + report.Deleted}))
				}
			}
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(execution)
		}

		fmt.Println(i18n.T(i18n.MsgEvalRunStarted, pongo2.Context{"id": execution.ID}))
		fmt.Println(i18n.T(i18n.MsgEvalRunStatus, pongo2.Context{"status": execution.Status}))
		if concurrency > 0 {
			fmt.Println(i18n.T(i18n.MsgEvalRunConcurrency, pongo2.Context{"value": concurrency}))
		}
		if model != "" {
			fmt.Println(i18n.T(i18n.MsgEvalRunModel, pongo2.Context{"model": model}))
		}
		if temperature > 0 {
			fmt.Println(i18n.T(i18n.MsgEvalRunTemperature, pongo2.Context{"value": temperature}))
		}
		return nil
	},
}

var evalCasesCmd = &cobra.Command{
	Use:   "cases <id>",
	Short: i18n.T(i18n.MsgEvalCasesShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")
		evalsDir, _ := cmd.Flags().GetString("evals-dir")

		evalService := newEvalService(evalsDir)

		cases, err := evalService.ListEvalCases(context.Background(), assetID)
		if err != nil {
			return errors.New(i18n.T(i18n.MsgEvalCasesFailed, pongo2.Context{"error": err.Error()}))
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
	Short: i18n.T(i18n.MsgEvalCompareShort, nil),
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		v1 := args[1]
		v2 := args[2]
		format, _ := cmd.Flags().GetString("format")

		evalService := newEvalService("")
		result, err := evalService.CompareEval(context.Background(), assetID, v1, v2)
		if err != nil {
			return errors.New(i18n.T(i18n.MsgEvalCompareFailed, pongo2.Context{"error": err.Error()}))
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
	Short: i18n.T(i18n.MsgEvalReportShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		evalService := newEvalService("")
		report, err := evalService.GenerateReport(context.Background(), runID)
		if err != nil {
			return errors.New(i18n.T(i18n.MsgEvalReportFailed, pongo2.Context{"error": err.Error()}))
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(report)
		}

		fmt.Println(i18n.T(i18n.MsgEvalReportComplete, pongo2.Context{"run_id": runID, "score": report.RubricScore, "status": report.Status}))
		return nil
	},
}

var evalDiagnoseCmd = &cobra.Command{
	Use:   "diagnose <run-id>",
	Short: i18n.T(i18n.MsgEvalDiagnoseShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]
		format, _ := cmd.Flags().GetString("format")

		evalService := newEvalService("")
		diagnosis, err := evalService.DiagnoseEval(context.Background(), runID)
		if err != nil {
			return errors.New(i18n.T(i18n.MsgEvalDiagnoseFailed, pongo2.Context{"error": err.Error()}))
		}

		switch format {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(diagnosis)
		default:
			fmt.Println(i18n.T(i18n.MsgEvalDiagnoseComplete, pongo2.Context{"run_id": runID}))
			fmt.Println(i18n.T(i18n.MsgEvalDiagnoseSeverity, pongo2.Context{"severity": diagnosis.OverallSeverity}))
			fmt.Println(i18n.T(i18n.MsgEvalDiagnoseStrategy, pongo2.Context{"strategy": diagnosis.RecommendedStrategy}))
			for _, f := range diagnosis.Findings {
				fmt.Printf("## [%s] %s\n", f.Severity, f.Category)
				fmt.Println(i18n.T(i18n.MsgEvalDiagnoseLocation, pongo2.Context{"location": f.Location}))
				fmt.Println(i18n.T(i18n.MsgEvalDiagnoseProblem, pongo2.Context{"problem": f.Problem}))
				fmt.Println(i18n.T(i18n.MsgEvalDiagnoseSuggestion, pongo2.Context{"suggestion": f.Suggestion}))
			}
		}

		return nil
	},
}

var evalSetupCmd = &cobra.Command{
	Use:   "setup <asset_id>",
	Short: i18n.T(i18n.MsgEvalSetupShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		evalsDir, _ := cmd.Flags().GetString("evals-dir")
		model, _ := cmd.Flags().GetString("model")

		if err := pathutil.ValidateID(assetID); err != nil {
			return errors.New(i18n.T(i18n.MsgEvalSetupInvalidID, pongo2.Context{"error": err.Error()}))
		}

		if evalsDir == "" {
			evalsDir = "evals" // default to "evals" directory
		}

		if model == "" {
			model = "gpt-4o" // default model
		}

		// Create evals directory if it doesn't exist
		if err := os.MkdirAll(evalsDir, 0755); err != nil {
			return errors.New(i18n.T(i18n.MsgEvalSetupCreateDirFailed, pongo2.Context{"error": err.Error()}))
		}

		// Generate eval prompt file path
		evalFilePath := filepath.Join(evalsDir, assetID+".md")

		// Check if file already exists
		if _, err := os.Stat(evalFilePath); err == nil {
			return errors.New(i18n.T(i18n.MsgEvalSetupAlreadyExists, pongo2.Context{"path": evalFilePath}))
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
			return errors.New(i18n.T(i18n.MsgEvalSetupFormatFailed, pongo2.Context{"error": err.Error()}))
		}

		// Write the file
		if err := os.WriteFile(evalFilePath, []byte(mdContent), 0644); err != nil {
			return errors.New(i18n.T(i18n.MsgEvalSetupWriteFailed, pongo2.Context{"error": err.Error()}))
		}

		fmt.Println(i18n.T(i18n.MsgEvalSetupComplete, pongo2.Context{"path": evalFilePath}))
		fmt.Println(i18n.T(i18n.MsgEvalSetupModel, pongo2.Context{"model": model}))
		return nil
	},
}

var evalListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T(i18n.MsgEvalListShort, nil),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID, _ := cmd.Flags().GetString("asset-id")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if assetID == "" {
			return errors.New(i18n.T(i18n.MsgEvalListAssetIDRequired, nil))
		}

		evalService := newEvalService("")
		runs, err := evalService.ListEvalRuns(context.Background(), assetID)
		if err != nil {
			return errors.New(i18n.T(i18n.MsgEvalListFailed, pongo2.Context{"error": err.Error()}))
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
	Short: i18n.T(i18n.MsgEvalCancelShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		executionID := args[0]
		evalService := newEvalService("")

		if err := evalService.CancelExecution(context.Background(), executionID); err != nil {
			return errors.New(i18n.T(i18n.MsgEvalCancelFailed, pongo2.Context{"error": err.Error()}))
		}

		fmt.Println(i18n.T(i18n.MsgEvalCancelStarted, pongo2.Context{"id": executionID}))
		return nil
	},
}

func init() {
	evalRunCmd.Flags().String("snapshot", "", i18n.T(i18n.MsgFlagSnapshot, nil))
	evalRunCmd.Flags().StringSlice("case", []string{}, i18n.T(i18n.MsgFlagCase, nil))
	evalRunCmd.Flags().Int("concurrency", 0, i18n.T(i18n.MsgFlagConcurrency, nil))
	evalRunCmd.Flags().String("model", "", i18n.T(i18n.MsgFlagModel, nil))
	evalRunCmd.Flags().Float64("temperature", 0, i18n.T(i18n.MsgFlagTemperature, nil))
	evalRunCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))
	evalRunCmd.Flags().Bool("no-sync", false, i18n.T(i18n.MsgFlagNoSync, nil))
	evalRunCmd.Flags().String("dir", "", i18n.T(i18n.MsgFlagDir, nil))
	evalRunCmd.Flags().String("evals-dir", "", i18n.T(i18n.MsgFlagEvalsDir, nil))

	evalCasesCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))
	evalCasesCmd.Flags().String("evals-dir", "", i18n.T(i18n.MsgFlagEvalsDir, nil))

	evalCompareCmd.Flags().String("format", "table", i18n.T(i18n.MsgFlagFormat, nil))

	evalReportCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))

	evalDiagnoseCmd.Flags().String("format", "markdown", i18n.T(i18n.MsgFlagFormat, nil))

	evalSetupCmd.Flags().String("evals-dir", "evals", i18n.T(i18n.MsgFlagEvalsDir, nil))
	evalSetupCmd.Flags().String("model", "gpt-4o", i18n.T(i18n.MsgFlagModel, nil))

	evalListCmd.Flags().String("asset-id", "", i18n.T(i18n.MsgFlagAssetID, nil))
	evalListCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))

	evalCancelCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))
}
