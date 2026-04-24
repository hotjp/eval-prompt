package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ep",
	Short: "eval-prompt - Prompt 资产管理工具",
	Long: `eval-prompt 是一个团队级 Prompt 资产管理工具。
	以纯 Go 二进制单文件形式分发，通过浏览器访问 Web UI，
	Agent 通过 CLI/MCP 协议消费，所有数据不出域。`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(assetCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(labelCmd)
	rootCmd.AddCommand(evalCmd)
	rootCmd.AddCommand(triggerCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(adaptCmd)
	rootCmd.AddCommand(optimizeCmd)
}
