// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/lock"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/yamlutil"
)

// AssetHandler handles asset CRUD API endpoints.
type AssetHandler struct {
	indexer          service.AssetIndexer
	fileManager      service.AssetFileManager
	semanticAnalyzer service.SemanticAnalyzer
	gitBridge        service.GitBridger
	model            string
	logger           *slog.Logger
	config           *config.Config
}

// NewAssetHandler creates a new AssetHandler.
func NewAssetHandler(indexer service.AssetIndexer, fileManager service.AssetFileManager, logger *slog.Logger, cfg *config.Config) *AssetHandler {
	return &AssetHandler{
		indexer:     indexer,
		fileManager: fileManager,
		logger:      logger,
		config:      cfg,
	}
}

// WithGitBridge sets the git bridge for version history and diff.
func (h *AssetHandler) WithGitBridge(bridge service.GitBridger) *AssetHandler {
	h.gitBridge = bridge
	return h
}

// WithSemanticAnalyzer sets the semantic analyzer for trigger auto-generation.
func (h *AssetHandler) WithSemanticAnalyzer(sa service.SemanticAnalyzer, model string) *AssetHandler {
	h.semanticAnalyzer = sa
	h.model = model
	return h
}

// getCurrentRepoPath returns the current repository path.
// It checks config.PromptAssets.RepoPath first, then falls back to lock file's current repo.
func (h *AssetHandler) getCurrentRepoPath() string {
	if h.config != nil && h.config.PromptAssets.RepoPath != "" {
		return h.config.PromptAssets.RepoPath
	}
	l, err := lock.ReadLock()
	if err != nil {
		return ""
	}
	return l.GetCurrent()
}

// generateTriggers analyzes content and updates triggers in frontmatter.
// Returns the generated triggers or nil if generation failed/skipped.
func (h *AssetHandler) generateTriggers(ctx context.Context, id, content string) ([]domain.TriggerEntry, error) {
	if h.semanticAnalyzer == nil {
		return nil, nil
	}

	// Get existing frontmatter for description and asset_type
	fm, err := h.fileManager.GetFrontmatter(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get frontmatter: %w", err)
	}

	// Analyze content
	result, err := h.semanticAnalyzer.AnalyzeContent(ctx, service.AnalyzeContentRequest{
		Content:     content,
		Description: fm.Description,
		AssetType:     fm.AssetType,
	})
	if err != nil || result == nil {
		return nil, err
	}

	if len(result.Triggers) == 0 {
		return nil, nil
	}

	// Convert service.TriggerEntry to domain.TriggerEntry
	incoming := make([]domain.TriggerEntry, len(result.Triggers))
	for i, t := range result.Triggers {
		incoming[i] = domain.TriggerEntry{
			Pattern:    t.Pattern,
			Examples:   t.Examples,
			Confidence: t.Confidence,
		}
	}

	// Merge with existing triggers (keep higher confidence)
	merged := mergeTriggers(fm.Triggers, incoming)

	// Update frontmatter with merged triggers (write only, no git commit)
	err = h.fileManager.UpdateFrontmatterFileOnly(ctx, id, func(frontmatter *domain.FrontMatter) error {
		frontmatter.Triggers = merged
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("update frontmatter: %w", err)
	}

	return merged, nil
}

// mergeTriggers merges new triggers with existing ones, keeping higher confidence.
// If a pattern already exists with higher confidence, it is kept.
func mergeTriggers(existing, incoming []domain.TriggerEntry) []domain.TriggerEntry {
	if len(incoming) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return incoming
	}

	// Build a map of pattern -> TriggerEntry for existing
	merged := make(map[string]domain.TriggerEntry)
	for _, t := range existing {
		merged[t.Pattern] = t
	}

	// Merge incoming, keeping higher confidence
	for _, t := range incoming {
		existing, ok := merged[t.Pattern]
		if !ok || t.Confidence > existing.Confidence {
			merged[t.Pattern] = t
		}
	}

	// Convert back to slice
	result := make([]domain.TriggerEntry, 0, len(merged))
	for _, t := range merged {
		result = append(result, t)
	}
	return result
}

