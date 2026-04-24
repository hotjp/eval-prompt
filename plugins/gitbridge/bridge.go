// Package gitbridge provides Git operations for prompt assets using go-git.
package gitbridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eval-prompt/internal/service"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// Bridge implements service.GitBridger using go-git.
type Bridge struct {
	repo *git.Repository
}

// NewBridge creates a new Bridge instance.
func NewBridge() *Bridge {
	return &Bridge{}
}

// Ensure Bridge implements GitBridger.
var _ service.GitBridger = (*Bridge)(nil)

// repoPath returns the absolute path to the repository.
func (b *Bridge) repoPath() string {
	if b.repo == nil {
		return ""
	}
	cfg, _ := b.repo.Config()
	if cfg != nil && cfg.Core.Worktree != "" {
		return cfg.Core.Worktree
	}
	return ""
}

// InitRepo initializes a new Git repository at the given path.
func (b *Bridge) InitRepo(ctx context.Context, path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Initialize repository
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	b.repo = repo

	// Write default .gitignore
	if err := writeDefaultGitignore(path); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}

	// Stage and commit .gitignore
	gitignorePath := filepath.Join(path, ".gitignore")
	if _, err := b.StageAndCommit(ctx, gitignorePath, "chore: add default .gitignore"); err != nil {
		return fmt.Errorf("commit .gitignore: %w", err)
	}

	return nil
}

// StageAndCommit stages the file at filePath and creates a commit with the given message.
func (b *Bridge) StageAndCommit(ctx context.Context, filePath, message string) (string, error) {
	if b.repo == nil {
		return "", errors.New("repository not initialized")
	}

	// Get worktree
	worktree, err := b.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("get worktree: %w", err)
	}

	// Stage file
	if err := worktree.AddWithOptions(&git.AddOptions{
		Pathspec: filePath,
	}); err != nil {
		return "", fmt.Errorf("stage file %s: %w", filePath, err)
	}

	// Create commit
	commit, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "eval-prompt",
			Email: "agent@eval-prompt.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("create commit: %w", err)
	}

	return commit.String(), nil
}

// Diff returns the diff output between two commits (commit1 and commit2).
func (b *Bridge) Diff(ctx context.Context, commit1, commit2 string) (string, error) {
	if b.repo == nil {
		return "", errors.New("repository not initialized")
	}

	c1, err := b.repo.ResolveRevision(plumbing.Revision(commit1))
	if err != nil {
		return "", fmt.Errorf("resolve commit1 %s: %w", commit1, err)
	}
	c2, err := b.repo.ResolveRevision(plumbing.Revision(commit2))
	if err != nil {
		return "", fmt.Errorf("resolve commit2 %s: %w", commit2, err)
	}

	fromCommit, err := b.repo.CommitObject(c1)
	if err != nil {
		return "", fmt.Errorf("get from commit: %w", err)
	}
	toCommit, err := b.repo.CommitObject(c2)
	if err != nil {
		return "", fmt.Errorf("get to commit: %w", err)
	}

	var buf bytes.Buffer
	patch, err := fromCommit.Patch(toCommit)
	if err != nil {
		return "", fmt.Errorf("generate patch: %w", err)
	}
	patch.Encode(&buf)

	return buf.String(), nil
}

// Log returns the commit log for a file, limited to the specified number of entries.
func (b *Bridge) Log(ctx context.Context, filePath string, limit int) ([]service.CommitInfo, error) {
	if b.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	// Get commits for the file
	fileLog, err := b.repo.Log(&git.LogOptions{
		Pathspec: filePath,
		Order:    git.LogOrderDFSPost,
	})
	if err != nil {
		return nil, fmt.Errorf("get file log: %w", err)
	}
	defer fileLog.Close()

	var commits []service.CommitInfo
	count := 0
	for {
		c, err := fileLog.Next()
		if err != nil {
			break
		}
		commits = append(commits, service.CommitInfo{
			Hash:      c.Hash.String(),
			ShortHash: c.Hash.String()[:7],
			Subject:   c.Message,
			Body:      c.Message,
			Author:    c.Author.Name,
			Timestamp: c.Author.When,
		})
		count++
		if limit > 0 && count >= limit {
			break
		}
	}

	return commits, nil
}

// Status returns the current working tree status: added, modified, and deleted files.
func (b *Bridge) Status(ctx context.Context) (added, modified, deleted []string, err error) {
	if b.repo == nil {
		return nil, nil, nil, errors.New("repository not initialized")
	}

	worktree, err := b.repo.Worktree()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get status: %w", err)
	}

	for path, fs := range status {
		switch fs {
		case git.StatusAdded:
			added = append(added, path)
		case git.StatusModified:
			modified = append(modified, path)
		case git.StatusDeleted:
			deleted = append(deleted, path)
		}
	}

	return added, modified, deleted, nil
}

// Open opens an existing Git repository at the given path.
func (b *Bridge) Open(path string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}
	b.repo = repo
	return nil
}
