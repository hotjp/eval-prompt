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
func (b *Bridge) InitRepo(ctx context.Context, path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	b.repoPath = path

	// Initialize repository
	if _, err := b.runGit(ctx, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
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

	out, err := b.runGit(ctx, "status", "--porcelain")
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
		}
	}

	return added, modified, deleted, nil
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