// AssetResponse represents the API response for an asset.
type AssetResponse struct {
	ID                     string                  `json:"id"`
	Name                   string                  `json:"name"`
	Description            string                  `json:"description,omitempty"`
	AssetType              string                  `json:"asset_type,omitempty"`
	Tags                   []string                `json:"tags,omitempty"`
	State                  string                  `json:"state,omitempty"`
	Labels                 map[string]string       `json:"labels,omitempty"`
	Snapshots              []SnapshotResponse      `json:"snapshots,omitempty"`
	CreatedAt              time.Time               `json:"created_at,omitempty"`
	UpdatedAt              time.Time               `json:"updated_at,omitempty"`
	Category               string                  `json:"category,omitempty"`
	EvalHistory            []EvalHistoryResponse   `json:"eval_history,omitempty"`
	EvalStats              EvalStatsResponse       `json:"eval_stats,omitempty"`
	Triggers               []TriggerResponse       `json:"triggers,omitempty"`
	TestCases              []TestCaseResponse     `json:"test_cases,omitempty"`
	RecommendedSnapshotID  string                  `json:"recommended_snapshot_id,omitempty"`
}

// EvalHistoryResponse represents an eval history entry in API response.
type EvalHistoryResponse struct {
	RunID              string  `json:"run_id"`
	SnapshotID         string  `json:"snapshot_id"`
	Score              int     `json:"score"`
	DeterministicScore float64 `json:"deterministic_score"`
	RubricScore        int     `json:"rubric_score"`
	Model              string  `json:"model"`
	EvalCaseVersion    string  `json:"eval_case_version"`
	TokensIn           int     `json:"tokens_in"`
	TokensOut          int     `json:"tokens_out"`
	DurationMs         int64   `json:"duration_ms"`
	Date               string  `json:"date"`
	By                 string  `json:"by"`
}

// EvalStatsResponse represents eval statistics in API response.
type EvalStatsResponse map[string]ModelStatResponse

// ModelStatResponse represents statistics for a single model.
type ModelStatResponse struct {
	Count   int     `json:"count"`
	Mean    float64 `json:"mean"`
	StdDev  float64 `json:"stddev"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	LastRun string  `json:"last_run"`
}

// TriggerResponse represents a trigger entry in API response.
type TriggerResponse struct {
	Pattern    string   `json:"pattern"`
	Examples   []string `json:"examples,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
}

// TestCaseResponse represents a test case in API response.
type TestCaseResponse struct {
	ID       string                  `json:"id"`
	Name     string                  `json:"name,omitempty"`
	Input    interface{}             `json:"input,omitempty"`
	Expected *TestCaseExpectedResponse `json:"expected,omitempty"`
	Rubric   []TestCaseRubricResponse `json:"rubric,omitempty"`
}

// TestCaseExpectedResponse represents expected output for a test case.
type TestCaseExpectedResponse struct {
	Score   int    `json:"score,omitempty"`
	Content string `json:"content,omitempty"`
}

// TestCaseRubricResponse represents a rubric check in a test case.
type TestCaseRubricResponse struct {
	Check    string  `json:"check"`
	Weight   float64 `json:"weight,omitempty"`
	Criteria string  `json:"criteria,omitempty"`
}

// SnapshotResponse represents a snapshot in API response.
type SnapshotResponse struct {
	Version    string    `json:"version"`
	CommitHash string    `json:"commit_hash,omitempty"`
	Author     string    `json:"author,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	EvalScore  *float64  `json:"eval_score,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// ListAssets handles GET /api/v1/assets.
//
//	@Summary List assets
//	@Description Get all assets with optional filtering
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param category query string false "Category filter (content/eval/metric)"
//	@Param asset_type query string false "Business line filter"
//	@Param tag query string false "Tag filter"
//	@Param state query string false "State filter"
//	@Success 200 {object} map[string]interface{}
//	@Router /api/v1/assets [get]
func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	category := r.URL.Query().Get("category")
	bizLine := r.URL.Query().Get("asset_type")
	tag := r.URL.Query().Get("tag")
	state := r.URL.Query().Get("state")

	filters := service.SearchFilters{
		RepoPath: h.getCurrentRepoPath(),
		Category: category,
		AssetType:  bizLine,
		State:    state,
	}
	if tag != "" {
		filters.Tags = []string{tag}
	}

	results, err := h.indexer.Search(ctx, "", filters)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list assets: %v", err)
		return
	}

	assets := make([]AssetResponse, len(results))
	for i, r := range results {
		assets[i] = AssetResponse{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Category:    r.Category,
			AssetType:     r.AssetType,
			Tags:        r.Tags,
			State:       r.State,
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"assets": assets,
		"total":  len(assets),
	})
}

