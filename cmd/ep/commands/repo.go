package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgRepoCmd, nil),
	Short: i18n.T(i18n.MsgRepoCmdShort, nil),
	Long:  i18n.T(i18n.MsgRepoCmdLong, nil),
}

var repoListCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgRepoList, nil),
	Short: i18n.T(i18n.MsgRepoListShort, nil),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoLock, err := lock.ReadLock()
		if err != nil {
			return errors.New(i18n.T(i18n.MsgRepoLockReadFail, pongo2.Context{"error": err.Error()}))
		}

		if len(repoLock.Repos) == 0 {
			fmt.Print(i18n.T(i18n.MsgRepoNoRepos, nil))
			return nil
		}

		current := repoLock.GetCurrent()
		for _, entry := range repoLock.Repos {
			status := lock.ValidatePath(entry.Path)
			marker := "  "
			statusText := ""

			if entry.Path == current {
				marker = i18n.T(i18n.MsgRepoMarkerCurrent, nil)
				switch status {
				case lock.PathValid:
					statusText = i18n.T(i18n.MsgRepoMarkerCurrent, nil)
				case lock.PathNotFound:
					statusText = i18n.T(i18n.MsgRepoMarkerCurrentNotFound, nil)
				case lock.PathNotGit:
					statusText = i18n.T(i18n.MsgRepoMarkerCurrentInvalid, nil)
				}
			} else {
				switch status {
				case lock.PathValid:
					statusText = ""
				case lock.PathNotFound:
					statusText = i18n.T(i18n.MsgRepoNotFound, nil)
				case lock.PathNotGit:
					statusText = i18n.T(i18n.MsgRepoNotGit, nil)
				}
			}

			if statusText != "" {
				fmt.Print(i18n.T(i18n.MsgRepoSwitchComplete, pongo2.Context{"marker": marker, "path": entry.Path, "status": statusText}))
			} else {
				fmt.Print(i18n.T(i18n.MsgRepoSwitchComplete, pongo2.Context{"marker": marker, "path": entry.Path, "status": ""}))
			}
		}

		fmt.Print(i18n.T(i18n.MsgRepoCurrent, pongo2.Context{"current": current}))
		return nil
	},
}

var repoSwitchCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgRepoSwitchCmd, nil),
	Short: i18n.T(i18n.MsgRepoSwitchCmdShort, nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		absPath, err := filepath.Abs(path)
		if err != nil {
			return errors.New(i18n.T(i18n.MsgRepoSwitchLockWriteFail, pongo2.Context{"error": err.Error()}))
		}

		// Reject path traversal
		if strings.Contains(absPath, "..") {
			return errors.New(i18n.T(i18n.MsgRepoSwitchLockWriteFail, nil))
		}

		repoLock, err := lock.ReadLock()
		if err != nil {
			return errors.New(i18n.T(i18n.MsgRepoLockReadFail, pongo2.Context{"error": err.Error()}))
		}

		// Check if path exists in repos
		found := false
		for _, entry := range repoLock.Repos {
			if entry.Path == absPath {
				found = true
				break
			}
		}

		if !found {
			// Path not in lock, ask to add it
			fmt.Print(i18n.T(i18n.MsgRepoSwitchPathNotFound, pongo2.Context{"path": absPath}))
			fmt.Print(i18n.T(i18n.MsgRepoSwitchAskCreate, nil))
			var confirm string
			yes, _ := cmd.Flags().GetBool("yes")
			if yes {
				confirm = "y"
			} else {
				fmt.Scanln(&confirm)
			}
			if strings.ToLower(confirm) != "y" {
				fmt.Print(i18n.T(i18n.MsgRepoSwitchCancel, nil))
				return nil
			}
			repoLock.AddRepo(absPath)
		}

		// Validate path
		status := lock.ValidatePath(absPath)
		switch status {
		case lock.PathValid:
			// Good, switch to it
		case lock.PathNotFound:
			fmt.Print(i18n.T(i18n.MsgRepoSwitchPathNotFound, pongo2.Context{"path": absPath}))
			fmt.Print(i18n.T(i18n.MsgRepoSwitchAskCreate, nil))
			var confirm string
			yes, _ := cmd.Flags().GetBool("yes")
			if yes {
				confirm = "y"
			} else {
				fmt.Scanln(&confirm)
			}
			if strings.ToLower(confirm) != "y" {
				fmt.Print(i18n.T(i18n.MsgRepoSwitchCancel, nil))
				return nil
			}
			// Create directory structure
			dirs := []string{"prompts", ".evals", ".traces"}
			for _, dir := range dirs {
				fullPath := filepath.Join(absPath, dir)
				if err := os.MkdirAll(fullPath, 0755); err != nil {
					return errors.New(i18n.T(i18n.MsgRepoSwitchDirFail, pongo2.Context{"error": err.Error()}))
				}
			}
			// Init git
			bridge := gitbridge.NewBridge()
			if err := bridge.InitRepo(context.Background(), absPath); err != nil {
				fmt.Print(i18n.T(i18n.MsgRepoSwitchGitWarn, pongo2.Context{"error": err.Error()}))
				gitignore := `.eval-prompt/
*.db
.traces/
*.jsonl
`
				os.WriteFile(filepath.Join(absPath, ".gitignore"), []byte(gitignore), 0644)
			}
		case lock.PathNotGit:
			fmt.Print(i18n.T(i18n.MsgRepoSwitchNotGit, pongo2.Context{"path": absPath}))
			fmt.Print(i18n.T(i18n.MsgRepoSwitchAskGit, nil))
			var confirm string
			yes, _ := cmd.Flags().GetBool("yes")
			if yes {
				confirm = "y"
			} else {
				fmt.Scanln(&confirm)
			}
			if strings.ToLower(confirm) != "y" {
				fmt.Print(i18n.T(i18n.MsgRepoSwitchCancel, nil))
				return nil
			}
			bridge := gitbridge.NewBridge()
			if err := bridge.InitRepo(context.Background(), absPath); err != nil {
				return errors.New(i18n.T(i18n.MsgRepoSwitchGitFail, pongo2.Context{"error": err.Error()}))
			}
		}

		// Update lock
		repoLock.SetCurrent(absPath)
		if err := lock.WriteLock(repoLock); err != nil {
			return errors.New(i18n.T(i18n.MsgRepoSwitchLockWriteFail, pongo2.Context{"error": err.Error()}))
		}

		fmt.Print(i18n.T(i18n.MsgRepoSwitchComplete, pongo2.Context{"path": absPath}))

		// Note: config.PromptAssets.RepoPath update requires server restart or config reload
		// For CLI, we just update the lock file. The serve command will read from config.

		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoSwitchCmd)
	repoSwitchCmd.Flags().BoolP("yes", "y", false, i18n.T(i18n.MsgCommonConfirm, nil))
	rootCmd.AddCommand(repoCmd)
}
