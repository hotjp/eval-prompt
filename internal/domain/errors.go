package domain

import (
	"fmt"
	"net/http"
)

// Layer represents the architectural layer for error code classification.
type Layer int

const (
	// Layer1 represents the Storage layer (L1).
	Layer1 Layer = 1
	// Layer2 represents the Domain layer (L2).
	Layer2 Layer = 2
	// Layer3 represents the Authz layer (L3).
	Layer3 Layer = 3
	// Layer4 represents the Service layer (L4).
	Layer4 Layer = 4
	// Layer5 represents the Gateway layer (L5).
	Layer5 Layer = 5
)

// String returns the layer identifier (L1, L2, etc.).
func (l Layer) String() string {
	return fmt.Sprintf("L%d", l)
}

// ErrorCode represents a domain error code with layer and sequence number.
// Format: L{layer}{3-digit-sequence}
// Range: L1=[001,199], L2=[200,399], L3=[400,599], L4=[600,799], L5=[800,999]
type ErrorCode struct {
	Layer    Layer
	Sequence int
}

// String returns the formatted error code string.
func (e ErrorCode) String() string {
	return fmt.Sprintf("L%d%03d", e.Layer, e.Sequence)
}

// ParseErrorCode parses an error code string into an ErrorCode.
func ParseErrorCode(s string) (ErrorCode, error) {
	if len(s) < 4 {
		return ErrorCode{}, fmt.Errorf("invalid error code format: %s", s)
	}

	var layer Layer
	var seq int

	_, err := fmt.Sscanf(s[:2], "L%d", &layer)
	if err != nil {
		return ErrorCode{}, fmt.Errorf("invalid layer in error code: %s", s)
	}

	_, err = fmt.Sscanf(s[2:], "%03d", &seq)
	if err != nil {
		return ErrorCode{}, fmt.Errorf("invalid sequence in error code: %s", s)
	}

	return ErrorCode{Layer: layer, Sequence: seq}, nil
}

// Common error codes for L2-Domain layer (200-399).
var (
	// ErrDomainBase is the base error code for domain errors.
	ErrDomainBase = ErrorCode{Layer: Layer2, Sequence: 0}

	// ErrInvalidEntityID indicates an invalid entity ID format.
	ErrInvalidEntityID = ErrorCode{Layer: Layer2, Sequence: 201}
	// ErrEntityNotFound indicates that an entity was not found.
	ErrEntityNotFound = ErrorCode{Layer: Layer2, Sequence: 202}
	// ErrInvalidStateTransition indicates an invalid state transition.
	ErrInvalidStateTransition = ErrorCode{Layer: Layer2, Sequence: 203}
	// ErrAggregateNotFound indicates that an aggregate was not found.
	ErrAggregateNotFound = ErrorCode{Layer: Layer2, Sequence: 204}
	// ErrConcurrencyConflict indicates a version conflict.
	ErrConcurrencyConflict = ErrorCode{Layer: Layer2, Sequence: 205}
	// ErrDomainRuleViolation indicates a business rule violation.
	ErrDomainRuleViolation = ErrorCode{Layer: Layer2, Sequence: 206}
	// ErrInvalidEvent indicates an invalid domain event.
	ErrInvalidEvent = ErrorCode{Layer: Layer2, Sequence: 207}
	// ErrEventOutOfOrder indicates events are out of order.
	ErrEventOutOfOrder = ErrorCode{Layer: Layer2, Sequence: 208}

	// L2 209-299: Asset-related errors
	// ErrAssetNameEmpty indicates the asset name is empty.
	ErrAssetNameEmpty = ErrorCode{Layer: Layer2, Sequence: 209}
	// ErrAssetNameTooLong indicates the asset name exceeds the maximum length.
	ErrAssetNameTooLong = ErrorCode{Layer: Layer2, Sequence: 210}
	// ErrAssetContentHashMismatch indicates content hash mismatch.
	ErrAssetContentHashMismatch = ErrorCode{Layer: Layer2, Sequence: 211}
	// ErrAssetFilePathInvalid indicates an invalid file path.
	ErrAssetFilePathInvalid = ErrorCode{Layer: Layer2, Sequence: 212}
	// ErrAssetStateTransition indicates an invalid asset state transition.
	ErrAssetStateTransition = ErrorCode{Layer: Layer2, Sequence: 213}

	// L2 220-239: Snapshot-related errors
	// ErrSnapshotVersionInvalid indicates an invalid version format.
	ErrSnapshotVersionInvalid = ErrorCode{Layer: Layer2, Sequence: 220}
	// ErrSnapshotNotFound indicates a snapshot was not found.
	ErrSnapshotNotFound = ErrorCode{Layer: Layer2, Sequence: 221}
	// ErrSnapshotCommitFailed indicates a git commit failed.
	ErrSnapshotCommitFailed = ErrorCode{Layer: Layer2, Sequence: 222}

	// L2 240-259: Label-related errors
	// ErrLabelNameInvalid indicates an invalid label name.
	ErrLabelNameInvalid = ErrorCode{Layer: Layer2, Sequence: 240}
	// ErrLabelNotFound indicates a label was not found.
	ErrLabelNotFound = ErrorCode{Layer: Layer2, Sequence: 241}
	// ErrLabelAlreadyExists indicates a label already exists.
	ErrLabelAlreadyExists = ErrorCode{Layer: Layer2, Sequence: 242}

	// L2 260-279: Eval-related errors
	// ErrEvalCaseNotFound indicates an eval case was not found.
	ErrEvalCaseNotFound = ErrorCode{Layer: Layer2, Sequence: 260}
	// ErrEvalRunNotFound indicates an eval run was not found.
	ErrEvalRunNotFound = ErrorCode{Layer: Layer2, Sequence: 261}
	// ErrEvalThresholdNotMet indicates the eval score is below threshold.
	ErrEvalThresholdNotMet = ErrorCode{Layer: Layer2, Sequence: 262}
	// ErrEvalRubricInvalid indicates an invalid rubric.
	ErrEvalRubricInvalid = ErrorCode{Layer: Layer2, Sequence: 263}

	// L2 280-299: Outbox-related errors
	// ErrOutboxEventNotFound indicates an outbox event was not found.
	ErrOutboxEventNotFound = ErrorCode{Layer: Layer2, Sequence: 280}
	// ErrOutboxEventFailed indicates an outbox event processing failed.
	ErrOutboxEventFailed = ErrorCode{Layer: Layer2, Sequence: 281}
)

