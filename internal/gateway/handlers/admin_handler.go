// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/gitbridge"
)

// AdminHandler handles admin API endpoints.
type AdminHandler struct {
	logger         *slog.Logger
	startTime      time.Time
	restartCount   atomic.Int64
	lastReloadTime atomic.Value // time.Time
	pid            int
	config         *config.Config
	restartFunc    func()
	indexer       service.AssetIndexer
	gitBridge     service.GitBridger
	configManager service.ConfigManager
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(logger *slog.Logger, cfg *config.Config, restartFunc func(), indexer service.AssetIndexer, gitBridge service.GitBridger, configManager service.ConfigManager) *AdminHandler {
	h := &AdminHandler{
		logger:       logger,
		startTime:    time.Now(),
		pid:          os.Getpid(),
		config:       cfg,
		restartFunc:  restartFunc,
		indexer:     indexer,
		gitBridge:   gitBridge,
		configManager: configManager,
	}
	h.lastReloadTime.Store(time.Time{})

	if repoLock, err := lock.ReadLock(); err == nil {
		if current := repoLock.GetCurrent(); current != "" {
			cfg.PromptAssets.RepoPath = current
		}
	}

	if cfg.PromptAssets.RepoPath != "" {
		repoPath := cfg.PromptAssets.RepoPath
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(".", repoPath)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := h.runGitWithContext(ctx, []string{"rev-parse", "--git-dir"}, repoPath)
		if err != nil {
			h.logger.Warn("repo_path is not a valid git repository", "repo_path", cfg.PromptAssets.RepoPath, "error", err, "layer", "L5")
		}
	}

	return h
}

// StatusResponse represents the server status response.
type StatusResponse struct {
	UptimeSeconds int64     `json:"uptime_seconds"`
	Version       string    `json:"version"`
	PID           int       `json:"pid"`
	MemoryMB      int64     `json:"memory_mb"`
	LastReload    string    `json:"last_reload,omitempty"`
	RestartCount  int64     `json:"restart_count"`
}

// ReloadResponse represents the reload response.
type ReloadResponse struct {
	Status    string `json:"status"`
	Message  string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// RestartResponse represents the restart response.
type RestartResponse struct {
	Status    string `json:"status"`
	Message  string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// GitInfoResponse represents git repository info.
type GitInfoResponse struct {
	Branch      string `json:"branch"`
	Dirty       bool   `json:"dirty"`
	ShortCommit string `json:"short_commit"`
	Remote      string `json:"remote"`
}

// GetGitInfo handles GET /api/v1/admin/git-info.
func (h *AdminHandler) GetGitInfo(w http.ResponseWriter, r *http.Request) {
	repoPath := h.config.PromptAssets.RepoPath
	if repoPath == "" {
		repoPath = "."
	}

	var branch, shortCommit, remote string
	var dirty bool

	out, err := h.runGit([]string{"rev-parse", "--abbrev-ref", "HEAD"}, repoPath)
	if err == nil {
		branch = strings.TrimSpace(out)
	}

	out, err = h.runGit([]string{"rev-parse", "--short", "HEAD"}, repoPath)
	if err == nil {
		shortCommit = strings.TrimSpace(out)
	}

	out, err = h.runGit([]string{"status", "--porcelain"}, repoPath)
	if err == nil && strings.TrimSpace(out) != "" {
		dirty = true
	}

	out, err = h.runGit([]string{"remote", "get-url", "origin"}, repoPath)
	if err == nil {
		remote = strings.TrimSpace(out)
		if strings.HasPrefix(remote, "git@github.com:") {
			remote = strings.TrimPrefix(remote, "git@github.com:")
			remote = strings.TrimSuffix(remote, ".git")
		} else if strings.HasPrefix(remote, "https://github.com/") {
			remote = strings.TrimPrefix(remote, "https://github.com/")
			remote = strings.TrimSuffix(remote, ".git")
		}
	}

	h.writeJSON(w, http.StatusOK, GitInfoResponse{
		Branch:      branch,
		Dirty:       dirty,
		ShortCommit: shortCommit,
		Remote:      remote,
	})
}

// RepoConfigResponse represents the repo_path configuration response.
type RepoConfigResponse struct {
	RepoPath  string `json:"repo_path"`
	AssetsDir string `json:"assets_dir"`
	EvalsDir  string `json:"evals_dir"`
}

// GetRepoConfig handles GET /api/v1/admin/repo-config.
func (h *AdminHandler) GetRepoConfig(w http.ResponseWriter, r *http.Request) {
	if h.config == nil {
		h.writeError(w, http.StatusServiceUnavailable, "config not available")
		return
	}

	resp := RepoConfigResponse{
		RepoPath:  h.config.PromptAssets.RepoPath,
		AssetsDir: h.config.PromptAssets.AssetsDir,
		EvalsDir:  h.config.PromptAssets.EvalsDir,
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// FirstUseResponse represents the first-use check response.
type FirstUseResponse struct {
	FirstUse bool `json:"first_use"`
}

// GetFirstUse handles GET /api/v1/admin/first-use.
func (h *AdminHandler) GetFirstUse(w http.ResponseWriter, r *http.Request) {
	repoLock, err := lock.ReadLock()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read repo lock: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, FirstUseResponse{
		FirstUse: len(repoLock.Repos) == 0,
	})
}

// RepoStatusResponse represents the repository status response.
type RepoStatusResponse struct {
	Current    *RepoStatus `json:"current,omitempty"`
	Repos      []RepoInfo  `json:"repos"`
	IsFirstUse bool        `json:"is_first_use"`
}

// RepoStatus represents the current repository status.
type RepoStatus struct {
	Path        string `json:"path"`
	Valid       bool   `json:"valid"`
	Branch      string `json:"branch,omitempty"`
	Dirty       bool   `json:"dirty"`
	ShortCommit string `json:"short_commit,omitempty"`
	Error       string `json:"error,omitempty"`
	OutsideHome bool   `json:"outside_home,omitempty"`
}

// GetRepoStatus handles GET /api/v1/admin/repo-status.
func (h *AdminHandler) GetRepoStatus(w http.ResponseWriter, r *http.Request) {
	repoLock, err := lock.ReadLock()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read repo lock: %v", err)
		return
	}

	knownPaths := make(map[string]bool)
	for _, entry := range repoLock.Repos {
		knownPaths[entry.Path] = true
	}

	configRepoPath := h.config.PromptAssets.RepoPath
	if configRepoPath != "" {
		knownPaths[configRepoPath] = true
	}

	reposMap := make(map[string]RepoInfo)
	for _, entry := range repoLock.Repos {
		absPath, _ := filepath.Abs(entry.Path)
		if _, ok := reposMap[absPath]; !ok {
			reposMap[absPath] = RepoInfo{
				Path:   entry.Path,
				Status: lock.ValidatePath(entry.Path).String(),
			}
		}
	}

	repos := make([]RepoInfo, 0, len(reposMap))
	for _, info := range reposMap {
		repos = append(repos, info)
	}
	sortSlice(repos)

	currentRepoPath := configRepoPath
	if currentRepoPath == "" {
		currentRepoPath = repoLock.GetCurrent()
	}

	var current *RepoStatus
	if currentRepoPath != "" {
		pathStatus := lock.ValidatePath(currentRepoPath)
		if pathStatus == lock.PathValid {
			branch, shortCommit, dirty := "", "", false

			out, err := h.runGit([]string{"rev-parse", "--abbrev-ref", "HEAD"}, currentRepoPath)
			if err == nil {
				branch = strings.TrimSpace(out)
			}
			out, err = h.runGit([]string{"rev-parse", "--short", "HEAD"}, currentRepoPath)
			if err == nil {
				shortCommit = strings.TrimSpace(out)
			}
			out, err = h.runGit([]string{"status", "--porcelain"}, currentRepoPath)
			if err == nil && strings.TrimSpace(out) != "" {
				dirty = true
			}

			current = &RepoStatus{
				Path:        currentRepoPath,
				Valid:       true,
				Branch:      branch,
				Dirty:       dirty,
				ShortCommit: shortCommit,
				OutsideHome: isOutsideHome(currentRepoPath),
			}
		} else {
			errMsg := "not a git repository"
			if pathStatus == lock.PathNotFound {
				errMsg = "path not found"
			}
			current = &RepoStatus{
				Path:        currentRepoPath,
				Valid:       false,
				Error:       errMsg,
				OutsideHome: isOutsideHome(currentRepoPath),
			}
		}
	}

	isFirstUse := len(repoLock.Repos) == 0 && (configRepoPath == "" || !filepath.IsAbs(configRepoPath))

	h.writeJSON(w, http.StatusOK, RepoStatusResponse{
		Current:    current,
		Repos:      repos,
		IsFirstUse: isFirstUse,
	})
}

// RepoConfigUpdateRequest represents the request to update repo_path config.
type RepoConfigUpdateRequest struct {
	RepoPath  string `json:"repo_path"`
	AssetsDir string `json:"assets_dir"`
	EvalsDir  string `json:"evals_dir"`
}

// UpdateRepoConfig handles PUT /api/v1/admin/repo-config.
func (h *AdminHandler) UpdateRepoConfig(w http.ResponseWriter, r *http.Request) {
	if h.config == nil {
		h.writeError(w, http.StatusServiceUnavailable, "config not available")
		return
	}

	var req RepoConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.RepoPath == "" {
		h.writeError(w, http.StatusBadRequest, "repo_path cannot be empty")
		return
	}

	h.config.PromptAssets.RepoPath = req.RepoPath
	h.config.PromptAssets.AssetsDir = req.AssetsDir
	h.config.PromptAssets.EvalsDir = req.EvalsDir

	if err := h.config.Save(); err != nil {
		h.logger.Error("failed to save config", "error", err, "layer", "L5")
	}

	repoLock, err := lock.ReadLock()
	if err == nil {
		absPath := req.RepoPath
		if !filepath.IsAbs(absPath) {
			absPath, _ = filepath.Abs(absPath)
		}
		repoLock.AddRepo(absPath)
		repoLock.SetCurrent(absPath)
		if err := lock.WriteLock(repoLock); err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to sync lock file: %v", err)
			return
		}
	}

	if h.configManager != nil {
		h.configManager.Notify(r.Context(), "repo", []string{"repo_path"})
	}

	h.writeJSON(w, http.StatusOK, RepoConfigResponse{
		RepoPath:  h.config.PromptAssets.RepoPath,
		AssetsDir: h.config.PromptAssets.AssetsDir,
		EvalsDir:  h.config.PromptAssets.EvalsDir,
	})
}

// ConfigUpdateRequest represents a request to update config fields.
type ConfigUpdateRequest map[string]interface{}

// SaveConfig handles PUT /api/v1/admin/config
func (h *AdminHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	if h.config == nil {
		h.writeError(w, http.StatusServiceUnavailable, "config not available")
		return
	}

	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	// Handle lang field
	if lang, ok := req["lang"]; ok {
		if langStr, ok := lang.(string); ok {
			h.config.Lang = langStr
			h.logger.Info("language updated", "lang", langStr, "layer", "L5")
		}
	}

	if err := h.config.Save(); err != nil {
		h.logger.Error("failed to save config", "error", err, "layer", "L5")
		h.writeError(w, http.StatusInternalServerError, "failed to save config: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RepoInfo represents information about a single repository.
type RepoInfo struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// RepoListResponse represents the response for repo list API.
type RepoListResponse struct {
	Repos   []RepoInfo `json:"repos"`
	Current string     `json:"current"`
}

// GetRepoList handles GET /api/v1/admin/repo-list.
func (h *AdminHandler) GetRepoList(w http.ResponseWriter, r *http.Request) {
	repoLock, err := lock.ReadLock()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read repo lock: %v", err)
		return
	}

	infos := make([]RepoInfo, 0, len(repoLock.Repos))
	for _, entry := range repoLock.Repos {
		status := lock.ValidatePath(entry.Path)
		infos = append(infos, RepoInfo{
			Path:   entry.Path,
			Status: status.String(),
		})
	}

	current := repoLock.GetCurrent()
	validCurrent := ""
	for _, entry := range repoLock.Repos {
		if entry.Path == current && lock.ValidatePath(entry.Path) == lock.PathValid {
			validCurrent = current
			break
		}
	}
	h.writeJSON(w, http.StatusOK, RepoListResponse{
		Repos:   infos,
		Current: validCurrent,
	})
}

// RepoSwitchRequest represents the request to switch the current repo.
type RepoSwitchRequest struct {
	Path string `json:"path"`
}

// RepoSwitchResponse represents the response for repo switch API.
type RepoSwitchResponse struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// PutRepoSwitch handles PUT /api/v1/admin/repo-switch.
func (h *AdminHandler) PutRepoSwitch(w http.ResponseWriter, r *http.Request) {
	var req RepoSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.Path == "" {
		h.writeError(w, http.StatusBadRequest, "path cannot be empty")
		return
	}

	rawPath := req.Path
	if strings.HasPrefix(rawPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			rawPath = filepath.Join(home, rawPath[2:])
		}
	}
	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid path: %v", err)
		return
	}

	status := lock.ValidatePath(absPath)
	switch status {
	case lock.PathNotFound:
		bridge := gitbridge.NewBridge()
		if err := bridge.InitRepo(r.Context(), absPath); err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to initialize repo: %v", err)
			return
		}
	case lock.PathNotGit:
		h.writeError(w, http.StatusUnprocessableEntity, "path is not a git repository: %s", absPath)
		return
	}

	repoLock, err := lock.ReadLock()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read repo lock: %v", err)
		return
	}

	found := false
	for _, entry := range repoLock.Repos {
		if entry.Path == absPath {
			found = true
			break
		}
	}
	if !found {
		repoLock.AddRepo(absPath)
	}

	repoLock.SetCurrent(absPath)
	if err := lock.WriteLock(repoLock); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to write repo lock: %v", err)
		return
	}

	if h.config != nil {
		h.config.PromptAssets.RepoPath = absPath
		if err := h.config.Save(); err != nil {
			h.logger.Error("failed to save config", "error", err, "layer", "L5")
		}
	}

	if h.configManager != nil {
		h.configManager.Notify(r.Context(), "repo", []string{"repo_path"})
	}

	// Trigger async reconcile after repo switch to refresh index with new repo's assets
	if h.indexer != nil {
		go func() {
			ctx := context.Background()
			if report, err := h.indexer.Reconcile(ctx); err != nil {
				h.logger.Warn("reconcile failed after repo switch", "error", err, "layer", "L5")
			} else {
				h.logger.Info("reconcile completed after repo switch",
					"added", report.Added, "updated", report.Updated, "deleted", report.Deleted, "layer", "L5")
			}
		}()
	}

	h.writeJSON(w, http.StatusOK, RepoSwitchResponse{
		Status: "ok",
		Path:   absPath,
	})
}

