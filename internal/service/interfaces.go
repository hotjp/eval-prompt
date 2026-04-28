// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"time"

	"github.com/eval-prompt/internal/domain"
)

// CommitInfo represents git commit information.
type CommitInfo struct {
	Hash      string
	ShortHash string
	Subject   string
	Body      string
	Author    string
	Timestamp time.Time
}

// GitBridger is the interface for Git operations on prompt assets.
// Implemented by plugins/gitbridge.
type GitBridger interface {
	// InitRepo initializes a new Git repository at the given path.
	InitRepo(ctx context.Context, path string) error

	// StageAndCommit stages the file at filePath and creates a commit with the given message.
	// Returns the commit hash.
	StageAndCommit(ctx context.Context, filePath, message string) (string, error)

	// StageAndCommitFiles stages multiple files and creates a single commit with the given message.
	// Returns the commit hash.
	StageAndCommitFiles(ctx context.Context, filePaths []string, message string) (string, error)

	// Diff returns the diff output between two commits (commit1 and commit2).
	Diff(ctx context.Context, commit1, commit2 string) (string, error)

	// Log returns the commit log for a file, limited to the specified number of entries.
	Log(ctx context.Context, filePath string, limit int) ([]CommitInfo, error)

	// Status returns the current working tree status: added, modified, and deleted files.
	Status(ctx context.Context) (added, modified, deleted []string, err error)

	// Pull fetches and merges changes from the remote repository.
	Pull(ctx context.Context) error

	// RepoPath returns the root path of the Git repository.
	RepoPath() string

	// SetPath sets the repository path for the Bridge.
	SetPath(path string)
}

// AssetIndexer is the interface for indexing and searching prompt assets.
// Implemented by plugins/search.
type AssetIndexer interface {
	// Reconcile synchronizes the index with the Git repository.
	Reconcile(ctx context.Context) (ReconcileReport, error)

	// Search searches for assets matching the query and filters.
	Search(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error)

	// GetByID retrieves an asset by its ID.
	GetByID(ctx context.Context, id string) (*AssetDetail, error)

	// Save saves an asset to the index.
	Save(ctx context.Context, asset Asset) error

	// Delete removes an asset from the index.
	Delete(ctx context.Context, id string) error

	// GetMainFileContent reads the main file content from an asset.yaml.
	// assetPath is the path to asset.yaml (e.g., "assets/skills/calculator.yaml").
	// Returns the main file content, resolved main path, isExternal flag, and error.
	GetMainFileContent(ctx context.Context, assetPath string) (content string, mainPath string, isExternal bool, err error)

	// WriteMainFileContent writes content to the main file specified in asset.yaml.
	// assetPath is the path to asset.yaml.
	// Returns the new content hash and error.
	WriteMainFileContent(ctx context.Context, assetPath string, content string) (newContentHash string, err error)

	// GetAssetFiles returns the files and external file lists from an asset.yaml.
	// assetPath is the path to asset.yaml.
	// Returns the files list, external list, and error.
	GetAssetFiles(ctx context.Context, assetPath string) (files []FileInfo, external []FileInfo, err error)

	// ReInit reinitializes the indexer with a new repository path.
	// It clears the current index and updates the git bridge path.
	ReInit(ctx context.Context, path string) error

	// CommitFile stages and commits an asset without modifying its content.
	// For folder-based assets, commits both asset.yaml and main file.
	// Returns the commit hash.
	CommitFile(ctx context.Context, id string, commitMsg string) (string, error)

	// CommitFiles stages and commits multiple assets in batch.
	// Returns a map of asset ID to commit hash.
	CommitFiles(ctx context.Context, ids []string, commitMsg string) (map[string]string, error)
}

// SearchFilters contains filter criteria for asset search.
type SearchFilters struct {
	RepoPath string
	Category string
	AssetType  string
	Tags     []string
	State    string
	Label    string
}

// AssetSummary is a condensed asset representation for search results.
type AssetSummary struct {
	ID          string
	Name        string
	Description string
	Category    string
	AssetType   string
	Tags        []string
	State       string
	LatestScore *float64
	Keywords    []string    `json:"keywords,omitempty"` // LLM-generated keywords for enhanced search
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
}

