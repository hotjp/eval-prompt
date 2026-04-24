package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/eval-prompt/plugins/search"
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "标签操作",
}

func init() {
	labelCmd.AddCommand(labelListCmd)
	labelCmd.AddCommand(labelSetCmd)
	labelCmd.AddCommand(labelUnsetCmd)
}

var labelListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "列出标签",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		indexer := search.NewIndexer()
		detail, err := indexer.GetByID(context.Background(), assetID)
		if err != nil {
			return fmt.Errorf("获取资产失败: %w", err)
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
	Short: "设置标签",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		name := args[1]
		version := args[2]

		fmt.Printf("设置标签 %s -> %s (版本 %s)\n", name, assetID, version)
		// TODO: Implement label set via storage
		return fmt.Errorf("not implemented: requires storage integration")
	},
}

var labelUnsetCmd = &cobra.Command{
	Use:   "unset <id> <name>",
	Short: "取消标签",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		name := args[1]

		fmt.Printf("取消标签 %s 从 %s\n", name, assetID)
		// TODO: Implement label unset via storage
		return fmt.Errorf("not implemented: requires storage integration")
	},
}

func init() {
	labelListCmd.Flags().Bool("json", false, "JSON 输出")
}
