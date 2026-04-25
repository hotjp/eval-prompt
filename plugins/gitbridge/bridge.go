// Package gitbridge provides Git operations for prompt assets using system git.
package gitbridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eval-prompt/internal/service"
)

// Bridge implements service.GitBridger using system git.
type Bridge struct {
	repoPath string
}

// NewBridge creates a new Bridge instance.
func NewBridge() *Bridge {
	return &Bridge{}
}

// SetPath sets the repository path for the Bridge.
func (b *Bridge) SetPath(path string) {
	b.repoPath = path
}

// Ensure Bridge implements GitBridger.
var _ service.GitBridger = (*Bridge)(nil)

// runGit executes a git command and returns the output.
func (b *Bridge) runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = b.repoPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), stderr.String(), err)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// InitRepo initializes a new Git repository at the given path.
// If the path is already a git repository, it returns nil (idempotent).
func (b *Bridge) InitRepo(ctx context.Context, path string) error {
	// Validate path
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	b.repoPath = path

	// Check if already a git repository
	if _, err := b.runGit(ctx, "rev-parse", "--git-dir"); err == nil {
		// Already a git repo, nothing to do
		return nil
	}

	// Initialize repository
	if _, err := b.runGit(ctx, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Create standard directory structure
	dirs := []string{"prompts", ".evals", ".traces"}
	for _, dir := range dirs {
		fullPath := filepath.Join(path, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Write default .gitignore
	if err := writeDefaultGitignore(path); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}

	// Stage and commit .gitignore
	if _, err := b.StageAndCommit(ctx, ".gitignore", "chore: add default .gitignore"); err != nil {
		return fmt.Errorf("commit .gitignore: %w", err)
	}

	return nil
}

// StageAndCommit stages the file at filePath and creates a commit with the given message.
func (b *Bridge) StageAndCommit(ctx context.Context, filePath, message string) (string, error) {
	if b.repoPath == "" {
		return "", errors.New("repository not initialized")
	}

	// Stage file
	if _, err := b.runGit(ctx, "add", filePath); err != nil {
		return "", fmt.Errorf("stage file %s: %w", filePath, err)
	}

	// Create commit
	hash, err := b.runGit(ctx, "commit", "-m", message, "--author=eval-prompt <agent@eval-prompt.local>")
	if err != nil {
		return "", fmt.Errorf("create commit: %w", err)
	}

	return strings.TrimSpace(hash), nil
}

// Diff returns the diff output between two commits (commit1 and commit2).
func (b *Bridge) Diff(ctx context.Context, commit1, commit2 string) (string, error) {
	if b.repoPath == "" {
		return "", errors.New("repository not initialized")
	}

	out, err := b.runGit(ctx, "diff", commit1, commit2)
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}

	return out, nil
}

// Log returns the commit log for a file, limited to the specified number of entries.
func (b *Bridge) Log(ctx context.Context, filePath string, limit int) ([]service.CommitInfo, error) {
	if b.repoPath == "" {
		return nil, errors.New("repository not initialized")
	}

	args := []string{"log", "--format=%H|%s|%an|%ad", "--date=iso"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}
	args = append(args, "--", filePath)

	out, err := b.runGit(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	var commits []service.CommitInfo
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) >= 4 {
			commits = append(commits, service.CommitInfo{
				Hash:      parts[0],
				ShortHash: parts[0][:7],
				Subject:   parts[1],
				Body:      parts[1],
				Author:    parts[2],
				Timestamp: parseGitTime(parts[3]),
			})
		}
	}

	return commits, nil
}

// parseGitTime parses git's iso format time.
func parseGitTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05 -0700", s)
	return t
}

// Status returns the current working tree status: added, modified, and deleted files.
func (b *Bridge) Status(ctx context.Context) (added, modified, deleted []string, err error) {
	if b.repoPath == "" {
		return nil, nil, nil, errors.New("repository not initialized")
	}

	// Use -u to show individual untracked files (not just directories)
	out, err := b.runGit(ctx, "status", "--porcelain", "-u")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("git status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		path := strings.TrimSpace(line[3:])
		if strings.Contains(status, "A") || status == "A" {
			added = append(added, path)
		} else if strings.Contains(status, "M") || status == "M" {
			modified = append(modified, path)
		} else if strings.Contains(status, "D") || status == "D" {
			deleted = append(deleted, path)
		} else if status == "??" {
			// Untracked files - treat as added
			added = append(added, path)
		}
	}

	return added, modified, deleted, nil
}

// Pull fetches and merges changes from the remote repository.
func (b *Bridge) Pull(ctx context.Context) error {
	if b.repoPath == "" {
		return errors.New("repository not initialized")
	}

	// First get the current branch
	branch, err := b.runGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	branch = strings.TrimSpace(branch)

	// Pull from origin on the current branch
	if _, err := b.runGit(ctx, "pull", "origin", branch); err != nil {
		return fmt.Errorf("git pull origin %s: %w", branch, err)
	}

	return nil
}

// Open opens an existing Git repository at the given path.
func (b *Bridge) Open(path string) error {
	// Verify it's a git repo
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return fmt.Errorf("open repo: %w", err)
	}
	b.repoPath = path
	return nil
}

// RepoPath returns the root path of the Git repository.
func (b *Bridge) RepoPath() string {
	return b.repoPath
}

// ReInit reinitializes the Bridge with a new repository path.
// It validates the path and updates the internal repoPath.
func (b *Bridge) ReInit(ctx context.Context, path string) error {
	if path == "" {
		return nil
	}

	cleanPath := path
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return err
		}
		cleanPath = absPath
	}

	// Verify it's a valid git repo
	if _, err := os.Stat(filepath.Join(cleanPath, ".git")); err != nil {
		return err
	}

	b.repoPath = cleanPath
	return nil
}
