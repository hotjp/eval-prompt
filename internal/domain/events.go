package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of a domain event.
type EventType string

// Common event types for Prompt Asset Management.
const (
	// Asset events
	EventPromptAssetCreated  EventType = "PromptAssetCreatedV1"
	EventPromptAssetUpdated  EventType = "PromptAssetUpdatedV1"
	EventPromptAssetDeleted  EventType = "PromptAssetDeletedV1"
	EventPromptAssetArchived EventType = "PromptAssetArchivedV1"

	// Snapshot events
	EventSnapshotCommitted EventType = "SnapshotCommittedV1"
	EventSnapshotTagged    EventType = "SnapshotTaggedV1"

	// Label events
	EventLabelCreated  EventType = "LabelCreatedV1"
	EventLabelUpdated  EventType = "LabelUpdatedV1"
	EventLabelPromoted EventType = "LabelPromotedV1"
	EventLabelRemoved  EventType = "LabelRemovedV1"

	// Eval events
	EventEvalStarted   EventType = "EvalStartedV1"
	EventEvalCompleted EventType = "EvalCompletedV1"
	EventEvalFailed    EventType = "EvalFailedV1"

	// Adaptation events
	EventPromptAdapted          EventType = "PromptAdaptedV1"
	EventOptimizationSuggested  EventType = "OptimizationSuggestedV1"
	EventOptimizationApplied    EventType = "OptimizationAppliedV1"
	EventOptimizationDiscarded EventType = "OptimizationDiscardedV1"
)

// EventStatus represents the processing status of an event.
type EventStatus int

const (
	EventStatusPending   EventStatus = iota
	EventStatusProcessed
	EventStatusFailed
)

// String returns the string representation of the event status.
func (s EventStatus) String() string {
	switch s {
	case EventStatusPending:
		return "pending"
	case EventStatusProcessed:
		return "processed"
	case EventStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// DomainEventInterface defines the interface for domain events.
// This is implemented by DomainEvent (struct) defined in domain.go.
type DomainEventInterface interface {
	EventID() ID
	EventType() EventType
	GetAggregateID() ID
	OccurredAt() time.Time
	GetVersion() int64
	ToMap() map[string]interface{}
	Validate() error
}

// NewBaseEvent creates a new base event with generated ULID and current time.
func NewBaseEvent(aggregateType string, aggregateID ID, eventType EventType, version int64) BaseEvent {
	return BaseEvent{
		EventIDValue:     NewAutoID(),
		AggregateType:   aggregateType,
		AggregateID:     aggregateID,
		EventType_:      eventType,
		OccurredAt_:      time.Now(),
		IdempotencyKey:  NewULID(),
		Version_:         version,
	}
}

// BaseEvent provides common fields for all domain events.
// This is embedded by specific event types.
type BaseEvent struct {
	EventIDValue    ID        `json:"event_id"`
	AggregateType   string    `json:"aggregate_type"`
	AggregateID     ID        `json:"aggregate_id"`
	EventType_      EventType `json:"event_type"`
	OccurredAt_      time.Time `json:"occurred_at"`
	IdempotencyKey  string    `json:"idempotency_key"`
	Version_         int64    `json:"version"`
}

// EventID returns the event ID.
func (e BaseEvent) EventID() ID {
	return e.EventIDValue
}

// GetAggregateID returns the aggregate ID.
func (e BaseEvent) GetAggregateID() ID {
	return e.AggregateID
}

// EventType returns the event type.
func (e BaseEvent) EventType() EventType {
	return e.EventType_
}

// OccurredAt returns when the event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.OccurredAt_
}

// GetVersion returns the aggregate version.
func (e BaseEvent) GetVersion() int64 {
	return e.Version_
}

// ToMap converts the base event to a map.
func (e BaseEvent) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"event_id":        e.EventID().String(),
		"aggregate_type":  e.AggregateType,
		"aggregate_id":    e.AggregateID.String(),
		"event_type":      string(e.EventType_),
		"occurred_at":     e.OccurredAt_.Format(time.RFC3339),
		"idempotency_key": e.IdempotencyKey,
		"version":         e.Version_,
	}
}

