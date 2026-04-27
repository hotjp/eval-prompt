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
	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/internal/pathutil"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/yamlutil"
	"github.com/eval-prompt/plugins/search"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: i18n.T(i18n.MsgAssetCmdShort, nil),
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
	Short: i18n.T(i18n.MsgAssetListShort, nil),
	RunE: func(cmd *cobra.Command, args []string) error {
		indexer := getIndexer()

		bizLine, _ := cmd.Flags().GetString("biz-line")
		tag, _ := cmd.Flags().GetString("tag")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		filters := service.SearchFilters{
			AssetType: bizLine,
		}
		if tag != "" {
			filters.Tags = []string{tag}
		}

		results, err := indexer.Search(cmd.Context(), "", filters)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetSearchFailed, nil), err)
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(results)
		}

		// Table output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprint(w, i18n.T(i18n.MsgAssetListHeader, nil))
		for _, r := range results {
			tags := ""
			if len(r.Tags) > 0 {
				tags = r.Tags[0]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Name, r.AssetType, r.State, tags)
		}
		return w.Flush()
	},
}

var assetShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: i18n.T(i18n.MsgAssetShowShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		indexer := getIndexer()

		id := args[0]
		detail, err := indexer.GetByID(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetGetFailed, nil), err)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(detail)
	},
}

var assetCatCmd = &cobra.Command{
	Use:   "cat <id>",
	Short: i18n.T(i18n.MsgAssetCatShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		indexer := getIndexer()

		id := args[0]
		detail, err := indexer.GetByID(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetGetFailed, nil), err)
		}

		// Output raw content for piping
		fmt.Print(detail.Description)
		return nil
	},
}

var assetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: i18n.T(i18n.MsgAssetCreateShort, nil),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		file, _ := cmd.Flags().GetString("file")
		contentFlag, _ := cmd.Flags().GetString("content")
		bizLine, _ := cmd.Flags().GetString("biz-line")

		if name == "" {
			return fmt.Errorf(i18n.T(i18n.MsgAssetNameRequired, nil))
		}

		// --content and --file are mutually exclusive
		if contentFlag != "" && file != "" {
			return fmt.Errorf(i18n.T(i18n.MsgAssetContentFileConflict, nil))
		}

		// Determine content source: --content > --file > stdin
		var content string
		if contentFlag != "" {
			content = contentFlag
		} else if file != "" {
			fileContent, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileReadFailed, nil), err)
			}
			content = string(fileContent)
		} else {
			// Try reading from stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				stdinContent, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetStdinReadFailed, nil), err)
				}
				content = string(stdinContent)
			}
			if content == "" {
				return fmt.Errorf(i18n.T(i18n.MsgAssetInputRequired, nil))
			}
		}

		// Generate ID if not provided
		if id == "" {
			id = domain.NewULID()
		}

		// Validate ID format if provided
		if !domain.IsValidULID(id) {
			return fmt.Errorf("%s: %s", i18n.T(i18n.MsgAssetInvalidIDFormat, nil), id)
		}

		// Compute content hash
		hash := sha256.Sum256([]byte(content))
		contentHash := hex.EncodeToString(hash[:])

		// Ensure prompts directory exists
		promptsDir := "prompts"
		if err := os.MkdirAll(promptsDir, 0755); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetDirCreateFailed, nil), err)
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
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetMarkdownFormatFailed, nil), err)
		}
		if err := os.WriteFile(filePath, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileWriteFailed, nil), err)
		}

		// Save to indexer
		indexer := getIndexer()
		asset := service.Asset{
			ID:          id,
			Name:        name,
			Description: content,
			AssetType:     bizLine,
			ContentHash: contentHash,
			FilePath:    filePath,
			State:       "created",
		}
		if err := indexer.Save(context.Background(), asset); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetSaveFailed, nil), err)
		}

		// Call Reconcile to update index
		if _, err := indexer.Reconcile(context.Background()); err != nil {
			// Log but don't fail
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T(i18n.MsgAssetReconcileWarn, nil), err)
		}

		fmt.Println(i18n.T(i18n.MsgAssetCreateSuccess, pongo2.Context{"id": id}))
		return nil
	},
}

var assetEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: i18n.T(i18n.MsgAssetEditShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement edit functionality - would open editor
		return fmt.Errorf("not implemented: use 'ep asset cat <id> | $EDITOR' and 'ep asset create --id <id> --file <edited_file>'")
	},
}

var assetRmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: i18n.T(i18n.MsgAssetRmShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if err := pathutil.ValidateID(id); err != nil {
			return fmt.Errorf("invalid asset id: %w", err)
		}

		// Find the file
		filePath := filepath.Join("prompts", id+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(i18n.T(i18n.MsgAssetFileNotFound, pongo2.Context{"id": id}))
			}
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileReadError, nil), err)
		}

		// Parse front matter
		fm, _, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFrontmatterParseError, nil), err)
		}

		// Check state - must be archived
		if fm.State != "archived" {
			return fmt.Errorf("%s: %s", i18n.T(i18n.MsgAssetPleaseArchiveFirst, nil), id)
		}

		// Delete the file
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileDeleteFailed, nil), err)
		}

		// Also remove from index
		indexer := getIndexer()
		if err := indexer.Delete(context.Background(), id); err != nil {
			// Log but don't fail - file is already deleted
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T(i18n.MsgAssetIndexRemoveWarn, nil), err)
		}

		fmt.Println(i18n.T(i18n.MsgAssetDeleteSuccess, pongo2.Context{"id": id}))
		return nil
	},
}

var assetArchiveCmd = &cobra.Command{
	Use:   "archive <id>",
	Short: i18n.T(i18n.MsgAssetArchiveShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if err := pathutil.ValidateID(id); err != nil {
			return fmt.Errorf("invalid asset id: %w", err)
		}

		// Find the file
		filePath := filepath.Join("prompts", id+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(i18n.T(i18n.MsgAssetFileNotFound, pongo2.Context{"id": id}))
			}
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileReadError, nil), err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFrontmatterParseError, nil), err)
		}

		// Update state
		fm.State = "archived"

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetMarkdownFormatFailed, nil), err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileWriteFailed, nil), err)
		}

		fmt.Println(i18n.T(i18n.MsgAssetArchiveSuccess, pongo2.Context{"id": id}))
		return nil
	},
}

var assetRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: i18n.T(i18n.MsgAssetRestoreShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if err := pathutil.ValidateID(id); err != nil {
			return fmt.Errorf("invalid asset id: %w", err)
		}

		// Find the file
		filePath := filepath.Join("prompts", id+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(i18n.T(i18n.MsgAssetFileNotFound, pongo2.Context{"id": id}))
			}
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileReadError, nil), err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFrontmatterParseError, nil), err)
		}

		// Update state
		fm.State = "active"

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetMarkdownFormatFailed, nil), err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileWriteFailed, nil), err)
		}

		fmt.Println(i18n.T(i18n.MsgAssetRestoreSuccess, pongo2.Context{"id": id}))
		return nil
	},
}

var assetPromoteCmd = &cobra.Command{
	Use:   "promote <asset_id> <snapshot_id>",
	Short: i18n.T(i18n.MsgAssetPromoteShort, nil),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		snapshotID := args[1]

		if err := pathutil.ValidateID(assetID); err != nil {
			return fmt.Errorf("invalid asset id: %w", err)
		}

		// Find the file
		filePath := filepath.Join("prompts", assetID+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(i18n.T(i18n.MsgAssetFileNotFound, pongo2.Context{"id": assetID}))
			}
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileReadError, nil), err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFrontmatterParseError, nil), err)
		}

		// Update recommended_snapshot_id
		fm.RecommendedSnapshotID = snapshotID

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetMarkdownFormatFailed, nil), err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileWriteFailed, nil), err)
		}

		fmt.Printf("已标记推荐版本: %s -> %s\n", assetID, snapshotID)
		return nil
	},
}

var assetDemoteCmd = &cobra.Command{
	Use:   "demote <asset_id>",
	Short: i18n.T(i18n.MsgAssetDemoteShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]

		if err := pathutil.ValidateID(assetID); err != nil {
			return fmt.Errorf("invalid asset id: %w", err)
		}

		// Find the file
		filePath := filepath.Join("prompts", assetID+".md")
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(i18n.T(i18n.MsgAssetFileNotFound, pongo2.Context{"id": assetID}))
			}
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileReadError, nil), err)
		}

		// Parse front matter
		fm, markdownContent, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFrontmatterParseError, nil), err)
		}

		// Clear recommended_snapshot_id
		fm.RecommendedSnapshotID = ""

		// Write back
		newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetMarkdownFormatFailed, nil), err)
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetFileWriteFailed, nil), err)
		}

		fmt.Printf("已取消推荐版本: %s\n", assetID)
		return nil
	},
}

func init() {
	assetCreateCmd.Flags().String("id", "", i18n.T(i18n.MsgFlagAssetID, nil))
	assetCreateCmd.Flags().String("name", "", i18n.T(i18n.MsgFlagAssetName, nil))
	assetCreateCmd.Flags().String("file", "", i18n.T(i18n.MsgFlagAssetFile, nil))
	assetCreateCmd.Flags().String("content", "", i18n.T(i18n.MsgFlagAssetContent, nil))
	assetCreateCmd.Flags().String("biz-line", "", i18n.T(i18n.MsgFlagAssetBizLine, nil))

	assetListCmd.Flags().String("biz-line", "", i18n.T(i18n.MsgFlagBizLine, nil))
	assetListCmd.Flags().String("tag", "", i18n.T(i18n.MsgFlagAssetTag, nil))
	assetListCmd.Flags().Bool("json", false, i18n.T(i18n.MsgFlagJsonOutput, nil))
}
