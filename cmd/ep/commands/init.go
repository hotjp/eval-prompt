package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <path>",
	Short: i18n.T(i18n.MsgInitTitle, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		fmt.Print(i18n.T(i18n.MsgInitStart, pongo2.Context{"path": path}))

		// Git init using go-git bridge
		bridge := gitbridge.NewBridge()
		if err := bridge.InitRepo(context.Background(), path); err != nil {
			fmt.Print(i18n.T(i18n.MsgInitGitWarn, pongo2.Context{"error": err.Error()}))
			// Fall back to manual gitignore
			gitignore := `.eval-prompt/
*.db
.traces/
*.jsonl
`
			os.WriteFile(filepath.Join(path, ".gitignore"), []byte(gitignore), 0644)
		} else {
			fmt.Print(i18n.T(i18n.MsgInitGitComplete, nil))
		}

		// Update repo lock file
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Print(i18n.T(i18n.MsgInitLockReadWarn, pongo2.Context{"error": err.Error()}))
		} else {
			repoLock, err := lock.ReadLock()
			if err != nil {
				fmt.Print(i18n.T(i18n.MsgInitLockReadWarn, pongo2.Context{"error": err.Error()}))
			} else {
				repoLock.AddRepo(absPath)
				repoLock.SetCurrent(absPath)
				if err := lock.WriteLock(repoLock); err != nil {
					fmt.Print(i18n.T(i18n.MsgInitLockWriteWarn, pongo2.Context{"error": err.Error()}))
				} else {
					fmt.Print(i18n.T(i18n.MsgInitLockAdded, nil))
				}
			}
		}

		// Create SQLite database path
		dbDir := filepath.Join(os.Getenv("HOME"), ".eval-prompt")
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf(i18n.T(i18n.MsgInitDBDirFail, pongo2.Context{"error": err.Error()}))
		}
		dbPath := filepath.Join(dbDir, "index.db")

		fmt.Print(i18n.T(i18n.MsgInitComplete, nil))
		fmt.Print(i18n.T(i18n.MsgInitGitPath, pongo2.Context{"path": path}))
		fmt.Print(i18n.T(i18n.MsgInitSQLitePath, pongo2.Context{"path": dbPath}))
		fmt.Print(i18n.T(i18n.MsgInitServeHint, nil))
		return nil
	},
}
