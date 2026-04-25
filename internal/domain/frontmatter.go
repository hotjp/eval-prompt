package domain

import (
	"math"
	"time"
)

// EvalHistoryEntry represents an eval run summary stored in YAML front matter.
// This is a lightweight representation for serialization in .md files.
type EvalHistoryEntry struct {
	RunID              string  `yaml:"run_id"`
	SnapshotID         string  `yaml:"snapshot_id"`
	Score              int     `yaml:"score"`
	DeterministicScore float64 `yaml:"deterministic_score"`
	RubricScore        int     `yaml:"rubric_score"`
	Model              string  `yaml:"model"`
	EvalCaseVersion    string  `yaml:"eval_case_version"`
	TokensIn           int     `yaml:"tokens_in"`
	TokensOut          int     `yaml:"tokens_out"`
	DurationMs         int64   `yaml:"duration_ms"`
	Date               string  `yaml:"date"`
	By                 string  `yaml:"by"`
}

// EvalStats map model name to ModelStat.
type EvalStats map[string]ModelStat

// ModelStat holds Welford algorithm parameters for incremental statistics.
type ModelStat struct {
	Count   int     `yaml:"count"`
	Mean    float64 `yaml:"mean"`
	M2      float64 `yaml:"m2"` // Welford algorithm parameter
	Min     float64 `yaml:"min"`
	Max     float64 `yaml:"max"`
	LastRun string  `yaml:"last_run"`
}

// Update updates the statistics with a new score using Welford's online algorithm.
func (s *ModelStat) Update(newScore float64) {
	s.Count++
	delta := newScore - s.Mean
	s.Mean += delta / float64(s.Count)
	delta2 := newScore - s.Mean
	s.M2 += delta * delta2
	if newScore < s.Min || s.Count == 1 {
		s.Min = newScore
	}
	if newScore > s.Max {
		s.Max = newScore
	}
	s.LastRun = time.Now().Format("2006-01-02")
}

// StdDev returns the standard deviation of the scores.
func (s *ModelStat) StdDev() float64 {
	if s.Count < 2 {
		return 0
	}
	return math.Sqrt(s.M2 / float64(s.Count-1))
}

// LabelEntry represents a label stored in YAML front matter.
// This is a lightweight representation for serialization in .md files.
type LabelEntry struct {
	Name     string `yaml:"name"`
	Snapshot string `yaml:"snapshot"`
	Date     string `yaml:"date"`
}

// FrontMatter represents the YAML front matter in a .md prompt file.
// This is the canonical format for storing prompt metadata in the filesystem.
type FrontMatter struct {
	ID                     string            `yaml:"id"`
	Name                   string            `yaml:"name"`
	Description            string            `yaml:"description,omitempty"`
	Version                string            `yaml:"version,omitempty"`
	ContentHash            string            `yaml:"content_hash"`
	State                  string            `yaml:"state"`
	BizLine                string            `yaml:"biz_line,omitempty"`
	Tags                   []string          `yaml:"tags,omitempty"`
	UpdatedAt              time.Time         `yaml:"updated_at,omitempty"`
	EvalHistory            []EvalHistoryEntry `yaml:"eval_history,omitempty"`
	EvalStats              EvalStats         `yaml:"eval_stats,omitempty"`
	Labels                 []LabelEntry       `yaml:"labels,omitempty"`
	RecommendedSnapshotID  string            `yaml:"recommended_snapshot_id,omitempty"`
}

// Validate validates the front matter structure.
func (f *FrontMatter) Validate() error {
	if f.ID == "" {
		return NewDomainError(ErrInvalidEntityID, "front matter id is required")
	}
	// NOTE: ID can be a human-readable name (not just ULID) to match asset naming conventions.
	if f.Name == "" {
		return NewDomainError(ErrAssetNameEmpty, "front matter name is required")
	}
	return nil
}

// HasEvalHistory returns true if there is eval history.
func (f *FrontMatter) HasEvalHistory() bool {
	return len(f.EvalHistory) > 0
}

// HasLabels returns true if there are labels.
func (f *FrontMatter) HasLabels() bool {
	return len(f.Labels) > 0
}

// EvalPromptFrontMatter represents the YAML front matter in an eval prompt .md file.
// This is the canonical format for storing eval prompt metadata in the filesystem.
type EvalPromptFrontMatter struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description,omitempty"`
	Version      string   `yaml:"version,omitempty"`
	ContentHash  string   `yaml:"content_hash"`
	State        string   `yaml:"state"`
	BizLine      string   `yaml:"biz_line,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
	EvalCaseIDs  []string `yaml:"eval_case_ids,omitempty"`
	Model        string   `yaml:"model"`
}

// Validate validates the eval prompt front matter structure.
func (f *EvalPromptFrontMatter) Validate() error {
	if f.ID == "" {
		return NewDomainError(ErrInvalidEntityID, "eval prompt front matter id is required")
	}
	// NOTE: ID can be a human-readable name (not just ULID) to match asset naming conventions.
	if f.Name == "" {
		return NewDomainError(ErrAssetNameEmpty, "eval prompt front matter name is required")
	}
	return nil
}

// HasEvalCaseIDs returns true if there are eval case IDs.
func (f *EvalPromptFrontMatter) HasEvalCaseIDs() bool {
	return len(f.EvalCaseIDs) > 0
}