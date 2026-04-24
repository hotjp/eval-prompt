package domain

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid"
)

// ULID entropy source - uses crypto/rand for secure randomness
var ulidEntropy = ulid.Monotonic(rand.Reader, 0)

// NewULID generates a new ULID (Universally Unique Lexicographically Sortable Identifier).
// ULIDs are time-ordered, 26 characters base32 encoded, and safe to use as IDs.
func NewULID() string {
	return ulid.MustNew(ulid.Now(), ulidEntropy).String()
}

// NewULIDWithTime generates a ULID with a specific timestamp.
// Useful for testing or when the timestamp is known.
func NewULIDWithTime(t time.Time) string {
	return ulid.MustNew(ulid.Timestamp(t), ulidEntropy).String()
}

// ParseULID parses a string into a ULID value.
// Returns an error if the string is not a valid ULID.
func ParseULID(s string) (ulid.ULID, error) {
	return ulid.Parse(s)
}

// IsValidULID checks if a string is a valid ULID format.
func IsValidULID(s string) bool {
	_, err := ulid.Parse(s)
	return err == nil
}

// ID represents a domain entity identifier based on ULID.
type ID struct {
	value string
}

// NewID creates a new domain ID from a ULID string.
func NewID(value string) (ID, error) {
	if !IsValidULID(value) {
		return ID{}, fmt.Errorf("invalid ULID: %s", value)
	}
	return ID{value: value}, nil
}

// MustNewID creates a new domain ID and panics if the value is invalid.
func MustNewID(value string) ID {
	id, err := NewID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// NewAutoID creates a new domain ID with an auto-generated ULID.
func NewAutoID() ID {
	return ID{value: NewULID()}
}

// String returns the string representation of the ID.
func (id ID) String() string {
	return id.value
}

// IsEmpty returns true if the ID is empty (zero value).
func (id ID) IsEmpty() bool {
	return id.value == ""
}

// Equal returns true if two IDs have the same value.
func (id ID) Equal(other ID) bool {
	return id.value == other.value
}

// IDs is a collection of domain IDs.
type IDs []ID

// Contains checks if the collection contains a specific ID.
func (ids IDs) Contains(id ID) bool {
	for _, other := range ids {
		if id.Equal(other) {
			return true
		}
	}
	return false
}

// StringSlice converts IDs to a string slice.
func (ids IDs) StringSlice() []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}

// Time represents a domain time value.
type Time struct {
	value time.Time
}

// NewTime creates a new domain Time from a time.Time.
func NewTime(t time.Time) Time {
	return Time{value: t}
}

// Now returns the current time.
func Now() Time {
	return Time{value: time.Now()}
}

// Unix returns the Unix timestamp.
func (t Time) Unix() int64 {
	return t.value.Unix()
}

// Time returns the underlying time.Time.
func (t Time) Time() time.Time {
	return t.value
}

// String returns the ISO 8601 formatted string.
func (t Time) String() string {
	return t.value.Format(time.RFC3339)
}

// Before returns true if t is before other.
func (t Time) Before(other Time) bool {
	return t.value.Before(other.value)
}

// After returns true if t is after other.
func (t Time) After(other Time) bool {
	return t.value.After(other.value)
}

// MarshalJSON implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

// Page represents pagination parameters.
type Page struct {
	Offset int
	Limit  int
}

// NewPage creates a new page with offset and limit.
func NewPage(offset, limit int) Page {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return Page{Offset: offset, Limit: limit}
}

// HasMore returns true if there are more pages after this one.
func (p Page) HasMore(total int) bool {
	return p.Offset+p.Limit < total
}

// Next returns the next page.
func (p Page) Next() Page {
	return Page{Offset: p.Offset + p.Limit, Limit: p.Limit}
}

// Result is a generic container for a paginated result.
type Result[T any] struct {
	Items   []T
	Total   int
	Page    Page
	HasMore bool
}

// NewResult creates a new result with items and pagination info.
func NewResult[T any](items []T, total int, page Page) Result[T] {
	return Result[T]{
		Items:   items,
		Total:   total,
		Page:    page,
		HasMore: page.HasMore(total),
	}
}

// Lock provides a simple mutual exclusion mechanism.
type Lock struct {
	mu sync.Mutex
}

// NewLock creates a new lock.
func NewLock() *Lock {
	return &Lock{}
}

// Lock acquires the lock.
func (l *Lock) Lock() {
	l.mu.Lock()
}

// Unlock releases the lock.
func (l *Lock) Unlock() {
	l.mu.Unlock()
}

// Reader is a generic interface for readers.
type Reader[T any] interface {
	Read(id ID) (T, error)
	List(page Page) (Result[T], error)
}

// Writer is a generic interface for writers.
type Writer[T any] interface {
	Create(entity *T) error
	Update(entity *T) error
	Delete(id ID) error
}

// Repository is a generic interface combining Reader and Writer.
type Repository[T any] interface {
	Reader[T]
	Writer[T]
}

// EventID is a type alias for ULID string for event IDs.
type EventID = ID

// AggregateID is a type alias for ULID string for aggregate IDs.
type AggregateID = ID

// EntityState represents the state of a domain entity.
type EntityState int

const (
	StateCreated EntityState = iota
	StateActive
	StateInactive
	StateDeleted
)

// String returns the string representation of the state.
func (s EntityState) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateActive:
		return "active"
	case StateInactive:
		return "inactive"
	case StateDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// IsValid returns true if the state is valid.
func (s EntityState) IsValid() bool {
	return s >= StateCreated && s <= StateDeleted
}

// ValidateID validates an entity ID format.
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if !IsValidULID(id) {
		return fmt.Errorf("ID must be a valid ULID: %s", id)
	}
	return nil
}

// SanitizeString sanitizes a string by trimming whitespace and removing control characters.
func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}

// ValidateNotEmpty validates that a string is not empty.
func ValidateNotEmpty(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}
	return nil
}

// ValidateLength validates that a string length is within bounds.
func ValidateLength(name, value string, min, max int) error {
	if len(value) < min {
		return fmt.Errorf("%s must be at least %d characters", name, min)
	}
	if len(value) > max {
		return fmt.Errorf("%s must be at most %d characters", name, max)
	}
	return nil
}