// AssetDetail is a full asset representation.
type AssetDetail struct {
	ID          string
	Name        string
	Description string
	AssetType     string
	Tags        []string
	State       string
	Snapshots   []SnapshotSummary
	Labels      []LabelInfo
	Category              string                    `json:"category,omitempty"`
	EvalHistory          []domain.EvalHistoryEntry `json:"eval_history,omitempty"`
	EvalStats            domain.EvalStats         `json:"eval_stats,omitempty"`
	Triggers             []domain.TriggerEntry    `json:"triggers,omitempty"`
	TestCases            []domain.TestCase         `json:"test_cases,omitempty"`
	RecommendedSnapshotID string                   `json:"recommended_snapshot_id,omitempty"`
	// Keywords are LLM-generated keywords for enhanced search
	Keywords     []string `json:"keywords,omitempty"`

	// AssetPath is the path to the asset.yaml registry file (e.g., "assets/skills/calculator.yaml")
	AssetPath string `json:"asset_path,omitempty"`
	// Main is the main file path from asset.yaml (e.g., "skills/calculator/handler.py")
	Main string `json:"main,omitempty"`
	// ContentHash is the SHA256 hash of the main file content (first 8 bytes, hex encoded)
	ContentHash string `json:"content_hash,omitempty"`
	// IsExternal indicates whether the main file is outside the repo
	IsExternal bool `json:"is_external,omitempty"`
}

// SnapshotSummary is a condensed snapshot representation.
type SnapshotSummary struct {
	Version    string
	CommitHash string
	Author     string
	Reason     string
	EvalScore  *float64
	CreatedAt  time.Time
}

// LabelInfo represents a label on an asset.
type LabelInfo struct {
	Name       string
	SnapshotID string
	UpdatedAt  time.Time
}

// ReconcileReport contains the results of a reconciliation.
type ReconcileReport struct {
	Added   int
	Updated int
	Deleted int
	Errors  []string
}

// ParseLabels converts domain.LabelEntry slice to LabelInfo slice.
func ParseLabels(fmLabels []domain.LabelEntry) []LabelInfo {
	if fmLabels == nil {
		return []LabelInfo{}
	}
	result := make([]LabelInfo, len(fmLabels))
	for i, l := range fmLabels {
		result[i] = LabelInfo{
			Name:       l.Name,
			SnapshotID: l.Snapshot,
			UpdatedAt:  func() time.Time { t, _ := time.Parse("2006-01-02", l.Date); return t }(),
		}
	}
	return result
}

// FileMetadata contains metadata embedded in a prompt file's frontmatter.
type FileMetadata struct {
	Name        string
	Description string
	AssetType     string
	Tags        []string
	State       string
	Version     string
}

// Asset represents a prompt asset (mirrors domain.Asset for plugin use).
type Asset struct {
	ID          string
	Name        string
	Description string
	AssetType   string
	Category    string
	Tags        []string
	ContentHash string
	FilePath    string
	RepoPath    string // repo isolation
	AssetPath   string // path to asset.yaml registry file (e.g., "assets/prompts/id.yaml")
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LLMResponse contains the LLM output and metadata.
// Mirrors plugins/llm.LLMResponse to avoid importing the plugin package in service layer.
type LLMResponse struct {
	Content   string
	TokensIn  int
	TokensOut int
	Model     string
}

// LLMInvoker abstracts LLM provider calls.
// Implemented by plugins/llm via an adapter.
type LLMInvoker interface {
	// Invoke calls the LLM with a prompt, model, and temperature.
	// Returns the text response and usage metadata.
	Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)

	// InvokeWithSchema calls the LLM and enforces a JSON output schema.
	// The schema is a JSON Schema (draft-07) describing the expected output structure.
	// Returns the parsed JSON response.
	InvokeWithSchema(ctx context.Context, prompt string, schema []byte) ([]byte, error)
}

// SemanticAnalyzer provides LLM-based semantic capabilities.
type SemanticAnalyzer interface {
	AnalyzeContent(ctx context.Context, req AnalyzeContentRequest) (*AnalyzeContentResult, error)
	ExplainDiff(ctx context.Context, req ExplainDiffRequest) (*ExplainDiffResult, error)
}

type AnalyzeContentRequest struct {
	Content     string
	Description string
	AssetType     string
}

type AnalyzeContentResult struct {
	Triggers []TriggerEntry
	Issues   []ContentIssue
	Score    ContentScore
}

type TriggerEntry struct {
	Pattern    string
	Examples   []string
	Confidence float64
}

type ContentIssue struct {
	Severity   string
	Location   string
	Problem    string
	Suggestion string
}

type ContentScore struct {
	Overall      float64
	Clarity      float64
	Completeness float64
}

type ExplainDiffRequest struct {
	OldContent string
	NewContent string
	OldVersion string
	NewVersion string
}

type ExplainDiffResult struct {
	Summary string
	Changes []SemanticChange
	Impact  string
}

type SemanticChange struct {
	Type        string
	Location    string
	Description string
	Significance string
}

