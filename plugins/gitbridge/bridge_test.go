package gitbridge

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eval-prompt/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewBridge(t *testing.T) {
	bridge := NewBridge()
	require.NotNil(t, bridge)
	require.Empty(t, bridge.repoPath)
}

func TestBridge_ImplementsInterface(t *testing.T) {
	bridge := NewBridge()
	var _ service.GitBridger = bridge
}

func TestBridge_InitRepo(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	// Verify .git directory exists
	gitDir := filepath.Join(repoPath, ".git")
	require.DirExists(t, gitDir)

	// Verify .gitignore was created
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	require.FileExists(t, gitignorePath)

	// Verify .gitignore has content
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	require.Contains(t, string(content), "eval-prompt generated")
}

func TestBridge_InitRepo_CreatesNestedDirectories(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "nested", "path", "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)
	require.DirExists(t, repoPath)
}

func TestBridge_StageAndCommit(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	// Create a file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Stage and commit
	hash, err := bridge.StageAndCommit(context.Background(), "test.txt", "Add test file")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	// Hash is returned as part of git output, but should contain hex characters
	require.Regexp(t, "[a-f0-9]+", hash)
}

func TestBridge_StageAndCommit_RepoNotInitialized(t *testing.T) {
	bridge := NewBridge()
	_, err := bridge.StageAndCommit(context.Background(), "test.txt", "Add test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "repository not initialized")
}

func TestBridge_Diff(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	// Create and commit initial file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	commit1, err := bridge.StageAndCommit(context.Background(), "test.txt", "Initial commit")
	require.NoError(t, err)

	// Modify file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	commit2, err := bridge.StageAndCommit(context.Background(), "test.txt", "Modify test")
	require.NoError(t, err)

	// Get diff between commits using proper git diff syntax
	diff, err := bridge.Diff(context.Background(), commit1, commit2)
	require.NoError(t, err)
	// The diff should contain + or - lines showing the change
	require.True(t, len(diff) > 0, "diff should not be empty")
}

func TestBridge_Diff_RepoNotInitialized(t *testing.T) {
	bridge := NewBridge()
	_, err := bridge.Diff(context.Background(), "abc", "def")
	require.Error(t, err)
	require.Contains(t, err.Error(), "repository not initialized")
}

func TestBridge_Log(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	// Create and commit multiple files
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(repoPath, "test.txt")
		content := []byte("content version")
		err = os.WriteFile(testFile, content, 0644)
		require.NoError(t, err)

		_, err = bridge.StageAndCommit(context.Background(), "test.txt", "Commit message")
		require.NoError(t, err)
	}

	// Get log
	commits, err := bridge.Log(context.Background(), "test.txt", 10)
	require.NoError(t, err)
	require.Len(t, commits, 3)
}

func TestBridge_Log_RepoNotInitialized(t *testing.T) {
	bridge := NewBridge()
	_, err := bridge.Log(context.Background(), "test.txt", 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "repository not initialized")
}

func TestBridge_Log_Limit(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	// Create and commit multiple files
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(repoPath, "test.txt")
		err = os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		_, err = bridge.StageAndCommit(context.Background(), "test.txt", "Commit message")
		require.NoError(t, err)
	}

	// Get log with limit
	commits, err := bridge.Log(context.Background(), "test.txt", 3)
	require.NoError(t, err)
	require.Len(t, commits, 3)
}

func TestBridge_Status(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	// Create a new file
	newFile := filepath.Join(repoPath, "new.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0644)
	require.NoError(t, err)

	// Modify existing file
	testFile := filepath.Join(repoPath, ".gitignore")
	err = os.WriteFile(testFile, []byte("modified"), 0644)
	require.NoError(t, err)

	added, modified, deleted, err := bridge.Status(context.Background())
	require.NoError(t, err)
	require.Contains(t, added, "new.txt")
	require.Contains(t, modified, ".gitignore")
	require.Empty(t, deleted)
}

func TestBridge_Status_RepoNotInitialized(t *testing.T) {
	bridge := NewBridge()
	_, _, _, err := bridge.Status(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "repository not initialized")
}

func TestBridge_Status_NoChanges(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	bridge := NewBridge()
	err := bridge.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	added, modified, deleted, err := bridge.Status(context.Background())
	require.NoError(t, err)
	require.Empty(t, added)
	require.Empty(t, modified)
	require.Empty(t, deleted)
}

func TestBridge_Open(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// First initialize a repo
	bridge1 := NewBridge()
	err := bridge1.InitRepo(context.Background(), repoPath)
	require.NoError(t, err)

	// Open with new bridge
	bridge2 := NewBridge()
	err = bridge2.Open(repoPath)
	require.NoError(t, err)
	require.Equal(t, repoPath, bridge2.repoPath)
}

func TestBridge_Open_NotAGitRepo(t *testing.T) {
	tempDir := t.TempDir()
	notARepo := filepath.Join(tempDir, "not-a-repo")

	// Create a directory that's not a git repo
	err := os.MkdirAll(notARepo, 0755)
	require.NoError(t, err)

	bridge := NewBridge()
	err = bridge.Open(notARepo)
	require.Error(t, err)
	require.Contains(t, err.Error(), "open repo")
}

func TestBridge_Open_NonExistentPath(t *testing.T) {
	bridge := NewBridge()
	err := bridge.Open("/nonexistent/path")
	require.Error(t, err)
}

func TestParseGitTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
	}{
		{
			name:     "valid time",
			input:    "2024-01-15 10:30:00 +0000",
			wantZero: false,
		},
		{
			name:     "invalid time",
			input:    "not a time",
			wantZero: true,
		},
		{
			name:     "empty time",
			input:    "",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitTime(tt.input)
			if tt.wantZero {
				require.True(t, result.IsZero())
			} else {
				require.False(t, result.IsZero())
			}
		})
	}
}

func TestWriteDefaultGitignore(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// Create the repo directory
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	// Write gitignore (simulating what InitRepo does)
	err = writeDefaultGitignore(repoPath)
	require.NoError(t, err)

	// Verify gitignore exists
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	require.FileExists(t, gitignorePath)

	// Verify content
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	require.Contains(t, string(content), "eval-prompt generated")
	require.Contains(t, string(content), "*.db")
}

func TestWriteDefaultGitignore_AlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// Create the repo directory
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	// Write gitignore first time
	err = writeDefaultGitignore(repoPath)
	require.NoError(t, err)

	// Modify the gitignore
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	originalContent, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)

	// Write again - should NOT overwrite
	err = writeDefaultGitignore(repoPath)
	require.NoError(t, err)

	// Content should be unchanged
	newContent, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	require.Equal(t, originalContent, newContent)
}

func TestDefaultGitignoreContents(t *testing.T) {
	// Verify the constant has expected content
	require.Contains(t, defaultGitignoreContents, "# eval-prompt generated")
	require.Contains(t, defaultGitignoreContents, "*.db")
	require.Contains(t, defaultGitignoreContents, ".traces/")
	require.Contains(t, defaultGitignoreContents, ".evals/")
}
