package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/plugins/search"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: i18n.T(i18n.MsgLabelCmdShort, nil),
}

func init() {
	labelCmd.AddCommand(labelListCmd)
	labelCmd.AddCommand(labelSetCmd)
	labelCmd.AddCommand(labelUnsetCmd)
}

var labelListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: i18n.T(i18n.MsgLabelListShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		indexer := search.NewIndexer()
		detail, err := indexer.GetByID(context.Background(), assetID)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgLabelGetFailed, nil), err)
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(detail.Labels)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "NAME\tSNAPSHOT\tUPDATED_AT\n")
		for _, l := range detail.Labels {
			fmt.Fprintf(w, "%s\t%s\t%s\n", l.Name, l.SnapshotID, l.UpdatedAt.Format("2006-01-02"))
		}
		return w.Flush()
	},
}

var labelSetCmd = &cobra.Command{
	Use:   "set <id> <name> <v>",
	Short: i18n.T(i18n.MsgLabelSetShort, nil),
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		name := args[1]
		version := args[2]

		fmt.Println(i18n.T(i18n.MsgLabelSetOutput, pongo2.Context{"name": name, "assetID": assetID, "version": version}))
		// TODO: Implement label set via storage
		return fmt.Errorf("not implemented: requires storage integration")
	},
}

var labelUnsetCmd = &cobra.Command{
	Use:   "unset <id> <name>",
	Short: i18n.T(i18n.MsgLabelUnsetShort, nil),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		name := args[1]

		fmt.Println(i18n.T(i18n.MsgLabelUnsetOutput, pongo2.Context{"name": name, "assetID": assetID}))
		// TODO: Implement label unset via storage
		return fmt.Errorf("not implemented: requires storage integration")
	},
}

func init() {
	labelListCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))
}
