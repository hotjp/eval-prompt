package commands

import (
	"fmt"

	"github.com/eval-prompt/internal/i18n"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var adaptCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgAdaptCmd, nil),
	Short: i18n.T(i18n.MsgAdaptCmdShort, nil),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		version := args[1]
		fromModel, _ := cmd.Flags().GetString("from")
		toModel, _ := cmd.Flags().GetString("to")
		saveAs, _ := cmd.Flags().GetString("save-as")
		autoEval, _ := cmd.Flags().GetBool("auto-eval")

		fmt.Print(i18n.T(i18n.MsgAdaptAsset, pongo2.Context{"asset": assetID, "version": version}))
		fmt.Print(i18n.T(i18n.MsgAdaptFromTo, pongo2.Context{"from": fromModel, "to": toModel}))

		if saveAs != "" {
			fmt.Print(i18n.T(i18n.MsgAdaptSaveAs, pongo2.Context{"name": saveAs}))
		}

		if autoEval {
			fmt.Print(i18n.T(i18n.MsgAdaptAutoEval, nil))
		}

		// TODO: Implement model adaptation via ModelAdapter plugin
		return fmt.Errorf("not implemented: requires ModelAdapter plugin")
	},
}

func init() {
	adaptCmd.Flags().String("from", "", i18n.T(i18n.MsgFlagModel, nil))
	adaptCmd.Flags().String("to", "", i18n.T(i18n.MsgFlagModel, nil))
	adaptCmd.Flags().String("save-as", "", i18n.T(i18n.MsgAdaptSaveAs, nil))
	adaptCmd.Flags().Bool("auto-eval", false, i18n.T(i18n.MsgAdaptAutoEval, nil))
}