// GetAsset handles GET /api/v1/assets/{id}.
//
//	@Summary Get asset by ID
//	@Description Get a single asset by its ID
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} AssetResponse
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Router /api/v1/assets/{id} [get]
func (h *AssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil || detail == nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Convert to response format
	snapshots := make([]SnapshotResponse, len(detail.Snapshots))
	for i, s := range detail.Snapshots {
		snapshots[i] = SnapshotResponse{
			Version:    s.Version,
			CommitHash: s.CommitHash,
			Author:     s.Author,
			Reason:     s.Reason,
			EvalScore:  s.EvalScore,
			CreatedAt:  s.CreatedAt,
		}
	}

	labels := make(map[string]string)
	for _, l := range detail.Labels {
		labels[l.Name] = l.SnapshotID
	}

	// Convert eval history
	evalHistory := make([]EvalHistoryResponse, len(detail.EvalHistory))
	for i, eh := range detail.EvalHistory {
		evalHistory[i] = EvalHistoryResponse{
			RunID:              eh.RunID,
			SnapshotID:         eh.SnapshotID,
			Score:              eh.Score,
			DeterministicScore: eh.DeterministicScore,
			RubricScore:        eh.RubricScore,
			Model:              eh.Model,
			EvalCaseVersion:    eh.EvalCaseVersion,
			TokensIn:           eh.TokensIn,
			TokensOut:          eh.TokensOut,
			DurationMs:         eh.DurationMs,
			Date:               eh.Date,
			By:                 eh.By,
		}
	}

	// Convert eval stats
	evalStats := make(EvalStatsResponse)
	for model, stat := range detail.EvalStats {
		evalStats[model] = ModelStatResponse{
			Count:   stat.Count,
			Mean:    stat.Mean,
			StdDev:  stat.StdDev(),
			Min:     stat.Min,
			Max:     stat.Max,
			LastRun: stat.LastRun,
		}
	}

	// Convert triggers
	triggers := make([]TriggerResponse, len(detail.Triggers))
	for i, t := range detail.Triggers {
		triggers[i] = TriggerResponse{
			Pattern:    t.Pattern,
			Examples:   t.Examples,
			Confidence: t.Confidence,
		}
	}

	// Convert test cases
	testCases := make([]TestCaseResponse, len(detail.TestCases))
	for i, tc := range detail.TestCases {
		testCases[i] = TestCaseResponse{
			ID:   tc.ID,
			Name: tc.Name,
			Input: tc.Input,
		}
		if tc.Expected != nil {
			testCases[i].Expected = &TestCaseExpectedResponse{
				Score:   tc.Expected.Score,
				Content: tc.Expected.Content,
			}
		}
		for _, r := range tc.Rubric {
			testCases[i].Rubric = append(testCases[i].Rubric, TestCaseRubricResponse{
				Check:    r.Check,
				Weight:   r.Weight,
				Criteria: r.Criteria,
			})
		}
	}

	resp := AssetResponse{
		ID:                     detail.ID,
		Name:                   detail.Name,
		Description:            detail.Description,
		AssetType:              detail.AssetType,
		Tags:                   detail.Tags,
		State:                  detail.State,
		Snapshots:              snapshots,
		Labels:                 labels,
		Category:               detail.Category,
		EvalHistory:            evalHistory,
		EvalStats:              evalStats,
		Triggers:               triggers,
		TestCases:              testCases,
		RecommendedSnapshotID:  detail.RecommendedSnapshotID,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// CreateAssetRequest represents the request body for creating an asset.
type CreateAssetRequest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	AssetType     string   `json:"asset_type,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Content     string   `json:"content,omitempty"`
	Category    string   `json:"category,omitempty"`
}

// CreateAsset handles POST /api/v1/assets.
//
//	@Summary Create asset
//	@Description Create a new asset
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param request body CreateAssetRequest true "Asset creation request"
//	@Success 201 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets [post]
func (h *AssetHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.ID == "" || req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "id and name are required")
		return
	}

	asset := service.Asset{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		AssetType:     req.AssetType,
		Tags:        req.Tags,
		State:       "created",
		RepoPath:    h.getCurrentRepoPath(),
	}

	if err := h.indexer.Save(ctx, asset); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create asset: %v", err)
		return
	}

	// Create placeholder file and commit to Git (best effort — non-fatal if git unavailable)
	if err := h.indexer.CreatePlaceholder(ctx, req.ID, req.Name, req.AssetType, req.Tags, req.Category); err != nil {
		// Log but don't fail — placeholder is a courtesy for Git users
		h.logger.Warn("failed to create placeholder file", "asset_id", req.ID, "error", err, "layer", "L5")
	}

	h.logger.Info("asset created", "asset_id", req.ID, "layer", "L5")

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":      req.ID,
		"message": "asset created successfully",
	})
}

// UpdateAssetRequest represents the request body for updating an asset.
type UpdateAssetRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	AssetType     *string  `json:"asset_type,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	State       *string  `json:"state,omitempty"`
}

// UpdateAsset handles PUT /api/v1/assets/{id}.
//
//	@Summary Update asset
//	@Description Update an existing asset
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Param request body UpdateAssetRequest true "Asset update request"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id} [put]
func (h *AssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req UpdateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	// Get existing asset
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Apply updates
	if req.Name != nil {
		detail.Name = *req.Name
	}
	if req.Description != nil {
		detail.Description = *req.Description
	}
	if req.AssetType != nil {
		detail.AssetType = *req.AssetType
	}
	if req.Tags != nil {
		detail.Tags = req.Tags
	}
	if req.State != nil {
		detail.State = *req.State
	}

	asset := service.Asset{
		ID:          detail.ID,
		Name:        detail.Name,
		Description: detail.Description,
		AssetType:     detail.AssetType,
		Tags:        detail.Tags,
		State:       detail.State,
	}

	if err := h.indexer.Save(ctx, asset); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to update asset: %v", err)
		return
	}

	h.logger.Info("asset updated", "asset_id", id, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "asset updated successfully",
	})
}

