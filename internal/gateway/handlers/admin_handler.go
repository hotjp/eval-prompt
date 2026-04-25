// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/eval-prompt/internal/config"
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
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(logger *slog.Logger, cfg *config.Config, restartFunc func()) *AdminHandler {
	h := &AdminHandler{
		logger:    logger,
		startTime: time.Now(),
		pid:       os.Getpid(),
		config:    cfg,
	}
	h.lastReloadTime.Store(time.Time{})
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
	repoPath := "."

	var branch, shortCommit, remote string
	var dirty bool

	// Get current branch
	out, err := h.runGit([]string{"rev-parse", "--abbrev-ref", "HEAD"}, repoPath)
	if err == nil {
		branch = strings.TrimSpace(out)
	}

	// Get short commit hash
	out, err = h.runGit([]string{"rev-parse", "--short", "HEAD"}, repoPath)
	if err == nil {
		shortCommit = strings.TrimSpace(out)
	}

	// Check dirty state
	out, err = h.runGit([]string{"status", "--porcelain"}, repoPath)
	if err == nil && strings.TrimSpace(out) != "" {
		dirty = true
	}

	// Get remote URL
	out, err = h.runGit([]string{"remote", "get-url", "origin"}, repoPath)
	if err == nil {
		remote = strings.TrimSpace(out)
		// Shorten common remote URLs
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

	h.config.PromptAssets.RepoPath = req.RepoPath
	h.config.PromptAssets.AssetsDir = req.AssetsDir
	h.config.PromptAssets.EvalsDir = req.EvalsDir

	h.writeJSON(w, http.StatusOK, RepoConfigResponse{
		RepoPath:  h.config.PromptAssets.RepoPath,
		AssetsDir: h.config.PromptAssets.AssetsDir,
		EvalsDir:  h.config.PromptAssets.EvalsDir,
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
	Graceful      bool `json:"graceful"`
	TimeoutSeconds int `json:"timeout_seconds"`
}

// Restart handles POST /api/v1/admin/restart.
func (h *AdminHandler) Restart(w http.ResponseWriter, r *http.Request) {
	var req RestartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use defaults if body is empty
		req = RestartRequest{Graceful: true, TimeoutSeconds: 10}
	}

	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 10
	}

	h.logger.Info("admin action", "action", "restart", "pid", h.pid, "graceful", req.Graceful, "timeout", req.TimeoutSeconds, "layer", "L5")

	h.restartCount.Add(1)

	// Signal restart - this will trigger graceful shutdown and restart
	if h.restartFunc != nil {
		go func() {
			time.Sleep(100 * time.Millisecond) // Give time for response to be sent
			h.restartFunc()
		}()
	} else {
		// Fallback: send SIGTERM directly
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
