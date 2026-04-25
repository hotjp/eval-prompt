package domain

// EvalHistoryEntry represents an eval run summary stored in YAML front matter.
// This is a lightweight representation for serialization in .md files.
type EvalHistoryEntry struct {
	RunID       string `yaml:"run_id"`
	SnapshotID  string `yaml:"snapshot_id"`
	Score       int    `yaml:"score"`
	Model       string `yaml:"model"`
	Date        string `yaml:"date"`
	By          string `yaml:"by"`
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
	Tags                   []string          `yaml:"tags,omitempty"`
	EvalHistory            []EvalHistoryEntry `yaml:"eval_history,omitempty"`
	Labels                 []LabelEntry       `yaml:"labels,omitempty"`
	RecommendedSnapshotID  string            `yaml:"recommended_snapshot_id,omitempty"`
}

// Validate validates the front matter structure.
func (f *FrontMatter) Validate() error {
	if f.ID == "" {
		return NewDomainError(ErrInvalidEntityID, "front matter id is required")
	}
	if !IsValidULID(f.ID) {
		return ErrInvalidID(f.ID)
	}
	if f.Name == "" {
		return NewDomainError(ErrAssetNameEmpty, "front matter name is required")
	}
	if f.ContentHash == "" {
		return NewDomainError(ErrAssetContentHashMismatch, "content_hash is required")
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
	Tags         []string `yaml:"tags,omitempty"`
	EvalCaseIDs  []string `yaml:"eval_case_ids,omitempty"`
	Model        string   `yaml:"model"`
}

// Validate validates the eval prompt front matter structure.
func (f *EvalPromptFrontMatter) Validate() error {
	if f.ID == "" {
		return NewDomainError(ErrInvalidEntityID, "eval prompt front matter id is required")
	}
	if !IsValidULID(f.ID) {
		return ErrInvalidID(f.ID)
	}
	if f.Name == "" {
		return NewDomainError(ErrAssetNameEmpty, "eval prompt front matter name is required")
	}
	if f.ContentHash == "" {
		return NewDomainError(ErrAssetContentHashMismatch, "content_hash is required")
	}
	return nil
}

// HasEvalCaseIDs returns true if there are eval case IDs.
func (f *EvalPromptFrontMatter) HasEvalCaseIDs() bool {
	return len(f.EvalCaseIDs) > 0
}