// TraceEvent represents a single event in an evaluation trace.
type TraceEvent struct {
	SpanID    string         `json:"span_id"`
	ParentID  string         `json:"parent_id,omitempty"`
	Name      string         `json:"name"`
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"` // span_start | span_end | event | error
	Data      map[string]any `json:"data,omitempty"`
}

// TraceCollector collects evaluation trace events and writes them to JSONL files.
type TraceCollector interface {
	// StartSpan begins a new trace span and returns an updated context with span info.
	StartSpan(ctx context.Context, assetID, snapshotID string) (context.Context, error)

	// RecordEvent records a trace event to the current span.
	RecordEvent(ctx context.Context, event TraceEvent) error

	// Finalize completes the trace and returns the path to the trace file.
	Finalize(ctx context.Context) (string, error)
}

// DeterministicCheck defines a deterministic check to run on trace events.
type DeterministicCheck struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // command_executed, file_exists, json_valid, content_contains, json_path
	Path     string `json:"path,omitempty"`
	Expected string `json:"expected,omitempty"`
	JSONPath string `json:"json_path,omitempty"`
}

// DeterministicResult contains the result of deterministic evaluation.
type DeterministicResult struct {
	Passed  bool     `json:"passed"`
	Score   float64  `json:"score"` // 0.0 - 1.0
	Message string   `json:"message,omitempty"`
	Failed  []string `json:"failed,omitempty"` // IDs of failed checks
}

// Rubric defines the evaluation rubric structure.
type Rubric struct {
	MaxScore int           `json:"max_score"`
	Checks   []RubricCheck `json:"checks"`
}

// RubricCheck defines a single check in the rubric.
type RubricCheck struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
}

// RubricResult contains the result of rubric-based evaluation.
type RubricResult struct {
	Score    int                 `json:"score"`
	MaxScore int                 `json:"max_score"`
	Passed   bool                `json:"passed"`
	Details  []RubricCheckResult `json:"details,omitempty"`
	Message  string              `json:"message,omitempty"`
}

// EvalRunner runs evaluations on prompt outputs.
// Implemented by plugins/eval.
type EvalRunner interface {
	// RunDeterministic runs deterministic checks on trace events.
	RunDeterministic(ctx context.Context, trace []TraceEvent, checks []DeterministicCheck) (DeterministicResult, error)

	// RunRubric runs LLM-based rubric evaluation on output.
	RunRubric(ctx context.Context, output string, rubric Rubric, invoker LLMInvoker, model string) (RubricResult, error)
}

// PromptContent represents the content of a prompt asset.
type PromptContent struct {
	Description string    `json:"description"`
	Instruction string    `json:"instruction"`
	Examples    []Example `json:"examples,omitempty"`
	Variables   []string  `json:"variables,omitempty"`
}

// Example represents a prompt example.
type Example struct {
	Input    string `json:"input"`
	Output   string `json:"output"`
	Footnote string `json:"footnote,omitempty"`
}

// AdaptedPrompt contains the result of prompt adaptation.
type AdaptedPrompt struct {
	Content          string             `json:"content"`
	ParamAdjustments map[string]float64 `json:"param_adjustments,omitempty"`
	FormatChanges    []string           `json:"format_changes,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
}

// ModelParams contains recommended model parameters.
type ModelParams struct {
	Temperature      float64 `json:"temperature"`
	MaxTokens        int     `json:"max_tokens"`
	TopP             float64 `json:"top_p,omitempty"`
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `json:"presence_penalty,omitempty"`
}

// ModelProfile contains the characteristics of a model.
type ModelProfile struct {
	ContextWindow     int     `json:"context_window"`
	InstructionStyle  string  `json:"instruction_style"` // xml_preference | markdown_preference | explicit_preference
	FewShotCapacity   int     `json:"few_shot_capacity"`
	TemperatureCurve  string  `json:"temperature_curve"` // linear | steep | flat
	SystemRoleSupport bool    `json:"system_role_support"`
	JSONReliability   float64 `json:"json_reliability"` // 0.0 - 1.0
}

// ModelAdapter adapts prompts for different models.
// Implemented by plugins/modeladapter.
type ModelAdapter interface {
	// Adapt converts a prompt from source model to target model format.
	Adapt(ctx context.Context, prompt PromptContent, sourceModel, targetModel string) (AdaptedPrompt, error)

	// RecommendParams returns recommended parameters for a target model and task type.
	RecommendParams(ctx context.Context, targetModel string, taskType string) (ModelParams, error)

	// EstimateScore estimates the expected score for a prompt on a target model.
	EstimateScore(ctx context.Context, promptID string, targetModel string) (float64, error)

	// GetModelProfile returns the characteristics of a model.
	GetModelProfile(ctx context.Context, model string) (ModelProfile, error)
}
