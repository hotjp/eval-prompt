package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/eval-prompt/plugins/search"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: i18n.T(i18n.MsgTriggerCmdShort, nil),
}

func init() {
	triggerCmd.AddCommand(triggerMatchCmd)
}

var triggerMatchCmd = &cobra.Command{
	Use:   "match <input>",
	Short: i18n.T(i18n.MsgTriggerMatchShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
		top, _ := cmd.Flags().GetInt("top")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		indexer := search.NewIndexer()
		gitBridge := gitbridge.NewBridge()
		triggerService := service.NewTriggerService(indexer, gitBridge)

		matches, err := triggerService.MatchTrigger(cmd.Context(), input, top)
		if err != nil {
			return fmt.Errorf(i18n.T(i18n.MsgTriggerMatchFailed, pongo2.Context{"error": err.Error()}))
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(matches)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tNAME\tRELEVANCE\tDESCRIPTION\n")
		for _, m := range matches {
			desc := m.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%.2f\t%s\n", m.AssetID, m.Name, m.Relevance, desc)
		}
		return w.Flush()
	},
}

func init() {
	triggerMatchCmd.Flags().Int("top", 5, i18n.T(i18n.MsgFlagTop, nil))
	triggerMatchCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))
}