func (h *AdminHandler) runGit(args []string, repoPath string) (string, error) {
	if repoPath == "" {
		repoPath = "."
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), stderr.String(), err)
	}
	return string(out), nil
}

func (h *AdminHandler) runGitWithContext(ctx context.Context, args []string, repoPath string) (string, error) {
	if repoPath == "" {
		repoPath = "."
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), stderr.String(), err)
	}
	return string(out), nil
}

// GetStatus handles GET /api/v1/admin/status.
func (h *AdminHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	memStat := &runtime.MemStats{}
	runtime.ReadMemStats(memStat)

	lastReload := h.lastReloadTime.Load().(time.Time)
	lastReloadStr := ""
	if !lastReload.IsZero() {
		lastReloadStr = lastReload.Format(time.RFC3339)
	}

	resp := StatusResponse{
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
		Version:       "v0.1.0",
		PID:           h.pid,
		MemoryMB:      int64(memStat.Alloc / 1024 / 1024),
		LastReload:    lastReloadStr,
		RestartCount:  h.restartCount.Load(),
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// ReloadConfig handles POST /api/v1/admin/reload.
func (h *AdminHandler) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("admin action", "action", "reload", "layer", "L5")

	_, err := config.Load("")
	if err != nil {
		h.logger.Error("failed to reload config", "error", err, "layer", "L5")
		h.writeError(w, http.StatusInternalServerError, "Failed to reload config: %v", err)
		return
	}

	h.lastReloadTime.Store(time.Now())
	h.logger.Info("config reloaded", "layer", "L5")

	h.writeJSON(w, http.StatusOK, ReloadResponse{
		Status:    "ok",
		Message:   "Configuration reloaded successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// RestartRequest represents the restart request body.
type RestartRequest struct {
	Graceful       bool `json:"graceful"`
	TimeoutSeconds int  `json:"timeout_seconds"`
}

// Restart handles POST /api/v1/admin/restart.
func (h *AdminHandler) Restart(w http.ResponseWriter, r *http.Request) {
	var req RestartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = RestartRequest{Graceful: true, TimeoutSeconds: 10}
	}

	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 10
	}

	h.logger.Info("admin action", "action", "restart", "pid", h.pid, "graceful", req.Graceful, "timeout", req.TimeoutSeconds, "layer", "L5")

	h.restartCount.Add(1)

	if h.restartFunc != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			h.restartFunc()
		}()
	} else {
		go func() {
			time.Sleep(100 * time.Millisecond)
			syscall.Kill(h.pid, syscall.SIGTERM)
		}()
	}

	h.writeJSON(w, http.StatusOK, RestartResponse{
		Status:    "ok",
		Message:   "Restart signal sent",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// ReconcileReportResponse represents the reconcile result.
type ReconcileReportResponse struct {
	Added   int      `json:"added"`
	Updated int      `json:"updated"`
	Deleted int      `json:"deleted"`
	Errors  []string `json:"errors"`
}

// Reconcile handles POST /api/v1/admin/reconcile.
func (h *AdminHandler) Reconcile(w http.ResponseWriter, r *http.Request) {
	if h.indexer == nil {
		h.writeError(w, http.StatusServiceUnavailable, "indexer not available")
		return
	}

	h.logger.Info("admin action", "action", "reconcile", "layer", "L5")

	report, err := h.indexer.Reconcile(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "reconcile failed: %v", err)
		return
	}

	errors := report.Errors
	if errors == nil {
		errors = []string{}
	}

	h.writeJSON(w, http.StatusOK, ReconcileReportResponse{
		Added:   report.Added,
		Updated: report.Updated,
		Deleted: report.Deleted,
		Errors:  errors,
	})
}

// GitPull handles POST /api/v1/admin/git-pull.
func (h *AdminHandler) GitPull(w http.ResponseWriter, r *http.Request) {
	if h.gitBridge == nil {
		h.writeError(w, http.StatusServiceUnavailable, "git bridge not available")
		return
	}

	h.logger.Info("admin action", "action", "git-pull", "layer", "L5")

	if err := h.gitBridge.Pull(r.Context()); err != nil {
		h.writeError(w, http.StatusInternalServerError, "git pull failed: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "pulled successfully"})
}

// OpenFolder handles POST /api/v1/admin/open-folder.
func (h *AdminHandler) OpenFolder(w http.ResponseWriter, r *http.Request) {
	repoPath := h.config.PromptAssets.RepoPath
	if repoPath == "" {
		h.writeError(w, http.StatusBadRequest, "no repo path configured")
		return
	}

	// Expand ~ to home directory (filepath.Abs does not do this)
	absPath := repoPath
	if strings.HasPrefix(absPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			absPath = filepath.Join(home, absPath[2:])
		}
	}

	// Validate the path is absolute
	if !filepath.IsAbs(absPath) {
		var err error
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid path: %v", err)
			return
		}
	}

	h.logger.Info("admin action", "action", "open-folder", "path", absPath, "layer", "L5")

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", absPath)
	case "linux":
		cmd = exec.Command("xdg-open", absPath)
	case "windows":
		cmd = exec.Command("explorer", absPath)
	default:
		h.writeError(w, http.StatusNotImplemented, "open folder not supported on this platform")
		return
	}

	if err := cmd.Run(); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to open folder: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "opened successfully"})
}

