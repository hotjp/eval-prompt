// Package service provides L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"

	"github.com/eval-prompt/internal/domain"
)

// ScanResult represents the result of scanning a source directory for import.
type ScanResult struct {
	// ScannedDirs is the list of directories that were scanned.
	ScannedDirs []string
	// CreatedAssets is the list of asset IDs that were created.
	CreatedAssets []string
	// UpdatedAssets is the list of asset IDs that were updated.
	UpdatedAssets []string
	// Errors contains any errors that occurred during scanning.
	Errors []string
	// Commits contains the commit hashes for each asset that was committed.
	Commits map[string]string
}

// AssetFileManager provides structured read/write access to asset files.
type AssetFileManager interface {
	// Scan scans the source directory for assets to import.
	// It detects asset types, generates asset.yaml files, and moves files
	// to the appropriate type directories (skills/, agents/, prompts/, etc.).
	// Returns a ScanResult with details about what was scanned and created.
	Scan(ctx context.Context, source string) (*ScanResult, error)

	// GetAssetYAML reads and parses an asset.yaml file.
	// Returns the parsed AssetYAML or error if the file doesn't exist.
	GetAssetYAML(ctx context.Context, assetPath string) (*domain.AssetYAML, error)

	// SaveAssetYAML writes an AssetYAML to disk and commits it to Git.
	// The assetPath is relative to the repo root (e.g., "assets/skills/calculator.yaml").
	SaveAssetYAML(ctx context.Context, assetPath string, ay *domain.AssetYAML, commitMsg string) (string, error)

	// MoveAssetFiles moves files from sourceDir to destDir and stages them in Git.
	// The sourceDir and destDir are relative to the repo root.
	MoveAssetFiles(ctx context.Context, sourceDir, destDir string) error

	// GetRepoPath returns the repository path used by this manager.
	GetRepoPath() string

	// GetMainFileContent reads the main file content from an asset.yaml.
	// assetPath is the path to asset.yaml (e.g., "assets/skills/calculator.yaml").
	// Returns the main file content, resolved main path, isExternal flag, and error.
	GetMainFileContent(ctx context.Context, assetPath string) (content string, mainPath string, isExternal bool, err error)

	// WriteMainFileContent writes content to the main file specified in asset.yaml.
	// assetPath is the path to asset.yaml.
	// Returns the new content hash and error.
	WriteMainFileContent(ctx context.Context, assetPath string, content string) (newContentHash string, err error)

	// GetAssetFiles returns the files and external file lists from an asset.yaml.
	// assetPath is the path to asset.yaml.
	// Returns the files list, external list, and error.
	GetAssetFiles(ctx context.Context, assetPath string) (files []FileInfo, external []FileInfo, err error)
}

// FileInfo represents information about a file in an asset.
type FileInfo struct {
	Path string
	Role string
}
