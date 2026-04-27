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

// AssetFileManager provides structured read/write access to prompt files.
// All frontmatter operations go through this interface to ensure consistent
// read-modify-write cycles with proper locking and conflict detection.
type AssetFileManager interface {
	// GetFrontmatter reads and parses the frontmatter of a prompt file.
	// Returns the parsed frontmatter or error if the file doesn't exist.
	GetFrontmatter(ctx context.Context, id string) (*domain.FrontMatter, error)

	// UpdateFrontmatter reads the existing file, applies the updater function
	// to the frontmatter, then writes the result back and commits to Git.
	// The body is preserved as-is. Returns the commit hash.
	// The updater function should modify the frontmatter in place;
	// return an error to abort the write.
	UpdateFrontmatter(ctx context.Context, id string, updater func(*domain.FrontMatter) error, commitMsg string) (string, error)

	// WriteContent reads the existing file, applies the updater to frontmatter,
	// replaces the body with newBody, then writes back and commits to Git.
	// Returns the commit hash. If the file doesn't exist, creates it with
	// default frontmatter and the given newBody.
	WriteContent(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string, commitMsg string) (string, error)

	// GetBody reads the prompt file, strips frontmatter, and returns only the body.
	// Returns the markdown body or error if the file doesn't exist.
	GetBody(ctx context.Context, id string) (string, error)

	// WriteFileOnly reads the existing file, applies the updater to frontmatter,
	// replaces the body with newBody, then writes back WITHOUT committing to Git.
	// If the file doesn't exist, creates it with default frontmatter and the given newBody.
	WriteFileOnly(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string) error

	// UpdateFrontmatterFileOnly reads existing file, applies updater to frontmatter,
	// writes back WITHOUT committing to Git. Body is preserved.
	UpdateFrontmatterFileOnly(ctx context.Context, id string, updater func(*domain.FrontMatter) error) error

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
}
