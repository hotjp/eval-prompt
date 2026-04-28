package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/eval-prompt/plugins/search"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: i18n.T(i18n.MsgSyncCmdShort, nil),
}

func init() {
	syncCmd.AddCommand(syncReconcileCmd)
	syncCmd.AddCommand(syncExportCmd)
}

// resolveWorkDir resolves the working directory from --repo or --dir flag.
// If --repo is specified, it switches to that repo in lock.json first.
// If --dir is specified, uses that directory directly without touching lock.
// If neither is specified, uses the current repo from lock.json.
// Returns an error if no repo is set and neither flag is provided.
func resolveWorkDir(cmd *cobra.Command) (string, error) {
	repoPath, _ := cmd.Flags().GetString("repo")
	dirPath, _ := cmd.Flags().GetString("dir")

	if repoPath != "" {
		// --repo specified: add to lock and switch to it
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return "", errors.New(i18n.T(i18n.MsgSyncResolvePathFailed, pongo2.Context{"error": err.Error()}))
		}

		repoLock, err := lock.ReadLock()
		if err != nil {
			return "", errors.New(i18n.T(i18n.MsgSyncReadLockFailed, pongo2.Context{"error": err.Error()}))
		}

		repoLock.AddRepo(absPath)
		repoLock.SetCurrent(absPath)
		if err := lock.WriteLock(repoLock); err != nil {
			return "", errors.New(i18n.T(i18n.MsgSyncWriteLockFailed, pongo2.Context{"error": err.Error()}))
		}

		fmt.Println(i18n.T(i18n.MsgSyncRepoSwitch, pongo2.Context{"path": absPath}))
		return absPath, nil
	}

	if dirPath != "" {
		// --dir specified: use it directly
		return dirPath, nil
	}

	// Neither --repo nor --dir specified: require current repo from lock
	repoLock, err := lock.ReadLock()
	if err != nil {
		return "", errors.New(i18n.T(i18n.MsgSyncReadLockFailed, pongo2.Context{"error": err.Error()}))
	}

	current := repoLock.GetCurrent()
	if current == "" {
		return "", errors.New(i18n.T(i18n.MsgSyncNoRepoSet, nil))
	}
	return current, nil
}

var syncReconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: i18n.T(i18n.MsgSyncReconcileShort, nil),
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := resolveWorkDir(cmd)
		if err != nil {
			return err
		}

		indexer := search.Default()
		indexer.SetPersistDir(filepath.Join(wd, ".eval-prompt"))
		if err := indexer.Load(); err != nil {
			fmt.Println(i18n.T(i18n.MsgSyncReconcileWarning, pongo2.Context{"error": err.Error()}))
		}
		gitBridge := gitbridge.NewBridge()
		if err := gitBridge.Open(wd); err != nil {
			return errors.New(i18n.T(i18n.MsgSyncOpenRepoFailed, pongo2.Context{"error": err.Error()}))
		}
		indexer.SetGitBridge(gitBridge)

		syncService := service.NewSyncService(indexer, gitBridge)
		report, err := syncService.Reconcile(context.Background())
		if err != nil {
			return errors.New(i18n.T(i18n.MsgSyncReconcileFailed, pongo2.Context{"error": err.Error()}))
		}

		fmt.Println(i18n.T(i18n.MsgSyncReconcileDone, pongo2.Context{"count": report.Added + report.Updated + report.Deleted}))
		fmt.Println(i18n.T(i18n.MsgSyncAdded, pongo2.Context{"count": report.Added}))
		fmt.Println(i18n.T(i18n.MsgSyncUpdated, pongo2.Context{"count": report.Updated}))
		fmt.Println(i18n.T(i18n.MsgSyncDeleted, pongo2.Context{"count": report.Deleted}))
		if len(report.Errors) > 0 {
			fmt.Println(i18n.T(i18n.MsgSyncError, pongo2.Context{"error": fmt.Sprintf("%v", report.Errors)}))
		}
		return nil
	},
}

func init() {
	syncReconcileCmd.Flags().String("repo", "", i18n.T(i18n.MsgFlagRepo, nil))
	syncReconcileCmd.Flags().String("dir", "", i18n.T(i18n.MsgFlagDir, nil))
}

var syncExportCmd = &cobra.Command{
	Use:   "export",
	Short: i18n.T(i18n.MsgSyncExportShort, nil),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		wd, err := resolveWorkDir(cmd)
		if err != nil {
			return err
		}

		indexer := search.Default()
		gitBridge := gitbridge.NewBridge()
		if err := gitBridge.Open(wd); err != nil {
			return errors.New(i18n.T(i18n.MsgSyncOpenRepoFailed, pongo2.Context{"error": err.Error()}))
		}
		indexer.SetGitBridge(gitBridge)

		syncService := service.NewSyncService(indexer, gitBridge)

		// Run reconcile first to populate the index from git
		if _, err := syncService.Reconcile(context.Background()); err != nil {
			return errors.New(i18n.T(i18n.MsgSyncReconcileFailed, pongo2.Context{"error": err.Error()}))
		}

		data, err := syncService.Export(context.Background(), format)
		if err != nil {
			return errors.New(i18n.T(i18n.MsgSyncExportFailed, pongo2.Context{"error": err.Error()}))
		}

		if output != "" {
			return os.WriteFile(output, data, 0644)
		}

		fmt.Print(string(data))
		return nil
	},
}

func init() {
	syncExportCmd.Flags().String("format", "json", i18n.T(i18n.MsgFlagFormat, nil))
	syncExportCmd.Flags().String("output", "", i18n.T(i18n.MsgFlagOutput, nil))
	syncExportCmd.Flags().String("repo", "", i18n.T(i18n.MsgFlagRepo, nil))
	syncExportCmd.Flags().String("dir", "", i18n.T(i18n.MsgFlagDir, nil))
}