// SaveContentRequest represents the request body for saving file content.
type SaveContentRequest struct {
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message,omitempty"`
	ContentHash   string `json:"content_hash,omitempty"` // for conflict detection
}

// GetAssetContent handles GET /api/v1/assets/{id}/content.
// Returns the main file content for folder-based assets, or stripped body for legacy .md assets.
func (h *AssetHandler) GetAssetContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Get asset detail to check if it's using new folder structure
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil || detail == nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Check if using new folder structure (asset.yaml)
	if detail.AssetPath != "" {
		// New folder structure: read main file from asset.yaml
		content, mainPath, isExternal, err := h.fileManager.GetMainFileContent(ctx, detail.AssetPath)
		if err != nil {
			h.writeError(w, http.StatusNotFound, "content not found: %s", err)
			return
		}

		// Compute content hash
		hashed := sha256.Sum256([]byte(content))
		contentHash := hex.EncodeToString(hashed[:8])

		// Set response headers
		w.Header().Set("X-Content-Hash", contentHash)
		w.Header().Set("X-Main-Path", mainPath)
		w.Header().Set("X-Is-External", fmt.Sprintf("%v", isExternal))

		h.writeJSON(w, http.StatusOK, map[string]any{
			"id":           id,
			"content":      content,
			"content_hash": contentHash,
			"main_path":    mainPath,
			"is_external":  isExternal,
		})
		return
	}

	// Legacy .md file structure: read and strip frontmatter
	fullContent, err := h.indexer.GetFileContent(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "content not found: %s", err)
		return
	}

	// Strip frontmatter — find the second ---
	lines := strings.Split(fullContent, "\n")
	frontmatterEnd := -1
	inFrontmatter := false
	for i, line := range lines {
		if i == 0 && strings.HasPrefix(line, "---") {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(line, "---") {
			frontmatterEnd = i
			break
		}
	}

	var body string
	var contentHash string
	var updatedAt string
	if frontmatterEnd >= 0 {
		// Parse frontmatter to get content_hash and updated_at
		frontmatterBlock := strings.Join(lines[1:frontmatterEnd], "\n")
		fullFrontmatter := "---\n" + frontmatterBlock + "\n---"
		fm, _, err := yamlutil.ParseFrontMatter(fullFrontmatter)
		if err != nil {
			h.logger.Warn("failed to parse frontmatter", "asset_id", id, "error", err, "layer", "L5")
		}
		if fm != nil {
			contentHash = fm.ContentHash
			if !fm.UpdatedAt.IsZero() {
				updatedAt = fm.UpdatedAt.Format(time.RFC3339)
				w.Header().Set("Last-Modified", updatedAt)
			}
		}
		body = yamlutil.NormalizeBody(strings.Join(lines[frontmatterEnd+1:], "\n"))
	} else {
		// No frontmatter found, return as-is
		body = fullContent
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":           id,
		"content":      body,
		"content_hash":  contentHash,
		"updated_at":   updatedAt,
	})
}

