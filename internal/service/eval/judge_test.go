package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "simple number",
			input:   "7.5",
			want:    7.5,
			wantErr: false,
		},
		{
			name:    "decimal starting with zero",
			input:   "0.82",
			want:    0.82,
			wantErr: false,
		},
		{
			name:    "number with prefix Score",
			input:   "Score: 8.3/10",
			want:    8.3,
			wantErr: false,
		},
		{
			name:    "number with whitespace",
			input:   "  9.5  ",
			want:    9.5,
			wantErr: false,
		},
		{
			name:    "integer",
			input:   "10",
			want:    10.0,
			wantErr: false,
		},
		{
			name:    "invalid string",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "only whitespace",
			input:   "   ",
			want:    0,
			wantErr: true,
		},
		{
			name:    "text without number",
			input:   "no numbers here",
			want:    0,
			wantErr: true,
		},
		{
			name:    "newline prefix",
			input:   "\n7.5",
			want:    7.5,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFloat(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.want, got, 0.001)
			}
		})
	}
}
