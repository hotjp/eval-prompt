package domain

import (
	"fmt"
	"time"
)

// Label represents a named pointer to a Snapshot (e.g., "prod", "dev", "staging").
type Label struct {
	ID         ID
	AssetID    ID
	Name       string // prod, dev, staging
	SnapshotID ID
	UpdatedAt  time.Time
}

// LabelName constants for common label types.
const (
	LabelNameProd    = "prod"
	LabelNameDev     = "dev"
	LabelNameStaging = "staging"
)

// Validate validates the label entity.
func (l *Label) Validate() error {
	if l.ID.IsEmpty() {
		return ErrInvalidID(l.ID.String())
	}
	if l.AssetID.IsEmpty() {
		return fmt.Errorf("asset_id is required")
	}
	if l.Name == "" {
		return NewDomainError(ErrLabelNameInvalid, "label name is required")
	}
	if l.SnapshotID.IsEmpty() {
		return fmt.Errorf("snapshot_id is required")
	}
	return nil
}

// NewLabel creates a new Label.
func NewLabel(assetID, snapshotID ID, name string) *Label {
	return &Label{
		ID:         NewAutoID(),
		AssetID:    assetID,
		Name:       name,
		SnapshotID: snapshotID,
		UpdatedAt:  time.Now(),
	}
}

// IsProd returns true if this is the production label.
func (l *Label) IsProd() bool {
	return l.Name == LabelNameProd
}

// IsDev returns true if this is the development label.
func (l *Label) IsDev() bool {
	return l.Name == LabelNameDev
}

// IsStaging returns true if this is the staging label.
func (l *Label) IsStaging() bool {
	return l.Name == LabelNameStaging
}
