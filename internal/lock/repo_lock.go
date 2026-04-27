package lock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// RepoLock is the repository lock file structure.
type RepoLock struct {
	Repos   []RepoEntry `json:"repos"`
	Current string      `json:"current"`
}

// RepoEntry represents a single repository entry in the lock file.
type RepoEntry struct {
	Path   string    `json:"path"`
	InitAt time.Time `json:"init_at"`
}

// PathStatus represents the validation status of a path.
type PathStatus int

const (
	PathValid PathStatus = iota // exists and is a git repository
	PathNotFound                 // directory does not exist
	PathNotGit                   // directory exists but is not a git repository
)

func (s PathStatus) String() string {
	switch s {
	case PathValid:
		return "valid"
	case PathNotFound:
		return "notfound"
	case PathNotGit:
		return "notgit"
	default:
		return "unknown"
	}
}

const lockFileName = "lock.json"

// LockFilePath returns the lock file path (~/.ep/lock.json).
func LockFilePath() (string, error) {
	// Try EP_HOME first
	epHome := os.Getenv("EP_HOME")
	if epHome == "" {
		// Fallback to user home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory and EP_HOME is not set")
		}
		return filepath.Join(homeDir, ".ep", lockFileName), nil
	}
	return filepath.Join(epHome, lockFileName), nil
}

// ReadLock reads the lock file from ~/.ep/lock.json.
// Returns an empty lock if the file does not exist.
func ReadLock() (*RepoLock, error) {
	path, err := LockFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty lock if file doesn't exist
			return &RepoLock{}, nil
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lock RepoLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	return &lock, nil
}

// WriteLock writes the lock file to ~/.ep/lock.json.
func WriteLock(lock *RepoLock) error {
	path, err := LockFilePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	// If directory exists but is not writable (common when root created it),
	// try to fix ownership to allow the current user to write.
	if !isWritable(dir) {
		if err := chownDirToCurrentUser(dir); err != nil {
			return fmt.Errorf("lock directory %s is not writable and could not be fixed: %w", dir, err)
		}
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock: %w", err)
	}

	// Atomic write using temp file + rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename lock file: %w", err)
	}

	return nil
}

// AddRepo adds a repository to the lock or updates its timestamp if it already exists.
func (l *RepoLock) AddRepo(path string) {
	cleanPath := filepath.Clean(path)
	for i, entry := range l.Repos {
		if entry.Path == cleanPath {
			l.Repos[i].InitAt = time.Now()
			return
		}
	}
	l.Repos = append(l.Repos, RepoEntry{
		Path:   cleanPath,
		InitAt: time.Now(),
	})
}

// RemoveRepo removes a repository from the lock.
func (l *RepoLock) RemoveRepo(path string) {
	for i, entry := range l.Repos {
		if entry.Path == path {
			l.Repos = append(l.Repos[:i], l.Repos[i+1:]...)
			// If this was the current, clear current
			if l.Current == path {
				l.Current = ""
			}
			return
		}
	}
}

// SetCurrent sets the current repository path.
func (l *RepoLock) SetCurrent(path string) {
	l.Current = path
}

// GetCurrent returns the current repository path.
func (l *RepoLock) GetCurrent() string {
	return l.Current
}

// GetCurrentIfValid returns the current repository path only if it exists in repos.
// Returns empty string if current is not set or not in the repos list.
func (l *RepoLock) GetCurrentIfValid() string {
	if l.Current == "" {
		return ""
	}
	for _, entry := range l.Repos {
		if entry.Path == l.Current {
			return l.Current
		}
	}
	return ""
}

// isWritable checks if the current user has write permission on the directory.
func isWritable(dir string) bool {
	// Try to write a temporary file
	tmp, err := os.CreateTemp(dir, ".lock-check")
	if err != nil {
		return false
	}
	tmp.Close()
	os.Remove(tmp.Name())
	return true
}

// chownDirToCurrentUser changes directory ownership to the current user.
// It only succeeds if running as root (or with CAP_CHOWN), otherwise it returns an error.
func chownDirToCurrentUser(dir string) error {
	stat, err := os.Stat(dir)
	if err != nil {
		return err
	}
	// Already owned by current user
	if int(stat.Sys().(*syscall.Stat_t).Uid) == os.Getuid() {
		return nil
	}
	// Try to chown - only works if we have CAP_CHOWN (i.e., running as root)
	return os.Chown(dir, os.Getuid(), os.Getgid())
}

// ValidatePath checks if the given path is a valid git repository.
func ValidatePath(path string) PathStatus {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return PathNotFound
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return PathNotFound
		}
		return PathNotFound
	}

	if !info.IsDir() {
		return PathNotFound
	}

	// Use git rev-parse to check if it's a git repository (handles worktrees)
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = absPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return PathNotGit
	}
	return PathValid
}