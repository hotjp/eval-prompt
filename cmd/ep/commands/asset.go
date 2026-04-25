package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/yamlutil"
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
	assetCmd.AddCommand(assetArchiveCmd)
	assetCmd.AddCommand(assetRestoreCmd)
	assetCmd.AddCommand(assetRmCmd)
	assetCmd.AddCommand(assetPromoteCmd)
	assetCmd.AddCommand(assetDemoteCmd)
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
		contentFlag, _ := cmd.Flags().GetString("content")
		bizLine, _ := cmd.Flags().GetString("biz-line")

		if name == "" {
			return fmt.Errorf("name 是必需的")
		}

		// --content and --file are mutually exclusive
		if contentFlag != "" && file != "" {
			return fmt.Errorf("--content 和 --file 不能同时使用")
		}

		// Determine content source: --content > --file > stdin
		var content string
		if contentFlag != "" {
			content = contentFlag
		} else if file != "" {
			fileContent, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("读取文件失败: %w", err)
			}
			content = string(fileContent)
		} else {
			// Try reading from stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				stdinContent, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("读取 stdin 失败: %w", err)
				}
				content = string(stdinContent)
			}
			if content == "" {
				return fmt.Errorf("必须提供 --content、--file 或 stdin 输入")
			}
		}

		// Generate ID if not provided
		if id == "" {
			id = domain.NewULID()
		}

		// Validate ID format if provided
		if !domain.IsValidULID(id) {
			return fmt.Errorf("id 必须是有效的 ULID 格式")
		}

		// Compute content hash
		hash := sha256.Sum256([]byte(content))
		contentHash := hex.EncodeToString(hash[:])

		// Ensure prompts directory exists
		promptsDir := "prompts"
		if err := os.MkdirAll(promptsDir, 0755); err != nil {
			return fmt.Errorf("创建 prompts 目录失败: %w", err)
		}

		// Write asset file
		filePath := filepath.Join(promptsDir, id+".md")
		fm := &domain.FrontMatter{
			ID:          id,
			Name:        name,
			ContentHash: contentHash,
			State:       "created",
		}
		markdown, err := yamlutil.FormatMarkdown(fm, content)
		if err != nil {
			return fmt.Errorf("格式化 markdown 失败: %w", err)
		}
		if err := os.WriteFile(filePath, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}

		// Save to indexer
		indexer := getIndexer()
		asset := service.Asset{
			ID:          id,
			Name:        name,
			Description: content,
			BizLine:     bizLine,
			ContentHash: contentHash,
			FilePath:    filePath,
			State:       "created",
		}
		if err := indexer.Save(context.Background(), asset); err != nil {
			return fmt.Errorf("保存资产失败: %w", err)
		}

		// Call Reconcile to update index
		if _, err := indexer.Reconcile(context.Background()); err != nil {
			// Log but don't fail
			fmt.Fprintf(os.Stderr, "警告: Reconcile 失败: %v\n", err)
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
	Short: "删除资产（必须先 archive）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Find the file
		filePath := filepath.Join("prompts", id+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("资产文件不存在: %s", id)
			}
			return fmt.Errorf("读取资产文件失败: %w", err)
		}

		// Parse front matter
		fm, _, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("解析 front matter 失败: %w", err)
		}

		// Check state - must be archived
		if fm.State != "archived" {
			return fmt.Errorf("请先 archive: %s", id)
		}

		// Delete the file
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("删除资产文件失败: %w", err)
		}

		// Also remove from index
		indexer := getIndexer()
		if err := indexer.Delete(context.Background(), id); err != nil {
			// Log but don't fail - file is already deleted
			fmt.Fprintf(os.Stderr, "警告: 从索引删除失败: %v\n", err)
		}

		fmt.Printf("资产已删除: %s\n", id)
		return nil
	},
}

var assetArchiveCmd = &cobra.Command{
	Use:   "archive <id>",
	Short: "归档资产",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Find the file
		filePath := filepath.Join("prompts", id+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("资产文件不存在: %s", id)
			}
			return fmt.Errorf("读取资产文件失败: %w", err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("解析 front matter 失败: %w", err)
		}

		// Update state
		fm.State = "archived"

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}

		fmt.Printf("资产已归档: %s\n", id)
		return nil
	},
}

var assetRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "恢复资产",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Find the file
		filePath := filepath.Join("prompts", id+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("资产文件不存在: %s", id)
			}
			return fmt.Errorf("读取资产文件失败: %w", err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("解析 front matter 失败: %w", err)
		}

		// Update state
		fm.State = "active"

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}

		fmt.Printf("资产已恢复: %s\n", id)
		return nil
	},
}

var assetPromoteCmd = &cobra.Command{
	Use:   "promote <asset_id> <snapshot_id>",
	Short: "标记推荐版本",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		snapshotID := args[1]

		// Find the file
		filePath := filepath.Join("prompts", assetID+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("资产文件不存在: %s", assetID)
			}
			return fmt.Errorf("读取资产文件失败: %w", err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("解析 front matter 失败: %w", err)
		}

		// Update recommended_snapshot_id
		fm.RecommendedSnapshotID = snapshotID

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}

		fmt.Printf("已标记推荐版本: %s -> %s\n", assetID, snapshotID)
		return nil
	},
}

var assetDemoteCmd = &cobra.Command{
	Use:   "demote <asset_id>",
	Short: "取消推荐版本",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]

		// Find the file
		filePath := filepath.Join("prompts", assetID+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("资产文件不存在: %s", assetID)
			}
			return fmt.Errorf("读取资产文件失败: %w", err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("解析 front matter 失败: %w", err)
		}

		// Clear recommended_snapshot_id
		fm.RecommendedSnapshotID = ""

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}

		fmt.Printf("已取消推荐版本: %s\n", assetID)
		return nil
	},
}

func init() {
	assetCreateCmd.Flags().String("id", "", "资产 ID (可选，默认自动生成 ULID)")
	assetCreateCmd.Flags().String("name", "", "资产名称 (必需)")
	assetCreateCmd.Flags().String("file", "", "资产文件路径")
	assetCreateCmd.Flags().String("content", "", "资产内容 (支持 stdin)")
	assetCreateCmd.Flags().String("biz-line", "", "业务线")

	assetListCmd.Flags().String("biz-line", "", "按业务线过滤")
	assetListCmd.Flags().String("tag", "", "按标签过滤")
	assetListCmd.Flags().Bool("json", false, "JSON 输出")
}
