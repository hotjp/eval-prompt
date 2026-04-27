package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: i18n.T(i18n.MsgSnapshotCmdShort, nil),
}

func init() {
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotDiffCmd)
	snapshotCmd.AddCommand(snapshotCheckoutCmd)
}

var snapshotListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: i18n.T(i18n.MsgSnapshotListShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		bridge := gitbridge.NewBridge()
		repoPath := getRepoPath()

		if err := bridge.Open(repoPath); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgSnapshotOpenRepoFailed, nil), err)
		}

		// Build file path pattern for this asset
		filePath := fmt.Sprintf("prompts/%s/*.md", assetID)

		commits, err := bridge.Log(context.Background(), filePath, limit)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgSnapshotHistoryFailed, nil), err)
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(commits)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "HASH\tCOMMITTER\tDATE\tMESSAGE\n")
		for _, c := range commits {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.ShortHash, c.Author, c.Timestamp.Format("2006-01-02"), c.Subject)
		}
		return w.Flush()
	},
}

var snapshotDiffCmd = &cobra.Command{
	Use:   "diff <id> <v1> <v2>",
	Short: i18n.T(i18n.MsgSnapshotDiffShort, nil),
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args[0] // assetID
		v1 := args[1]
		v2 := args[2]

		bridge := gitbridge.NewBridge()
		repoPath := getRepoPath()

		if err := bridge.Open(repoPath); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgSnapshotOpenRepoFailed, nil), err)
		}

		// Build commit references
		commit1 := fmt.Sprintf("%s^{commit}", v1)
		commit2 := fmt.Sprintf("%s^{commit}", v2)

		diff, err := bridge.Diff(context.Background(), commit1, commit2)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgSnapshotDiffFailed, nil), err)
		}

		fmt.Print(diff)
		return nil
	},
}

var snapshotCheckoutCmd = &cobra.Command{
	Use:   "checkout <id> <v>",
	Short: i18n.T(i18n.MsgSnapshotCheckoutShort, nil),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		version := args[1]

		fmt.Println(i18n.T(i18n.MsgSnapshotCheckoutStarted, pongo2.Context{"assetID": assetID, "version": version}))
		// TODO: Implement checkout via git worktree or file restore
		return fmt.Errorf("not implemented: use 'git checkout %s -- prompts/%s/", version, assetID)
	},
}

func init() {
	snapshotListCmd.Flags().Int("limit", 20, i18n.T(i18n.MsgFlagSnapshotLimit, nil))
	snapshotListCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))
}

func getRepoPath() string {
	// Get repo path from environment or current directory
	if path := os.Getenv("EP_REPO_PATH"); path != "" {
		return path
	}
	return "."
}