// SaveAssetContent handles PUT /api/v1/assets/{id}/content.
// For folder-based assets: writes to main file and updates asset.yaml (no git commit).
// For legacy .md assets: replaces body, updates frontmatter, writes back (no git commit).
// Use CommitAsset to manually commit changes.
func (h *AssetHandler) SaveAssetContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req SaveContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.Content == "" {
		h.writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Get asset detail to check if it's using new folder structure
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil || detail == nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	var newHash string
	var updatedAt time.Time

	// Check if using new folder structure (asset.yaml)
	if detail.AssetPath != "" {
		// New folder structure: write to main file via asset.yaml
		// Conflict detection using content hash
		if req.ContentHash != "" {
			currentContent, _, _, err := h.fileManager.GetMainFileContent(ctx, detail.AssetPath)
			if err == nil {
				currentHash := sha256.Sum256([]byte(currentContent))
				currentHashStr := hex.EncodeToString(currentHash[:8])
				if currentHashStr != req.ContentHash {
					h.writeError(w, http.StatusConflict, "content has been modified by another session")
					return
				}
			}
		}

		// Write to main file
		newHash, err = h.fileManager.WriteMainFileContent(ctx, detail.AssetPath, req.Content)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to save content: %v", err)
			return
		}
		updatedAt = time.Now()

		h.logger.Info("asset content saved (folder structure)", "asset_id", id, "asset_path", detail.AssetPath, "layer", "L5")

		// Preference-Applied: return=representation
		w.Header().Set("Preference-Applied", "return=representation")
		w.Header().Set("Last-Modified", updatedAt.Format(time.RFC3339))

		h.writeJSON(w, http.StatusOK, map[string]any{
			"id":           id,
			"content":      req.Content,
			"content_hash": newHash,
			"updated_at":   updatedAt.Format(time.RFC3339),
			"message":      "content saved successfully (use /commit to save to git)",
		})
		return
	}

	// Legacy .md file structure
	// Conflict detection: compute hash from current file's body and compare with client's hash
	if req.ContentHash != "" {
		currentBody, err := h.fileManager.GetBody(ctx, id)
		if err == nil {
			currentHash := sha256.Sum256([]byte(yamlutil.NormalizeBody(currentBody)))
			currentHashStr := hex.EncodeToString(currentHash[:8])
			if currentHashStr != req.ContentHash {
				h.writeError(w, http.StatusConflict, "content has been modified by another session")
				return
			}
		}
	}

	// Normalize body for consistent hashing and storage
	normalizedContent := yamlutil.NormalizeBody(req.Content)

	// Compute new hash
	hashed := sha256.Sum256([]byte(normalizedContent))
	newHash = hex.EncodeToString(hashed[:8])
	updatedAt = time.Now()

	// WriteFileOnly saves the file without git commit
	err = h.fileManager.WriteFileOnly(ctx, id, func(fm *domain.FrontMatter) error {
		fm.ContentHash = newHash
		fm.UpdatedAt = updatedAt
		return nil
	}, normalizedContent)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save content: %v", err)
		return
	}

	h.logger.Info("asset content saved", "asset_id", id, "layer", "L5")

	// Async: generate triggers in background (does not block response)
	if h.semanticAnalyzer != nil {
		go func() {
			bgCtx := context.Background()
			triggers, err := h.generateTriggers(bgCtx, id, req.Content)
			if err == nil && len(triggers) > 0 {
				h.logger.Info("triggers auto-generated", "asset_id", id, "count", len(triggers), "layer", "L5")
			}
		}()
	}

	// Preference-Applied: return=representation
	w.Header().Set("Preference-Applied", "return=representation")
	w.Header().Set("Last-Modified", updatedAt.Format(time.RFC3339))

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":           id,
		"content":      normalizedContent,
		"content_hash": newHash,
		"updated_at":   updatedAt.Format(time.RFC3339),
		"message":      "content saved successfully (use /commit to save to git)",
	})
}

// CommitAsset handles POST /api/v1/assets/{id}/commit.
// Commits the current state of the asset file to Git.
func (h *AssetHandler) CommitAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	commitMsg := r.URL.Query().Get("message")
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Commit asset %s", id)
	}

	// Commit via indexer (stages and commits the existing file without modifying it)
	hash, err := h.indexer.CommitFile(ctx, id, commitMsg)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to commit: %v", err)
		return
	}

	h.logger.Info("asset committed", "asset_id", id, "commit", hash, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"commit":  hash,
		"message": "asset committed successfully",
	})
}

