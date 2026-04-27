package commands

import (
	"fmt"

	"github.com/eval-prompt/internal/i18n"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var optimizeCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgOptimizeCmd, nil),
	Short: i18n.T(i18n.MsgOptimizeCmdShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		strategy, _ := cmd.Flags().GetString("strategy")
		iterations, _ := cmd.Flags().GetInt("iterations")
		thresholdDelta, _ := cmd.Flags().GetFloat64("threshold-delta")
		autoPromote, _ := cmd.Flags().GetBool("auto-promote")

		fmt.Print(i18n.T(i18n.MsgOptimizeAsset, pongo2.Context{"asset": assetID}))
		fmt.Print(i18n.T(i18n.MsgOptimizeStrategy, pongo2.Context{"strategy": strategy}))
		fmt.Print(i18n.T(i18n.MsgOptimizeMaxIterations, pongo2.Context{"iterations": iterations}))
		fmt.Print(i18n.T(i18n.MsgOptimizeThresholdDelta, pongo2.Context{"delta": thresholdDelta}))

		if autoPromote {
			fmt.Print(i18n.T(i18n.MsgOptimizeAutoPromoteNote, nil))
		}

		// TODO: Implement auto-optimization workflow
		return fmt.Errorf("not implemented: requires EvalService + LLMInvoker integration")
	},
}

func init() {
	optimizeCmd.Flags().String("strategy", "failure_driven", i18n.T(i18n.MsgOptimizeFlagStrategy, nil))
	optimizeCmd.Flags().Int("iterations", 3, i18n.T(i18n.MsgOptimizeFlagIterations, nil))
	optimizeCmd.Flags().Float64("threshold-delta", 5, i18n.T(i18n.MsgOptimizeFlagThreshold, nil))
	optimizeCmd.Flags().Bool("auto-promote", false, i18n.T(i18n.MsgOptimizeFlagAutoPromote, nil))
}