// Validate validates the base event.
func (e BaseEvent) Validate() error {
	if e.EventID().IsEmpty() {
		return fmt.Errorf("event_id is required")
	}
	if e.AggregateType == "" {
		return fmt.Errorf("aggregate_type is required")
	}
	if e.AggregateID.IsEmpty() {
		return fmt.Errorf("aggregate_id is required")
	}
	if e.EventType_ == "" {
		return fmt.Errorf("event_type is required")
	}
	if e.Version_ < 0 {
		return fmt.Errorf("version must be non-negative")
	}
	return nil
}

// PromptAssetCreatedEvent is raised when a new prompt asset is created.
type PromptAssetCreatedEvent struct {
	BaseEvent
	Name        string   `json:"name"`
	Description string   `json:"description"`
	BizLine     string   `json:"biz_line"`
	Tags        []string `json:"tags"`
	FilePath    string   `json:"file_path"`
}

// NewPromptAssetCreatedEvent creates a new PromptAssetCreatedEvent.
func NewPromptAssetCreatedEvent(assetID ID, name, description, bizLine string, tags []string, filePath string, version int64) *PromptAssetCreatedEvent {
	return &PromptAssetCreatedEvent{
		BaseEvent:   NewBaseEvent("Asset", assetID, EventPromptAssetCreated, version),
		Name:        name,
		Description: description,
		BizLine:     bizLine,
		Tags:        tags,
		FilePath:    filePath,
	}
}

// ToMap converts the event to a map.
func (e *PromptAssetCreatedEvent) ToMap() map[string]interface{} {
	m := e.BaseEvent.ToMap()
	m["name"] = e.Name
	m["description"] = e.Description
	m["biz_line"] = e.BizLine
	m["tags"] = e.Tags
	m["file_path"] = e.FilePath
	return m
}