// CommitBatchAssetsRequest represents the request body for batch commit.
type CommitBatchAssetsRequest struct {
	IDs     []string `json:"ids"`
	Message string   `json:"message,omitempty"`
}

// CommitBatchAssets handles POST /api/v1/assets/commit.
// Commits multiple assets in batch.
func (h *AssetHandler) CommitBatchAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CommitBatchAssetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if len(req.IDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "ids are required")
		return
	}

	commitMsg := req.Message
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Batch commit %d assets", len(req.IDs))
	}

	commits, err := h.indexer.CommitFiles(ctx, req.IDs, commitMsg)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "batch commit failed: %v", err)
		return
	}

	h.logger.Info("batch assets committed", "count", len(commits), "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"commits": commits,
		"message": fmt.Sprintf("committed %d assets", len(commits)),
	})
}

// GetAssetFiles handles GET /api/v1/assets/{id}/files.
// Returns the list of files associated with an asset (from asset.yaml files and external lists).
func (h *AssetHandler) GetAssetFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Get asset detail to find the asset.yaml path
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil || detail == nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Check if using new folder structure (asset.yaml)
	if detail.AssetPath == "" {
		h.writeError(w, http.StatusNotFound, "asset does not use folder structure")
		return
	}

	// Get files and external lists from asset.yaml
	files, external, err := h.fileManager.GetAssetFiles(ctx, detail.AssetPath)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get asset files: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"files":   files,
		"external": external,
	})
}

// AssetHistory handles GET /api/v1/assets/{id}/history.
// Returns the git commit history for an asset.
func (h *AssetHandler) AssetHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	limit := 10 // default limit
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// Get asset to find the file path
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil || detail == nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Get commit history
	if h.gitBridge == nil {
		h.writeError(w, http.StatusServiceUnavailable, "git bridge not available")
		return
	}

	var commits []service.CommitInfo

	if detail.AssetPath != "" {
		// New folder structure: get history for both asset.yaml and main file
		yamlCommits, err := h.gitBridge.Log(ctx, detail.AssetPath, limit)
		if err == nil {
			commits = append(commits, yamlCommits...)
		}
		if detail.Main != "" {
			mainCommits, err := h.gitBridge.Log(ctx, detail.Main, limit)
			if err == nil {
				commits = append(commits, mainCommits...)
			}
		}
		// Deduplicate by hash
		commits = deduplicateCommits(commits)
		// Sort by timestamp descending
		sortCommitsByDate(commits)
		// Apply limit
		if len(commits) > limit {
			commits = commits[:limit]
		}
	} else {
		// Legacy .md structure
		filePath := fmt.Sprintf("prompts/%s.md", id)
		commits, err = h.gitBridge.Log(ctx, filePath, limit)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to get history: %v", err)
			return
		}
	}

	// Convert to response format
	type CommitResponse struct {
		Hash      string `json:"hash"`
		ShortHash string `json:"short_hash"`
		Date      string `json:"date"`
		Message   string `json:"message"`
		Author    string `json:"author"`
	}

	commitsResp := make([]CommitResponse, len(commits))
	for i, c := range commits {
		commitsResp[i] = CommitResponse{
			Hash:      c.Hash,
			ShortHash: c.ShortHash,
			Date:      c.Timestamp.Format(time.RFC3339),
			Message:   c.Subject,
			Author:    c.Author,
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"commits": commitsResp,
	})
}

// AssetDiff handles GET /api/v1/assets/{id}/diff.
// Returns the diff between two commits for an asset.
// AssetDiff handles GET /api/v1/assets/{id}/diff.
// Returns the diff between two commits for an asset, limited to the asset's files.
func (h *AssetHandler) AssetDiff(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		h.writeError(w, http.StatusBadRequest, "from and to commit hashes are required")
		return
	}

	// Get asset to verify it exists
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil || detail == nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Get diff
	if h.gitBridge == nil {
		h.writeError(w, http.StatusServiceUnavailable, "git bridge not available")
		return
	}

	diff, err := h.gitBridge.Diff(ctx, from, to)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get diff: %v", err)
		return
	}

	// Determine files affected
	var files []string
	if detail.AssetPath != "" {
		files = append(files, detail.AssetPath)
		if detail.Main != "" {
			files = append(files, detail.Main)
		}
	} else {
		files = append(files, fmt.Sprintf("prompts/%s.md", id))
	}

	// Filter diff to only include the asset's files
	filteredDiff := filterDiffByFiles(diff, files)

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"from":  from,
		"to":    to,
		"files": files,
		"diff":  filteredDiff,
	})
}

