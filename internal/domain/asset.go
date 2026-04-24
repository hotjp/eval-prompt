package domain

import (
	"time"
)

// Asset represents a prompt asset in the system.
// This is the domain entity for L2-Domain layer.
type Asset struct {
	ID          ID
	Name        string
	Description string
	BizLine     string
	Tags        []string
	ContentHash string
	FilePath    string
	State       State
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int64
}

// Validate validates the asset entity.
func (a *Asset) Validate() error {
	// Validate ID
	if a.ID.IsEmpty() {
		return ErrInvalidID(a.ID.String())
	}

	// Validate Name
	if err := ValidateLength("name", a.Name, 1, 100); err != nil {
		return err
	}

	// Validate ContentHash
	if a.ContentHash == "" {
		return NewDomainError(ErrAssetContentHashMismatch, "content hash is required")
	}

	// Validate FilePath
	if a.FilePath == "" {
		return NewDomainError(ErrAssetFilePathInvalid, "file path is required")
	}

	return nil
}

// CanPromote returns true if the asset can be promoted.
func (a *Asset) CanPromote() bool {
	return a.State == AssetStateEvaluated || a.State == AssetStatePromoted
}

// CanEval returns true if the asset can be evaluated.
func (a *Asset) CanEval() bool {
	return a.State == AssetStateCreated || a.State == AssetStateEvaluated
}

// CanArchive returns true if the asset can be archived.
func (a *Asset) CanArchive() bool {
	return a.State == AssetStateCreated || a.State == AssetStateEvaluated || a.State == AssetStatePromoted
}

// TransitionTo transitions the asset to a new state.
func (a *Asset) TransitionTo(newState State, event EventType) error {
	if !a.canTransitionTo(newState, event) {
		return ErrStateTransition("Asset", string(a.State), string(newState))
	}
	a.State = newState
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}

// canTransitionTo checks if the transition is valid without executing it.
func (a *Asset) canTransitionTo(to State, event EventType) bool {
	// Valid transitions as per DESIGN.md:
	// CREATED --[Eval Started]--> EVALUATING
	// EVALUATING --[Eval Completed]--> EVALUATED
	// EVALUATING --[Eval Failed]--> CREATED
	// EVALUATED --[Label Promoted]--> PROMOTED
	// EVALUATED --[Content Changed]--> CREATED
	// PROMOTED --[Content Changed]--> CREATED
	// CREATED/EVALUATED/PROMOTED --[Archive]--> ARCHIVED

	switch a.State {
	case AssetStateCreated:
		if to == AssetStateEvaluating && event == EventEvalStarted {
			return true
		}
		if to == AssetStateArchived && event == EventPromptAssetArchived {
			return true
		}
	case AssetStateEvaluating:
		if to == AssetStateEvaluated && event == EventEvalCompleted {
			return true
		}
		if to == AssetStateCreated && event == EventEvalFailed {
			return true
		}
	case AssetStateEvaluated:
		if to == AssetStatePromoted && event == EventLabelPromoted {
			return true
		}
		if to == AssetStateCreated && event == EventPromptAssetUpdated {
			return true
		}
		if to == AssetStateArchived && event == EventPromptAssetArchived {
			return true
		}
	case AssetStatePromoted:
		if to == AssetStateCreated && event == EventPromptAssetUpdated {
			return true
		}
		if to == AssetStateArchived && event == EventPromptAssetArchived {
			return true
		}
	}
	return false
}

// NewAsset creates a new Asset with the given parameters.
func NewAsset(name, description, bizLine string, tags []string, contentHash, filePath string) *Asset {
	now := time.Now()
	return &Asset{
		ID:          NewAutoID(),
		Name:        name,
		Description: description,
		BizLine:     bizLine,
		Tags:        tags,
		ContentHash: contentHash,
		FilePath:    filePath,
		State:       AssetStateCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     0,
	}
}

// NewAssetWithID creates a new Asset with a specific ID.
func NewAssetWithID(id ID, name, description, bizLine string, tags []string, contentHash, filePath string) *Asset {
	now := time.Now()
	return &Asset{
		ID:          id,
		Name:        name,
		Description: description,
		BizLine:     bizLine,
		Tags:        tags,
		ContentHash: contentHash,
		FilePath:    filePath,
		State:       AssetStateCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     0,
	}
}

// AssetSummary is a lightweight representation of an asset for listing.
type AssetSummary struct {
	ID        ID
	Name      string
	BizLine   string
	State     State
	UpdatedAt time.Time
}

// AssetDetail is a detailed representation of an asset.
type AssetDetail struct {
	Asset
	SnapshotCount int
	LabelCount    int
	EvalCaseCount int
}
