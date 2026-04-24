package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/yamlutil"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <source>...",
	Short: "批量导入 Prompt 资产",
	Long: `批量导入 Prompt 资产从文件或目录。

支持导入方式:
  - 单文件: ep import ./my-prompt.txt
  - Glob 模式: ep import "./prompts/**/*.md"
  - 多文件: ep import file1.txt file2.md
  - 目录: ep import ./prompts/

导入后生成 prompts/<id>.md 文件，包含 YAML front matter。`,
	Args: cobra.MinimumNArgs(1),
	RunE: runImport,
}

func init() {
	importCmd.Flags().String("biz-line", "", "指定业务线")
	importCmd.Flags().Bool("dry-run", false, "仅预览，不实际导入")
	importCmd.Flags().Bool("json", false, "JSON 输出")
	importCmd.Flags().String("prompts-dir", "prompts", "Prompt 文件目录")

	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	bizLine, _ := cmd.Flags().GetString("biz-line")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	promptsDir, _ := cmd.Flags().GetString("prompts-dir")

	report := &ImportReport{
		Total:    len(args),
		Imported: []ImportItem{},
		Skipped:  []SkippedItem{},
	}

	for _, source := range args {
		items, err := collectFiles(source)
		if err != nil {
			report.Skipped = append(report.Skipped, SkippedItem{
				FilePath: source,
				Reason:   err.Error(),
			})
			continue
		}

		for _, item := range items {
			result := processFile(item, promptsDir, bizLine, dryRun)
			switch result.Status {
			case StatusNew:
				report.Imported = append(report.Imported, ImportItem{
					ID:          result.ID,
					FilePath:    item.sourcePath,
					ContentHash: result.ContentHash,
				})
			case StatusSkip:
				report.Skipped = append(report.Skipped, SkippedItem{
					FilePath: item.sourcePath,
					Reason:   result.Reason,
				})
			}
		}
	}

	report.Total = len(report.Imported) + len(report.Skipped)

	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}

	// Human-readable output
	fmt.Println("Import Report:")
	fmt.Printf("  Total files found: %d\n", report.Total)
	fmt.Printf("  Imported:         %d\n", len(report.Imported))
	fmt.Printf("  Skipped (duplicate): %d\n", len(report.Skipped))

	if len(report.Imported) > 0 || len(report.Skipped) > 0 {
		fmt.Println("\n  Details:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, item := range report.Imported {
			relPath, _ := filepath.Rel(".", item.FilePath)
			fmt.Fprintf(w, "  [NEW]     %s  %s\n", item.ID[:8], relPath)
		}
		for _, item := range report.Skipped {
			relPath, _ := filepath.Rel(".", item.FilePath)
			fmt.Fprintf(w, "  [SKIP]    %s  (%s)\n", relPath, item.Reason)
		}
		w.Flush()
	}

	if dryRun {
		fmt.Println("\n[DRY-RUN] 未实际执行更改")
	}

	return nil
}

type fileItem struct {
	sourcePath string
	content    []byte
	hash       string
}

type processResult struct {
	Status       string
	ID           string
	ContentHash  string
	Reason       string
}

const (
	StatusNew  = "new"
	StatusSkip = "skip"
)

func collectFiles(source string) ([]fileItem, error) {
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("cannot access: %w", err)
	}

	if info.IsDir() {
		return collectFromDir(source)
	}

	// Check for glob patterns
	if strings.ContainsAny(source, "*?[") {
		return collectFromGlob(source)
	}

	// Single file
	content, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	hash := sha256.Sum256(content)

	return []fileItem{{
		sourcePath: source,
		content:    content,
		hash:       "sha256:" + hex.EncodeToString(hash[:]),
	}}, nil
}

func collectFromDir(dir string) ([]fileItem, error) {
	var items []fileItem

	// Scan for .md and .txt files
	patterns := []string{"*.md", "*.txt"}
	for _, pattern := range patterns {
		globPath := filepath.Join(dir, pattern)
		matches, err := filepath.Glob(globPath)
		if err != nil {
			continue
		}
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			hash := sha256.Sum256(content)
			items = append(items, fileItem{
				sourcePath: match,
				content:    content,
				hash:       "sha256:" + hex.EncodeToString(hash[:]),
			})
		}
	}

	return items, nil
}

func collectFromGlob(pattern string) ([]fileItem, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	var items []fileItem
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}

		content, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		hash := sha256.Sum256(content)
		items = append(items, fileItem{
			sourcePath: match,
			content:    content,
			hash:       "sha256:" + hex.EncodeToString(hash[:]),
		})
	}

	return items, nil
}

func processFile(item fileItem, promptsDir, bizLine string, dryRun bool) processResult {
	// Check if file path already exists in prompts dir
	destPath := filepath.Join(promptsDir, filepath.Base(item.sourcePath))
	if _, err := os.Stat(destPath); err == nil {
		return processResult{Status: StatusSkip, Reason: "file already exists in prompts dir"}
	}

	// Check if content hash already exists (would need to scan existing files)
	// For simplicity, we only check exact file path conflicts
	// A full implementation would scan promptsDir and compare hashes

	if dryRun {
		id := domain.NewULID()
		return processResult{
			Status:       StatusNew,
			ID:           id,
			ContentHash:  item.hash,
			Reason:       "dry-run",
		}
	}

	// Generate ID and front matter
	id := domain.NewULID()
	fm := &domain.FrontMatter{
		ID:          id,
		Name:        filepath.Base(item.sourcePath),
		Version:     "v1.0.0",
		ContentHash: item.hash,
		State:       "active",
		Tags:        []string{},
	}

	// Serialize and write
	markdown, err := yamlutil.FormatMarkdown(fm, string(item.content))
	if err != nil {
		return processResult{Status: StatusSkip, Reason: fmt.Sprintf("format markdown: %v", err)}
	}

	// Ensure prompts directory exists
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return processResult{Status: StatusSkip, Reason: fmt.Sprintf("create prompts dir: %v", err)}
	}

	// Write file with ULID-based name
	filename := fmt.Sprintf("%s.md", id)
	destPath = filepath.Join(promptsDir, filename)
	if err := os.WriteFile(destPath, []byte(markdown), 0644); err != nil {
		return processResult{Status: StatusSkip, Reason: fmt.Sprintf("write file: %v", err)}
	}

	return processResult{
		Status:      StatusNew,
		ID:          id,
		ContentHash: item.hash,
	}
}

type ImportReport struct {
	Total    int           `json:"total"`
	Imported []ImportItem  `json:"imported"`
	Skipped  []SkippedItem `json:"skipped"`
}

type ImportItem struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	ContentHash string `json:"content_hash"`
}

type SkippedItem struct {
	FilePath string `json:"file_path"`
	Reason   string `json:"reason"`
}