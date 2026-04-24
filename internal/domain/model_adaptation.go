package domain

import (
	"fmt"
	"time"
)

// ModelAdaptation represents an adaptation of a prompt for a different model.
type ModelAdaptation struct {
	ID               ID
	PromptID         ID
	SourceModel      string
	TargetModel      string
	AdaptedContent   string
	ParamAdjustments map[string]float64
	FormatChanges    []string
	EvalScore        float64
	EvalRunID        ID
	CreatedAt        time.Time
}

// NewModelAdaptation creates a new ModelAdaptation.
func NewModelAdaptation(promptID ID, sourceModel, targetModel, adaptedContent string) *ModelAdaptation {
	return &ModelAdaptation{
		ID:               NewAutoID(),
		PromptID:         promptID,
		SourceModel:      sourceModel,
		TargetModel:      targetModel,
		AdaptedContent:   adaptedContent,
		ParamAdjustments: make(map[string]float64),
		FormatChanges:    []string{},
		CreatedAt:        time.Now(),
	}
}

// Validate validates the model adaptation.
func (m *ModelAdaptation) Validate() error {
	if m.ID.IsEmpty() {
		return ErrInvalidID(m.ID.String())
	}
	if m.PromptID.IsEmpty() {
		return fmt.Errorf("prompt_id is required")
	}
	if m.SourceModel == "" {
		return fmt.Errorf("source_model is required")
	}
	if m.TargetModel == "" {
		return fmt.Errorf("target_model is required")
	}
	if m.AdaptedContent == "" {
		return fmt.Errorf("adapted_content is required")
	}
	return nil
}

// ModelAdaptationSummary is a lightweight representation of a model adaptation.
type ModelAdaptationSummary struct {
	ID          ID
	PromptID    ID
	SourceModel string
	TargetModel string
	EvalScore   float64
	CreatedAt   time.Time
}
