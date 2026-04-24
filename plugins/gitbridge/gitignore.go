package gitbridge

import (
	"fmt"
	"os"
	"path/filepath"
)

// defaultGitignoreContents is the default .gitignore content for prompt asset repositories.
const defaultGitignoreContents = `# eval-prompt generated
# SQLite database
*.db
*.db-wal
*.db-shm

# Trace logs
.traces/

# Eval outputs
.evals/**/*.yaml
.evals/**/*.json

# IDE and editor
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Temporary files
*.tmp
*.temp
*.log
`

// writeDefaultGitignore writes the default .gitignore file to the repository root.
func writeDefaultGitignore(repoPath string) error {
	gitignorePath := filepath.Join(repoPath, ".gitignore")

	// Check if .gitignore already exists
	if _, err := os.Stat(gitignorePath); err == nil {
		// File exists, do not overwrite
		return nil
	}

	if err := os.WriteFile(gitignorePath, []byte(defaultGitignoreContents), 0644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}

	return nil
}
