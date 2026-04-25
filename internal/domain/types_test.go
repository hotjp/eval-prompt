package domain

import (
	"testing"
	"time"

	"github.com/oklog/ulid"
	"github.com/stretchr/testify/require"
)

func TestNewULID(t *testing.T) {
	id := NewULID()
	require.NotEmpty(t, id)
	require.Len(t, id, 26)

	// Verify it's valid ULID
	_, err := ulid.Parse(id)
	require.NoError(t, err)
}

func TestNewULIDWithTime(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	id := NewULIDWithTime(t1)
	require.NotEmpty(t, id)

	// Verify it's valid ULID
	parsed, err := ulid.Parse(id)
	require.NoError(t, err)

	// Verify timestamp matches
	ts := parsed.Time()
	// Time() returns Unix milliseconds
	require.Equal(t, t1.Unix()*1000, int64(ts))
}

func TestParseULID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid ULID",
			input:   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			wantErr: false,
		},
		{
			name:    "invalid ULID",
			input:   "not-a-ulid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseULID(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValidULID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid ULID",
			input: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			want:  true,
		},
		{
			name:  "invalid ULID",
			input: "not-a-ulid",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidULID(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestID(t *testing.T) {
	t.Run("NewID valid", func(t *testing.T) {
		validULID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
		id, err := NewID(validULID)
		require.NoError(t, err)
		require.Equal(t, validULID, id.String())
	})

	t.Run("NewID invalid", func(t *testing.T) {
		_, err := NewID("invalid")
		require.Error(t, err)
	})

	t.Run("MustNewID valid", func(t *testing.T) {
		validULID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
		id := MustNewID(validULID)
		require.Equal(t, validULID, id.String())
	})

	t.Run("MustNewID invalid panics", func(t *testing.T) {
		require.Panics(t, func() {
			MustNewID("invalid")
		})
	})

	t.Run("NewAutoID", func(t *testing.T) {
		id := NewAutoID()
		require.NotEmpty(t, id.String())
		require.Len(t, id.String(), 26)
	})

	t.Run("IsEmpty", func(t *testing.T) {
		var empty ID
		require.True(t, empty.IsEmpty())

		validID, _ := NewID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
		require.False(t, validID.IsEmpty())
	})

	t.Run("Equal", func(t *testing.T) {
		id1, _ := NewID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
		id2, _ := NewID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
		id3, _ := NewID("01ARZ3NDEKTSV4RRFFQ69G5FBB")

		require.True(t, id1.Equal(id2))
		require.False(t, id1.Equal(id3))
	})
}

func TestIDs(t *testing.T) {
	id1, _ := NewID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	id2, _ := NewID("01ARZ3NDEKTSV4RRFFQ69G5FBB")
	id3, _ := NewID("01ARZ3NDEKTSV4RRFFQ69G5FCC")

	ids := IDs{id1, id2}

	t.Run("Contains", func(t *testing.T) {
		require.True(t, ids.Contains(id1))
		require.True(t, ids.Contains(id2))
		require.False(t, ids.Contains(id3))
	})

	t.Run("StringSlice", func(t *testing.T) {
		slice := ids.StringSlice()
		require.Equal(t, []string{id1.String(), id2.String()}, slice)
	})
}

func TestTime(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dt := NewTime(t1)

	t.Run("Unix", func(t *testing.T) {
		require.Equal(t, t1.Unix(), dt.Unix())
	})

	t.Run("Time", func(t *testing.T) {
		require.Equal(t, t1, dt.Time())
	})

	t.Run("String", func(t *testing.T) {
		require.Equal(t, t1.Format(time.RFC3339), dt.String())
	})

	t.Run("Before/After", func(t *testing.T) {
		t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		dt2 := NewTime(t2)

		require.True(t, dt.Before(dt2))
		require.True(t, dt2.After(dt))
	})

	t.Run("Now", func(t *testing.T) {
		now := Now()
		require.NotZero(t, now.Unix())
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		data, err := dt.MarshalJSON()
		require.NoError(t, err)
		require.Contains(t, string(data), "2026-01-01")
	})
}

func TestPage(t *testing.T) {
	tests := []struct {
		name           string
		offset, limit int
		expectedLimit  int
	}{
		{
			name:          "normal values",
			offset:        10,
			limit:         20,
			expectedLimit: 20,
		},
		{
			name:          "limit zero defaults to 20",
			offset:        0,
			limit:         0,
			expectedLimit: 20,
		},
		{
			name:          "limit negative defaults to 20",
			offset:        0,
			limit:         -5,
			expectedLimit: 20,
		},
		{
			name:          "limit over 100 caps to 100",
			offset:        0,
			limit:         200,
			expectedLimit: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := NewPage(tt.offset, tt.limit)
			require.Equal(t, tt.offset, page.Offset)
			require.Equal(t, tt.expectedLimit, page.Limit)
		})
	}

	t.Run("HasMore", func(t *testing.T) {
		page := Page{Offset: 0, Limit: 10}
		require.True(t, page.HasMore(15))
		require.False(t, page.HasMore(10))
		require.False(t, page.HasMore(5))
	})

	t.Run("Next", func(t *testing.T) {
		page := Page{Offset: 0, Limit: 10}
		next := page.Next()
		require.Equal(t, 10, next.Offset)
		require.Equal(t, 10, next.Limit)
	})
}

func TestResult(t *testing.T) {
	items := []string{"a", "b", "c"}
	page := NewPage(0, 10)

	result := NewResult(items, 15, page)

	require.Equal(t, items, result.Items)
	require.Equal(t, 15, result.Total)
	require.Equal(t, page, result.Page)
	require.True(t, result.HasMore) // 0 + 10 < 15 is true
}

func TestEntityState(t *testing.T) {
	tests := []struct {
		state   EntityState
		wantStr string
		wantOk  bool
	}{
		{StateCreated, "created", true},
		{StateActive, "active", true},
		{StateInactive, "inactive", true},
		{StateDeleted, "deleted", true},
		{EntityState(100), "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.wantStr, func(t *testing.T) {
			require.Equal(t, tt.wantStr, tt.state.String())
			require.Equal(t, tt.wantOk, tt.state.IsValid())
		})
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "valid ULID",
			id:      "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			wantErr: false,
		},
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			id:      "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateID(tt.id)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal string",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "with spaces",
			input: "  hello  ",
			want:  "hello",
		},
		{
			name:  "empty",
			input: "   ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidateNotEmpty(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "non-empty",
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "empty",
			value:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			value:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotEmpty("test_field", tt.value)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		min, max int
		wantErr bool
	}{
		{
			name:    "valid length",
			value:   "hello",
			min:     1,
			max:     10,
			wantErr: false,
		},
		{
			name:    "too short",
			value:   "",
			min:     1,
			max:     10,
			wantErr: true,
		},
		{
			name:    "too long",
			value:   "hello world",
			min:     1,
			max:     5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLength("test_field", tt.value, tt.min, tt.max)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLock(t *testing.T) {
	lock := NewLock()
	require.NotNil(t, lock)

	lock.Lock()
	lock.Unlock()
}

func TestLayer_String(t *testing.T) {
	tests := []struct {
		layer Layer
		want  string
	}{
		{Layer1, "L1"},
		{Layer2, "L2"},
		{Layer3, "L3"},
		{Layer4, "L4"},
		{Layer5, "L5"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, tt.layer.String())
		})
	}
}

func TestErrorCode_Layer(t *testing.T) {
	code := ErrorCode{Layer: Layer2, Sequence: 201}
	require.Equal(t, Layer2, code.Layer)
	require.Equal(t, 201, code.Sequence)
}

func TestParseErrorCode_InvalidSequences(t *testing.T) {
	_, err := ParseErrorCode("L2abc")
	require.Error(t, err)

	// L2000 parses successfully as Layer 2, Sequence 0 (valid since seq 0 is ErrDomainBase)
	code, err := ParseErrorCode("L2000")
	require.NoError(t, err)
	require.Equal(t, Layer2, code.Layer)
	require.Equal(t, 0, code.Sequence)
}
