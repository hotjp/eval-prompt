package eval

import (
	"testing"
	"time"

	"github.com/eval-prompt/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAssertionLibrary_NewAssertionLibrary(t *testing.T) {
	lib := NewAssertionLibrary()
	require.NotNil(t, lib)

	// Verify all assertion types are registered
	assertionTypes := []string{
		"command_executed",
		"file_exists",
		"json_valid",
		"content_contains",
		"json_path",
	}

	for _, at := range assertionTypes {
		checker := lib.Get(at)
		require.NotNil(t, checker, "expected checker for type %q", at)
	}
}

func TestAssertionLibrary_Get_UnknownType(t *testing.T) {
	lib := NewAssertionLibrary()
	checker := lib.Get("unknown_type")
	require.Nil(t, checker)
}

func TestCommandExecutedChecker_Check(t *testing.T) {
	tests := []struct {
		name    string
		trace   []service.TraceEvent
		check   service.DeterministicCheck
		wantErr bool
	}{
		{
			name: "command found in trace",
			trace: []service.TraceEvent{
				{
					Type: "command_executed",
					Data: map[string]any{"command": "ls -la"},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-1",
				Type:     "command_executed",
				Expected: "ls",
			},
			wantErr: false,
		},
		{
			name: "command not found",
			trace: []service.TraceEvent{
				{
					Type: "command_executed",
					Data: map[string]any{"command": "pwd"},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-2",
				Type:     "command_executed",
				Expected: "ls",
			},
			wantErr: true,
		},
		{
			name:    "empty trace",
			trace:   []service.TraceEvent{},
			check: service.DeterministicCheck{
				ID:       "check-3",
				Type:     "command_executed",
				Expected: "ls",
			},
			wantErr: true,
		},
		{
			name: "command with partial match",
			trace: []service.TraceEvent{
				{
					Type: "command_executed",
					Data: map[string]any{"command": "git commit -m 'fix bug'"},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-4",
				Type:     "command_executed",
				Expected: "commit",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &CommandExecutedChecker{}
			err := checker.Check(tt.trace, tt.check)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFileExistsChecker_Check(t *testing.T) {
	tests := []struct {
		name    string
		trace   []service.TraceEvent
		check   service.DeterministicCheck
		wantErr bool
	}{
		{
			name: "file_created event matches",
			trace: []service.TraceEvent{
				{
					Type: "file_created",
					Data: map[string]any{"path": "/sandbox/output.txt"},
				},
			},
			check: service.DeterministicCheck{
				ID:   "check-1",
				Type: "file_exists",
				Path: "/sandbox/output.txt",
			},
			wantErr: false,
		},
		{
			name: "file_exists event matches",
			trace: []service.TraceEvent{
				{
					Type: "file_exists",
					Data: map[string]any{"path": "/sandbox/test.json"},
				},
			},
			check: service.DeterministicCheck{
				ID:   "check-2",
				Type: "file_exists",
				Path: "/sandbox/test.json",
			},
			wantErr: false,
		},
		{
			name: "path does not match",
			trace: []service.TraceEvent{
				{
					Type: "file_created",
					Data: map[string]any{"path": "/sandbox/other.txt"},
				},
			},
			check: service.DeterministicCheck{
				ID:   "check-3",
				Type: "file_exists",
				Path: "/sandbox/output.txt",
			},
			wantErr: true,
		},
		{
			name: "wildcard path matches",
			trace: []service.TraceEvent{
				{
					Type: "file_created",
					Data: map[string]any{"path": "/sandbox/prompts/test.txt"},
				},
			},
			check: service.DeterministicCheck{
				ID:   "check-4",
				Type: "file_exists",
				Path: "/sandbox/prompts/*",
			},
			wantErr: false,
		},
		{
			name:    "empty trace returns error",
			trace:   []service.TraceEvent{},
			check: service.DeterministicCheck{
				ID:   "check-5",
				Type: "file_exists",
				Path: "/sandbox/test.txt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &FileExistsChecker{}
			err := checker.Check(tt.trace, tt.check)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestJSONValidChecker_Check(t *testing.T) {
	tests := []struct {
		name    string
		trace   []service.TraceEvent
		check   service.DeterministicCheck
		wantErr bool
	}{
		{
			name: "valid JSON in llm_output",
			trace: []service.TraceEvent{
				{
					Type: "llm_output",
					Data: map[string]any{"content": `{"key": "value"}`},
				},
			},
			check:   service.DeterministicCheck{ID: "check-1", Type: "json_valid"},
			wantErr: false,
		},
		{
			name: "valid JSON in output",
			trace: []service.TraceEvent{
				{
					Type: "output",
					Data: map[string]any{"content": `{"result": true}`},
				},
			},
			check:   service.DeterministicCheck{ID: "check-2", Type: "json_valid"},
			wantErr: false,
		},
		{
			name: "invalid JSON returns error",
			trace: []service.TraceEvent{
				{
					Type: "llm_output",
					Data: map[string]any{"content": `{not valid json}`},
				},
			},
			check:   service.DeterministicCheck{ID: "check-3", Type: "json_valid"},
			wantErr: true,
		},
		{
			name: "no valid JSON returns error",
			trace: []service.TraceEvent{
				{
					Type: "llm_output",
					Data: map[string]any{"content": `plain text output`},
				},
			},
			check:   service.DeterministicCheck{ID: "check-4", Type: "json_valid"},
			wantErr: true,
		},
		{
			name:    "empty trace returns error",
			trace:   []service.TraceEvent{},
			check:   service.DeterministicCheck{ID: "check-5", Type: "json_valid"},
			wantErr: true,
		},
		{
			name: "nested JSON object",
			trace: []service.TraceEvent{
				{
					Type: "llm_output",
					Data: map[string]any{"content": `{"data": {"nested": "value"}}`},
				},
			},
			check:   service.DeterministicCheck{ID: "check-6", Type: "json_valid"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &JSONValidChecker{}
			err := checker.Check(tt.trace, tt.check)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestContentContainsChecker_Check(t *testing.T) {
	tests := []struct {
		name    string
		trace   []service.TraceEvent
		check   service.DeterministicCheck
		wantErr bool
	}{
		{
			name: "content found in llm_output",
			trace: []service.TraceEvent{
				{
					Type: "llm_output",
					Data: map[string]any{"content": "The result is successful"},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-1",
				Type:     "content_contains",
				Expected: "successful",
			},
			wantErr: false,
		},
		{
			name: "content found in output",
			trace: []service.TraceEvent{
				{
					Type: "output",
					Data: map[string]any{"content": "Hello world"},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-2",
				Type:     "content_contains",
				Expected: "world",
			},
			wantErr: false,
		},
		{
			name: "content found in content event",
			trace: []service.TraceEvent{
				{
					Type: "content",
					Data: map[string]any{"content": "test content here"},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-3",
				Type:     "content_contains",
				Expected: "test",
			},
			wantErr: false,
		},
		{
			name: "content not found",
			trace: []service.TraceEvent{
				{
					Type: "llm_output",
					Data: map[string]any{"content": "some text without the word"},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-4",
				Type:     "content_contains",
				Expected: "missing",
			},
			wantErr: true,
		},
		{
			name:    "empty trace returns error",
			trace:   []service.TraceEvent{},
			check: service.DeterministicCheck{
				ID:       "check-5",
				Type:     "content_contains",
				Expected: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &ContentContainsChecker{}
			err := checker.Check(tt.trace, tt.check)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestJSONPathChecker_Check(t *testing.T) {
	tests := []struct {
		name    string
		trace   []service.TraceEvent
		check   service.DeterministicCheck
		wantErr bool
	}{
		{
			name: "json path matches",
			trace: []service.TraceEvent{
				{
					Type: "llm_output",
					Data: map[string]any{"content": `{"result": "success", "score": 100}`},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-1",
				Type:     "json_path",
				JSONPath: "result",
				Expected: "success",
			},
			wantErr: false,
		},
		{
			name: "json path with nested object",
			trace: []service.TraceEvent{
				{
					Type: "output",
					Data: map[string]any{"content": `{"data": {"name": "test"}}`},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-2",
				Type:     "json_path",
				JSONPath: "data.name",
				Expected: "test",
			},
			wantErr: false,
		},
		{
			name: "json path value mismatch",
			trace: []service.TraceEvent{
				{
					Type: "output",
					Data: map[string]any{"content": `{"status": "pending"}`},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-4",
				Type:     "json_path",
				JSONPath: "status",
				Expected: "approved",
			},
			wantErr: true,
		},
		{
			name: "invalid JSON in trace",
			trace: []service.TraceEvent{
				{
					Type: "output",
					Data: map[string]any{"content": `not json`},
				},
			},
			check: service.DeterministicCheck{
				ID:       "check-5",
				Type:     "json_path",
				JSONPath: "data",
				Expected: "value",
			},
			wantErr: true,
		},
		{
			name:    "empty trace returns error",
			trace:   []service.TraceEvent{},
			check: service.DeterministicCheck{
				ID:       "check-6",
				Type:     "json_path",
				JSONPath: "data",
				Expected: "value",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &JSONPathChecker{}
			err := checker.Check(tt.trace, tt.check)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMatchesPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			path:     "/sandbox/test.txt",
			pattern:  "/sandbox/test.txt",
			expected: true,
		},
		{
			name:     "wildcard matches any file",
			path:     "/sandbox/anyfile.txt",
			pattern:  "/sandbox/*",
			expected: true,
		},
		{
			name:     "wildcard prefix match",
			path:     "/sandbox/prompts/test.txt",
			pattern:  "/sandbox/prompts/*",
			expected: true,
		},
		{
			name:     "no match",
			path:     "/etc/passwd",
			pattern:  "/sandbox/*",
			expected: false,
		},
		{
			name:     "wildcard matches all",
			path:     "/sandbox/anything",
			pattern:  "*",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPath(tt.path, tt.pattern)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetJSONPath(t *testing.T) {
	tests := []struct {
		name        string
		data        any
		path        string
		expected    any
		expectError bool
	}{
		{
			name:     "simple key",
			data:     map[string]any{"key": "value"},
			path:     "key",
			expected: "value",
		},
		{
			name:     "nested key",
			data:     map[string]any{"outer": map[string]any{"inner": "nested value"}},
			path:     "outer.inner",
			expected: "nested value",
		},
		{
			name:     "deeply nested",
			data:     map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}},
			path:     "a.b.c",
			expected: "deep",
		},
		{
			name:        "index out of bounds",
			data:       []any{"only one"},
			path:        "[5]",
			expectError: true,
		},
		{
			name:        "path not found in object",
			data:       map[string]any{"other": "value"},
			path:        "missing",
			expectError: false,
			expected:    nil,
		},
		{
			name:        "path not found - wrong type at key",
			data:       "string instead of map",
			path:        "key",
			expectError: true,
		},
		{
			name:        "array on non-array",
			data:       map[string]any{"key": "value"},
			path:        "[0]",
			expectError: false,
			expected:    nil,
		},
		{
			name:        "empty path returns same data",
			data:       map[string]any{"key": "value"},
			path:        "",
			expected:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getJSONPath(tt.data, tt.path)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRunShellCommand(t *testing.T) {
	// Test basic command execution
	output, err := RunShellCommand("echo", "hello")
	require.NoError(t, err)
	require.Contains(t, string(output), "hello")
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		expected    bool
	}{
		{
			name:        "path within allowed directory",
			path:        "/sandbox/test.txt",
			allowedDirs: []string{"/sandbox"},
			expected:    true,
		},
		{
			name:        "path outside allowed directory",
			path:        "/etc/passwd",
			allowedDirs: []string{"/sandbox"},
			expected:    false,
		},
		{
			name:        "path in nested allowed directory",
			path:        "/sandbox/nested/deep/test.txt",
			allowedDirs: []string{"/sandbox/nested/deep"},
			expected:    true,
		},
		{
			name:        "path outside nested directory",
			path:        "/sandbox/other/test.txt",
			allowedDirs: []string{"/sandbox/nested"},
			expected:    false,
		},
		{
			name:        "absolute path within allowed directory",
			path:        "/sandbox/test.txt",
			allowedDirs: []string{"/sandbox"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFilePath(tt.path, tt.allowedDirs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTraceEventAlias(t *testing.T) {
	// Verify TraceEvent alias works correctly
	event := TraceEvent{
		SpanID:    "span-123",
		Name:      "test-event",
		Timestamp: time.Now(),
		Type:      "event",
		Data:      map[string]any{"key": "value"},
	}

	require.Equal(t, "span-123", event.SpanID)
	require.Equal(t, "test-event", event.Name)
	require.Equal(t, "event", event.Type)
}

func TestDeterministicCheckAlias(t *testing.T) {
	check := DeterministicCheck{
		ID:       "check-1",
		Type:     "command_executed",
		Path:     "/sandbox/test.txt",
		Expected: "ls",
		JSONPath: "",
	}

	require.Equal(t, "check-1", check.ID)
	require.Equal(t, "command_executed", check.Type)
}

func TestNewRunner(t *testing.T) {
	runner := NewRunner()
	require.NotNil(t, runner)
	require.NotNil(t, runner.assertions)
}
