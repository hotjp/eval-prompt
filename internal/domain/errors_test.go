package domain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorCode(t *testing.T) {
	t.Run("String format", func(t *testing.T) {
		code := ErrorCode{Layer: Layer2, Sequence: 201}
		require.Equal(t, "L2201", code.String())
	})

	t.Run("Parse valid no separator", func(t *testing.T) {
		code, err := ParseErrorCode("L2201")
		require.NoError(t, err)
		require.Equal(t, Layer2, code.Layer)
		require.Equal(t, 201, code.Sequence)
	})

	t.Run("Parse valid zero padding", func(t *testing.T) {
		code, err := ParseErrorCode("L3001")
		require.NoError(t, err)
		require.Equal(t, Layer3, code.Layer)
		require.Equal(t, 1, code.Sequence)
	})

	t.Run("Parse invalid too short", func(t *testing.T) {
		_, err := ParseErrorCode("L2")
		require.Error(t, err)
	})

	t.Run("Parse invalid format", func(t *testing.T) {
		_, err := ParseErrorCode("invalid")
		require.Error(t, err)
	})

	t.Run("Parse invalid layer", func(t *testing.T) {
		_, err := ParseErrorCode("LX_201")
		require.Error(t, err)
	})
}

func TestDomainError(t *testing.T) {
	t.Run("Error with message", func(t *testing.T) {
		err := NewDomainError(ErrInvalidEntityID, "invalid ID format")
		require.Equal(t, "L2201: invalid ID format", err.Error())
	})

	t.Run("Error with details", func(t *testing.T) {
		err := NewDomainErrorWithDetails(ErrInvalidEntityID, "invalid ID format", "abc123")
		require.Equal(t, "L2201: invalid ID format (abc123)", err.Error())
	})

	t.Run("ErrorCode returns string", func(t *testing.T) {
		err := NewDomainError(ErrInvalidEntityID, "invalid ID format")
		require.Equal(t, "L2201", err.ErrorCode())
	})

	t.Run("HTTPStatus for Layer2", func(t *testing.T) {
		// Bad request cases (201, 204, 206, 207, 208)
		require.Equal(t, http.StatusBadRequest, NewDomainError(ErrInvalidEntityID, "").HTTPStatus())
		require.Equal(t, http.StatusBadRequest, NewDomainError(ErrDomainRuleViolation, "").HTTPStatus())

		// Conflict cases (202, 205)
		require.Equal(t, http.StatusConflict, NewDomainError(ErrEntityNotFound, "").HTTPStatus())
		require.Equal(t, http.StatusConflict, NewDomainError(ErrConcurrencyConflict, "").HTTPStatus())

		// Default: internal server error (e.g. 203, 209, etc)
		require.Equal(t, http.StatusInternalServerError, NewDomainError(ErrInvalidStateTransition, "").HTTPStatus())
	})

	t.Run("HTTPStatus for Layer3", func(t *testing.T) {
		err := DomainError{Code: ErrorCode{Layer: Layer3, Sequence: 400}}
		require.Equal(t, http.StatusForbidden, err.HTTPStatus())
	})

	t.Run("IsNotFound", func(t *testing.T) {
		// L2 errors don't return 404, so IsNotFound is false for L2 domain errors
		require.False(t, NewDomainError(ErrEntityNotFound, "").IsNotFound())
		require.False(t, NewDomainError(ErrInvalidEntityID, "").IsNotFound())
		// L4 errors 601, 602 return 404
		l4NotFound := DomainError{Code: ErrorCode{Layer: Layer4, Sequence: 601}}
		require.True(t, l4NotFound.IsNotFound())
	})

	t.Run("IsConflict", func(t *testing.T) {
		require.True(t, NewDomainError(ErrConcurrencyConflict, "").IsConflict())
		require.False(t, NewDomainError(ErrInvalidEntityID, "").IsConflict())
	})
}

func TestWrapError(t *testing.T) {
	t.Run("wrap nil returns nil", func(t *testing.T) {
		result := WrapError(nil, ErrInvalidEntityID, "test")
		require.Nil(t, result)
	})

	t.Run("wrap actual error", func(t *testing.T) {
		original := NewDomainError(ErrInvalidEntityID, "original error")
		wrapped := WrapError(original, ErrEntityNotFound, "wrapped")
		require.Contains(t, wrapped.Error(), "wrapped")
		require.Contains(t, wrapped.Error(), "original error")
	})
}

func TestErrInvalidID(t *testing.T) {
	err := ErrInvalidID("invalid-id")
	require.Equal(t, "L2201", err.ErrorCode())
	require.Contains(t, err.Error(), "invalid entity ID format")
	require.Contains(t, err.Error(), "invalid-id")
}

func TestErrNotFound(t *testing.T) {
	id := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	err := ErrNotFound("Asset", id)
	require.Equal(t, "L2202", err.ErrorCode())
	require.Contains(t, err.Error(), "Asset not found")
}

func TestErrStateTransition(t *testing.T) {
	err := ErrStateTransition("Asset", "CREATED", "PROMOTED")
	require.Equal(t, "L2203", err.ErrorCode())
	require.Contains(t, err.Error(), "invalid state transition for Asset")
	require.Contains(t, err.Error(), "CREATED -> PROMOTED")
}

func TestErrConcurrency(t *testing.T) {
	id := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	err := ErrConcurrency("Asset", id, 1, 2)
	require.Equal(t, "L2205", err.ErrorCode())
	require.Contains(t, err.Error(), "version conflict for Asset")
	require.Contains(t, err.Error(), "expected=1 actual=2")
}

func TestErrDomainViolation(t *testing.T) {
	err := ErrDomainViolation("test_rule", "test details")
	require.Equal(t, "L2206", err.ErrorCode())
	require.Contains(t, err.Error(), "domain rule violation: test_rule")
	require.Contains(t, err.Error(), "test details")
}