// BatchTagAssetsRequest represents the request body for batch tag operations.
type BatchTagAssetsRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"` // "add" or "remove"
	Tag    string   `json:"tag"`
}

// BatchTagAssets handles POST /api/v1/assets/batch/tag.
// Batch add or remove tags from multiple assets.
func (h *AssetHandler) BatchTagAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req BatchTagAssetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if len(req.IDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "ids are required")
		return
	}

	if req.Tag == "" {
		h.writeError(w, http.StatusBadRequest, "tag is required")
		return
	}

	if req.Action != "add" && req.Action != "remove" {
		h.writeError(w, http.StatusBadRequest, "action must be 'add' or 'remove'")
		return
	}

	results := make(map[string]string)
	errors := make(map[string]string)

	for _, id := range req.IDs {
		// Get the asset
		detail, err := h.indexer.GetByID(ctx, id)
		if err != nil || detail == nil {
			errors[id] = "asset not found"
			continue
		}

		// Update tags based on action
		var newTags []string
		if req.Action == "add" {
			// Check if tag already exists
			hasTag := false
			for _, t := range detail.Tags {
				if t == req.Tag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				newTags = make([]string, len(detail.Tags))
				copy(newTags, detail.Tags)
				newTags = append(newTags, req.Tag)
			} else {
				newTags = make([]string, len(detail.Tags))
				copy(newTags, detail.Tags)
			}
		} else {
			// Remove tag
			for _, t := range detail.Tags {
				if t != req.Tag {
					newTags = append(newTags, t)
				}
			}
		}

		// Update the asset
		asset := service.Asset{
			ID:          detail.ID,
			Name:        detail.Name,
			Description: detail.Description,
			AssetType:     detail.AssetType,
			Tags:        newTags,
			State:       detail.State,
		}
		if err := h.indexer.Save(ctx, asset); err != nil {
			errors[id] = fmt.Sprintf("failed to update index: %v", err)
			continue
		}

		// For legacy .md structure, update frontmatter
		if detail.AssetPath == "" {
			_, err = h.fileManager.UpdateFrontmatter(ctx, id, func(fm *domain.FrontMatter) error {
				fm.Tags = newTags
				return nil
			}, fmt.Sprintf("Batch %s tag '%s'", req.Action, req.Tag))
			if err != nil {
				errors[id] = fmt.Sprintf("failed to update frontmatter: %v", err)
				continue
			}
		} else {
			// New folder structure: update asset.yaml
			ay, err := h.fileManager.GetAssetYAML(ctx, detail.AssetPath)
			if err != nil {
				errors[id] = fmt.Sprintf("failed to read asset.yaml: %v", err)
				continue
			}
			ay.Tags = newTags
			_, err = h.fileManager.SaveAssetYAML(ctx, detail.AssetPath, ay, fmt.Sprintf("Batch %s tag '%s'", req.Action, req.Tag))
			if err != nil {
				errors[id] = fmt.Sprintf("failed to update asset.yaml: %v", err)
				continue
			}
		}

		results[id] = "success"
		h.logger.Info("batch tag updated", "asset_id", id, "action", req.Action, "tag", req.Tag, "layer", "L5")
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"updated": results,
		"errors":  errors,
		"message": fmt.Sprintf("batch %s tag '%s' completed", req.Action, req.Tag),
	})
}

// ArchiveAsset handles POST /api/v1/assets/{id}/archive.
//
//	@Summary Archive asset
//	@Description Archive an asset (soft delete)
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id}/archive [post]
func (h *AssetHandler) ArchiveAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Update frontmatter state to archived
	hash, err := h.fileManager.UpdateFrontmatter(ctx, id, func(fm *domain.FrontMatter) error {
		fm.State = "archived"
		fm.UpdatedAt = time.Now()
		return nil
	}, fmt.Sprintf("Archive asset %s", id))
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset file not found: %s", err)
		return
	}

	// Update in-memory index so UI reflects change immediately
	fm, err := h.fileManager.GetFrontmatter(ctx, id)
	if err == nil {
		asset := service.Asset{
			ID:          fm.ID,
			Name:        fm.Name,
			Description: fm.Description,
			AssetType:     fm.AssetType,
			Tags:        fm.Tags,
			ContentHash: fm.ContentHash,
			State:       fm.State,
		}
		if err := h.indexer.Save(ctx, asset); err != nil {
			h.logger.Warn("failed to update index after archive", "asset_id", id, "error", err, "layer", "L5")
		}
	}

	h.logger.Info("asset archived", "asset_id", id, "commit", hash, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"state":   "archived",
		"message": "asset archived successfully",
	})
}

