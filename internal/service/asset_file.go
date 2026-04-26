// Package service provides L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"

	"github.com/eval-prompt/internal/domain"
)

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
}
