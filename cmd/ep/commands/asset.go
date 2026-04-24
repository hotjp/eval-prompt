package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/search"
	"github.com/spf13/cobra"
)

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "资产操作",
}

func init() {
	assetCmd.AddCommand(assetListCmd)
	assetCmd.AddCommand(assetShowCmd)
	assetCmd.AddCommand(assetCatCmd)
	assetCmd.AddCommand(assetCreateCmd)
	assetCmd.AddCommand(assetEditCmd)
	assetCmd.AddCommand(assetRmCmd)
}

// Global indexer instance for CLI
var cliIndexer service.AssetIndexer

func getIndexer() service.AssetIndexer {
	if cliIndexer == nil {
		cliIndexer = search.NewIndexer()
	}
	return cliIndexer
}

var assetListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有资产",
	RunE: func(cmd *cobra.Command, args []string) error {
		indexer := getIndexer()

		bizLine, _ := cmd.Flags().GetString("biz-line")
		tag, _ := cmd.Flags().GetString("tag")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		filters := service.SearchFilters{
			BizLine: bizLine,
		}
		if tag != "" {
			filters.Tags = []string{tag}
		}

		results, err := indexer.Search(cmd.Context(), "", filters)
		if err != nil {
			return fmt.Errorf("搜索失败: %w", err)
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(results)
		}

		// Table output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tNAME\tBIZ_LINE\tSTATE\tTAGS\n")
		for _, r := range results {
			tags := ""
			if len(r.Tags) > 0 {
				tags = r.Tags[0]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Name, r.BizLine, r.State, tags)
		}
		return w.Flush()
	},
}

var assetShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "显示资产详情",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		indexer := getIndexer()

		id := args[0]
		detail, err := indexer.GetByID(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("获取资产失败: %w", err)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(detail)
	},
}

var assetCatCmd = &cobra.Command{
	Use:   "cat <id>",
	Short: "纯文本输出（管道首选）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		indexer := getIndexer()

		id := args[0]
		detail, err := indexer.GetByID(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("获取资产失败: %w", err)
		}

		// Output raw content for piping
		fmt.Print(detail.Description)
		return nil
	},
}

var assetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建资产",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		file, _ := cmd.Flags().GetString("file")
		bizLine, _ := cmd.Flags().GetString("biz-line")

		if id == "" || name == "" {
			return fmt.Errorf("id 和 name 是必需的")
		}

		indexer := getIndexer()

		asset := service.Asset{
			ID:      id,
			Name:    name,
			State:   "created",
			BizLine: bizLine,
		}

		// Read file content if provided
		if file != "" {
			content, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("读取文件失败: %w", err)
			}
			asset.Description = string(content)
		}

		if err := indexer.Save(context.Background(), asset); err != nil {
			return fmt.Errorf("保存资产失败: %w", err)
		}

		fmt.Printf("资产已创建: %s\n", id)
		return nil
	},
}

var assetEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "编辑资产",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement edit functionality - would open editor
		return fmt.Errorf("not implemented: use 'ep asset cat <id> | $EDITOR' and 'ep asset create --id <id> --file <edited_file>'")
	},
}

var assetRmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "删除资产",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		indexer := getIndexer()

		if err := indexer.Delete(context.Background(), id); err != nil {
			return fmt.Errorf("删除资产失败: %w", err)
		}

		fmt.Printf("资产已删除: %s\n", id)
		return nil
	},
}

func init() {
	assetCreateCmd.Flags().String("id", "", "资产 ID")
	assetCreateCmd.Flags().String("name", "", "资产名称")
	assetCreateCmd.Flags().String("file", "", "资产文件路径")
	assetCreateCmd.Flags().String("biz-line", "", "业务线")

	assetListCmd.Flags().String("biz-line", "", "按业务线过滤")
	assetListCmd.Flags().String("tag", "", "按标签过滤")
	assetListCmd.Flags().Bool("json", false, "JSON 输出")
}