// DomainError represents a domain-level error with code, message, and details.
type DomainError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Layer     Layer     `json:"layer"`
	LayerName string    `json:"layer_name"`
}

// Error implements the error interface.
func (e DomainError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ErrorCode returns the error code string.
func (e DomainError) ErrorCode() string {
	return e.Code.String()
}

// HTTPStatus returns the appropriate HTTP status code for the error.
func (e DomainError) HTTPStatus() int {
	switch e.Code.Layer {
	case Layer1:
		return http.StatusInternalServerError
	case Layer2:
		switch e.Code.Sequence {
		case 201, 204, 206, 207, 208:
			return http.StatusBadRequest
		case 202, 205:
			return http.StatusConflict
		default:
			return http.StatusInternalServerError
		}
	case Layer3:
		return http.StatusForbidden
	case Layer4:
		switch e.Code.Sequence {
		case 600:
			return http.StatusBadRequest
		case 601, 602:
			return http.StatusNotFound
		default:
			return http.StatusInternalServerError
		}
	case Layer5:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// IsNotFound returns true if the error indicates a resource was not found.
func (e DomainError) IsNotFound() bool {
	return e.HTTPStatus() == http.StatusNotFound
}

// IsConflict returns true if the error indicates a conflict.
func (e DomainError) IsConflict() bool {
	return e.HTTPStatus() == http.StatusConflict
}

// NewDomainError creates a new domain error with the given code and message.
func NewDomainError(code ErrorCode, message string) DomainError {
	return DomainError{
		Code:      code,
		Message:   message,
		Layer:     code.Layer,
		LayerName: fmt.Sprintf("L%d", code.Layer),
	}
}

// NewDomainErrorWithDetails creates a new domain error with details.
func NewDomainErrorWithDetails(code ErrorCode, message, details string) DomainError {
	return DomainError{
		Code:      code,
		Message:   message,
		Details:   details,
		Layer:     code.Layer,
		LayerName: fmt.Sprintf("L%d", code.Layer),
	}
}

// WrapError wraps an error with domain error information.
func WrapError(err error, code ErrorCode, message string) error {
	if err == nil {
		return nil
	}
	return DomainError{
		Code:      code,
		Message:   message,
		Details:   err.Error(),
		Layer:     code.Layer,
		LayerName: fmt.Sprintf("L%d", code.Layer),
	}
}

// ErrInvalidID creates an invalid entity ID error.
func ErrInvalidID(id string) DomainError {
	return NewDomainErrorWithDetails(
		ErrInvalidEntityID,
		"invalid entity ID format",
		id,
	)
}

// ErrNotFound creates a not found error for the given entity type.
func ErrNotFound(entityType string, id ID) DomainError {
	return NewDomainErrorWithDetails(
		ErrEntityNotFound,
		fmt.Sprintf("%s not found", entityType),
		id.String(),
	)
}

// ErrStateTransition creates an invalid state transition error.
func ErrStateTransition(entityType string, from, to string) DomainError {
	return NewDomainErrorWithDetails(
		ErrInvalidStateTransition,
		fmt.Sprintf("invalid state transition for %s", entityType),
		fmt.Sprintf("%s -> %s", from, to),
	)
}

// ErrConcurrency creates a concurrency conflict error.
func ErrConcurrency(entityType string, id ID, expected, actual int64) DomainError {
	return NewDomainErrorWithDetails(
		ErrConcurrencyConflict,
		fmt.Sprintf("version conflict for %s", entityType),
		fmt.Sprintf("ID=%s expected=%d actual=%d", id.String(), expected, actual),
	)
}

// ErrDomainViolation creates a domain rule violation error.
func ErrDomainViolation(rule, details string) DomainError {
	return NewDomainErrorWithDetails(
		ErrDomainRuleViolation,
		fmt.Sprintf("domain rule violation: %s", rule),
		details,
	)
}
