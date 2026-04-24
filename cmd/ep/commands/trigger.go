package commands

import "github.com/spf13/cobra"

var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "触发匹配",
}

func init() {
	triggerCmd.AddCommand(triggerMatchCmd)
}

var triggerMatchCmd = &cobra.Command{
	Use:   "match <input>",
	Short: "匹配 Prompt",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return nil },
}