// RestoreAsset handles POST /api/v1/assets/{id}/restore.
//
//	@Summary Restore asset
//	@Description Restore an archived asset to active state
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id}/restore [post]
func (h *AssetHandler) RestoreAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Update frontmatter state to active
	hash, err := h.fileManager.UpdateFrontmatter(ctx, id, func(fm *domain.FrontMatter) error {
		fm.State = "active"
		fm.UpdatedAt = time.Now()
		return nil
	}, fmt.Sprintf("Restore asset %s", id))
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset file not found: %s", err)
		return
	}

	// Update in-memory index so UI reflects change immediately
	fm, err := h.fileManager.GetFrontmatter(ctx, id)
	if err == nil {
		asset := service.Asset{
			ID:          fm.ID,
			Name:        fm.Name,
			Description: fm.Description,
			AssetType:     fm.AssetType,
			Tags:        fm.Tags,
			ContentHash: fm.ContentHash,
			State:       fm.State,
		}
		if err := h.indexer.Save(ctx, asset); err != nil {
			h.logger.Warn("failed to update index after restore", "asset_id", id, "error", err, "layer", "L5")
		}
	}

	h.logger.Info("asset restored", "asset_id", id, "commit", hash, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"state":   "active",
		"message": "asset restored successfully",
	})
}

// DeleteAsset handles DELETE /api/v1/assets/{id}.
//
//	@Summary Delete asset
//	@Description Delete an asset by ID
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id} [delete]
func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Check if asset exists first
	if _, err := h.indexer.GetByID(ctx, id); err != nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	if err := h.indexer.Delete(ctx, id); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to delete asset: %v", err)
		return
	}

	h.logger.Info("asset deleted", "asset_id", id, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "asset deleted successfully",
	})
}

// writeJSON writes a JSON response.
func (h *AssetHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *AssetHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": fmt.Sprintf(format, args...),
	})
}

// deduplicateCommits removes duplicate commits based on hash.
func deduplicateCommits(commits []service.CommitInfo) []service.CommitInfo {
	seen := make(map[string]bool)
	result := make([]service.CommitInfo, 0, len(commits))
	for _, c := range commits {
		if !seen[c.Hash] {
			seen[c.Hash] = true
			result = append(result, c)
		}
	}
	return result
}

// sortCommitsByDate sorts commits by timestamp in descending order (newest first).
func sortCommitsByDate(commits []service.CommitInfo) {
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Timestamp.After(commits[j].Timestamp)
	})
}

// filterDiffByFiles filters a git diff output to only include changes to the specified files.
func filterDiffByFiles(diff string, files []string) string {
	if diff == "" || len(files) == 0 {
		return diff
	}

	// Create a set of files to include
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	// Split diff into blocks (each block starts with "diff --git")
	var result string
	var currentBlock string
	inBlock := false
	currentFile := ""

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			// Start of a new block - flush previous block if it matches
			if inBlock && currentFile != "" && fileSet[currentFile] {
				result += currentBlock
			}
			currentBlock = line + "\n"
			inBlock = true
			currentFile = ""

			// Extract file path from "diff --git a/path b/path"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				// a/path and b/path - extract the b/path (destination)
				// or just use a/path since they're usually the same
				gitPath := strings.TrimPrefix(parts[2], "a/")
				currentFile = gitPath
			}
		} else if inBlock {
			currentBlock += line + "\n"
			// Check if this block should be included
			if currentFile != "" && fileSet[currentFile] {
				// This block matches - we don't need to do anything special here
				// since we're already including all lines
			}
		} else {
			// Lines before first diff block (e.g., warnings)
			result += line + "\n"
		}
	}

	// Flush the last block
	if inBlock && currentFile != "" && fileSet[currentFile] {
		result += currentBlock
	}

	return result
}