// Validate validates the event.
func (e *PromptAssetCreatedEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// PromptAssetUpdatedEvent is raised when a prompt asset is updated.
type PromptAssetUpdatedEvent struct {
	BaseEvent
	ContentHash string `json:"content_hash"`
	Reason      string `json:"reason"`
}

// NewPromptAssetUpdatedEvent creates a new PromptAssetUpdatedEvent.
func NewPromptAssetUpdatedEvent(assetID ID, contentHash, reason string, version int64) *PromptAssetUpdatedEvent {
	return &PromptAssetUpdatedEvent{
		BaseEvent:   NewBaseEvent("Asset", assetID, EventPromptAssetUpdated, version),
		ContentHash: contentHash,
		Reason:      reason,
	}
}

// ToMap converts the event to a map.
func (e *PromptAssetUpdatedEvent) ToMap() map[string]interface{} {
	m := e.BaseEvent.ToMap()
	m["content_hash"] = e.ContentHash
	m["reason"] = e.Reason
	return m
}

// Validate validates the event.
func (e *PromptAssetUpdatedEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.ContentHash == "" {
		return fmt.Errorf("content_hash is required")
	}
	return nil
}

// SnapshotCommittedEvent is raised when a new snapshot is committed.
type SnapshotCommittedEvent struct {
	BaseEvent
	SnapshotID  ID     `json:"snapshot_id"`
	Version     string `json:"version"`
	ContentHash string `json:"content_hash"`
	CommitHash  string `json:"commit_hash"`
	Author      string `json:"author"`
	Reason      string `json:"reason"`
}

// NewSnapshotCommittedEvent creates a new SnapshotCommittedEvent.
func NewSnapshotCommittedEvent(assetID, snapshotID ID, version, contentHash, commitHash, author, reason string, aggVersion int64) *SnapshotCommittedEvent {
	return &SnapshotCommittedEvent{
		BaseEvent:   NewBaseEvent("Snapshot", snapshotID, EventSnapshotCommitted, aggVersion),
		SnapshotID:  snapshotID,
		Version:     version,
		ContentHash: contentHash,
		CommitHash:  commitHash,
		Author:      author,
		Reason:      reason,
	}
}

// ToMap converts the event to a map.
func (e *SnapshotCommittedEvent) ToMap() map[string]interface{} {
	m := e.BaseEvent.ToMap()
	m["snapshot_id"] = e.SnapshotID.String()
	m["version"] = e.Version
	m["content_hash"] = e.ContentHash
	m["commit_hash"] = e.CommitHash
	m["author"] = e.Author
	m["reason"] = e.Reason
	return m
}

// Validate validates the event.
func (e *SnapshotCommittedEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.SnapshotID.IsEmpty() {
		return fmt.Errorf("snapshot_id is required")
	}
	if e.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

// LabelPromotedEvent is raised when a label is promoted to a new snapshot.
type LabelPromotedEvent struct {
	BaseEvent
	LabelName    string `json:"label_name"`
	FromVersion  string `json:"from_version,omitempty"`
	ToVersion    string `json:"to_version"`
	EvalScore    int    `json:"eval_score,omitempty"`
}

// NewLabelPromotedEvent creates a new LabelPromotedEvent.
func NewLabelPromotedEvent(assetID ID, labelName, fromVersion, toVersion string, evalScore int, version int64) *LabelPromotedEvent {
	return &LabelPromotedEvent{
		BaseEvent:   NewBaseEvent("Label", assetID, EventLabelPromoted, version),
		LabelName:   labelName,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		EvalScore:   evalScore,
	}
}

// ToMap converts the event to a map.
func (e *LabelPromotedEvent) ToMap() map[string]interface{} {
	m := e.BaseEvent.ToMap()
	m["label_name"] = e.LabelName
	m["from_version"] = e.FromVersion
	m["to_version"] = e.ToVersion
	m["eval_score"] = e.EvalScore
	return m
}

// Validate validates the event.
func (e *LabelPromotedEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.LabelName == "" {
		return fmt.Errorf("label_name is required")
	}
	if e.ToVersion == "" {
		return fmt.Errorf("to_version is required")
	}
	return nil
}

// EvalCompletedEvent is raised when an eval run completes.
type EvalCompletedEvent struct {
	BaseEvent
	EvalRunID           ID     `json:"eval_run_id"`
	EvalCaseID          ID     `json:"eval_case_id"`
	SnapshotID          ID     `json:"snapshot_id"`
	Status              string `json:"status"` // passed, failed
	DeterministicScore  float64 `json:"deterministic_score,omitempty"`
	RubricScore         int     `json:"rubric_score,omitempty"`
	TotalScore          int     `json:"total_score,omitempty"`
	DurationMs          int64   `json:"duration_ms,omitempty"`
}

// NewEvalCompletedEvent creates a new EvalCompletedEvent.
func NewEvalCompletedEvent(evalRunID, evalCaseID, snapshotID ID, status string, detScore float64, rubricScore, totalScore, durationMs int64, version int64) *EvalCompletedEvent {
	return &EvalCompletedEvent{
		BaseEvent:            NewBaseEvent("EvalRun", evalRunID, EventEvalCompleted, version),
		EvalRunID:            evalRunID,
		EvalCaseID:           evalCaseID,
		SnapshotID:           snapshotID,
		Status:               status,
		DeterministicScore:   detScore,
		RubricScore:          int(rubricScore),
		TotalScore:           int(totalScore),
		DurationMs:           durationMs,
	}
}

// ToMap converts the event to a map.
func (e *EvalCompletedEvent) ToMap() map[string]interface{} {
	m := e.BaseEvent.ToMap()
	m["eval_run_id"] = e.EvalRunID.String()
	m["eval_case_id"] = e.EvalCaseID.String()
	m["snapshot_id"] = e.SnapshotID.String()
	m["status"] = e.Status
	m["deterministic_score"] = e.DeterministicScore
	m["rubric_score"] = e.RubricScore
	m["total_score"] = e.TotalScore
	m["duration_ms"] = e.DurationMs
	return m
}

// Validate validates the event.
func (e *EvalCompletedEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.EvalRunID.IsEmpty() {
		return fmt.Errorf("eval_run_id is required")
	}
	if e.Status != "passed" && e.Status != "failed" {
		return fmt.Errorf("status must be 'passed' or 'failed'")
	}
	return nil
}

// PromptAdaptedEvent is raised when a prompt is adapted for a different model.
type PromptAdaptedEvent struct {
	BaseEvent
	SourceModel    string `json:"source_model"`
	TargetModel    string `json:"target_model"`
	AdaptedContent string `json:"adapted_content"`
	ScoreDelta     int    `json:"score_delta,omitempty"`
}

// NewPromptAdaptedEvent creates a new PromptAdaptedEvent.
func NewPromptAdaptedEvent(assetID ID, sourceModel, targetModel, adaptedContent string, scoreDelta int, version int64) *PromptAdaptedEvent {
	return &PromptAdaptedEvent{
		BaseEvent:      NewBaseEvent("ModelAdaptation", assetID, EventPromptAdapted, version),
		SourceModel:    sourceModel,
		TargetModel:    targetModel,
		AdaptedContent: adaptedContent,
		ScoreDelta:     scoreDelta,
	}
}

// ToMap converts the event to a map.
func (e *PromptAdaptedEvent) ToMap() map[string]interface{} {
	m := e.BaseEvent.ToMap()
	m["source_model"] = e.SourceModel
	m["target_model"] = e.TargetModel
	m["adapted_content"] = e.AdaptedContent
	m["score_delta"] = e.ScoreDelta
	return m
}

// Validate validates the event.
func (e *PromptAdaptedEvent) Validate() error {
	if err := e.BaseEvent.Validate(); err != nil {
		return err
	}
	if e.SourceModel == "" {
		return fmt.Errorf("source_model is required")
	}
	if e.TargetModel == "" {
		return fmt.Errorf("target_model is required")
	}
	return nil
}

// EventToJSON serializes a domain event to JSON.
func EventToJSON(event DomainEventInterface) ([]byte, error) {
	m := event.ToMap()
	return json.Marshal(m)
}

// EventFromJSON deserializes a domain event from JSON.
func EventFromJSON(data []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// OutboxEvent represents an event stored in the outbox for reliable delivery.
type OutboxEvent struct {
	ID              EventID     `json:"id"`
	AggregateType   string      `json:"aggregate_type"`
	AggregateID     ID          `json:"aggregate_id"`
	EventType       EventType   `json:"event_type"`
	Payload         []byte      `json:"payload"`
	OccurredAt      time.Time   `json:"occurred_at"`
	IdempotencyKey  string      `json:"idempotency_key"`
	Status          EventStatus `json:"status"`
	RetryCount      int         `json:"retry_count"`
}

// NewOutboxEvent creates a new outbox event from a domain event.
func NewOutboxEvent(event DomainEventInterface) (*OutboxEvent, error) {
	payload, err := EventToJSON(event)
	if err != nil {
		return nil, err
	}

	return &OutboxEvent{
		ID:             event.EventID(),
		AggregateType:  event.GetAggregateID().String(),
		AggregateID:    event.GetAggregateID(),
		EventType:      event.EventType(),
		Payload:        payload,
		OccurredAt:     event.OccurredAt(),
		IdempotencyKey: event.GetAggregateID().String() + "/" + event.EventID().String(),
		Status:         EventStatusPending,
		RetryCount:     0,
	}, nil
}