// HandleRepoChange handles repo configuration change notifications.
func (h *AdminHandler) HandleRepoChange(ctx context.Context, domain string, changed []string) {
	if h.indexer == nil && h.gitBridge == nil {
		return
	}

	repoPath := h.config.PromptAssets.RepoPath
	h.logger.Info("repo config change handler triggered", "domain", domain, "changed", changed, "path", repoPath)

	if h.indexer != nil {
		if r, ok := h.indexer.(interface{ ReInit(ctx context.Context, path string) error }); ok {
			if err := r.ReInit(ctx, repoPath); err != nil {
				h.logger.Error("failed to reinit indexer", "error", err)
			}
		}
	}

	if h.gitBridge != nil {
		if r, ok := h.gitBridge.(interface{ ReInit(ctx context.Context, path string) error }); ok {
			if err := r.ReInit(ctx, repoPath); err != nil {
				h.logger.Error("failed to reinit gitBridge", "error", err)
			}
		}
	}
}

// writeJSON writes a JSON response.
func (h *AdminHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *AdminHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	h.writeJSON(w, status, ErrorResponse{
		Status:  "error",
		Message: fmt.Sprintf(format, args...),
	})
}

func isOutsideHome(path string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	absPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		absPath, _ = filepath.Abs(path)
	}
	absHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		absHome, _ = filepath.Abs(home)
	}
	rel, err := filepath.Rel(absHome, absPath)
	if err != nil {
		return true
	}
	return rel == "." || strings.HasPrefix(rel, "..")
}

func sortSlice(repos []RepoInfo) {
	for i := 0; i < len(repos); i++ {
		for j := i + 1; j < len(repos); j++ {
			if repos[i].Path > repos[j].Path {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}
