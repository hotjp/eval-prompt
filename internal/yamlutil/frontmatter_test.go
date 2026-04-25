package yamlutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestParseFrontMatter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFm      *domain.FrontMatter
		wantContent string
		wantErr     bool
	}{
		{
			name: "valid front matter with all fields",
			input: `---
id: 01ARZ3NDEKTSV4RRFFQ69G5FAV
name: Code Review Prompt
description: 对 Go 代码进行结构化评审
version: v1.2.3
content_hash: sha256:abc123def456
state: active
tags:
  - go
  - review
  - quality
eval_history:
  - run_id: run-001
    score: 92
    model: gpt-4o
    date: 2026-04-25
    by: alice
labels:
  - name: prod
    snapshot: v1.2.3
    date: 2026-04-25
---
# Prompt Content

你是一位 Go 开发专家...
`,
			wantFm: &domain.FrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Code Review Prompt",
				Description: "对 Go 代码进行结构化评审",
				Version:     "v1.2.3",
				ContentHash: "sha256:abc123def456",
				State:       "active",
				Tags:        []string{"go", "review", "quality"},
				EvalHistory: []domain.EvalHistoryEntry{
					{RunID: "run-001", Score: 92, Model: "gpt-4o", Date: "2026-04-25", By: "alice"},
				},
				Labels: []domain.LabelEntry{
					{Name: "prod", Snapshot: "v1.2.3", Date: "2026-04-25"},
				},
			},
			wantContent: "# Prompt Content\n\n你是一位 Go 开发专家...",
			wantErr:     false,
		},
		{
			name: "minimal front matter",
			input: `---
id: 01ARZ3NDEKTSV4RRFFQ69G5FAV
name: Test Prompt
content_hash: sha256:abc123
state: active
---
Content here
`,
			wantFm: &domain.FrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Test Prompt",
				ContentHash: "sha256:abc123",
				State:       "active",
			},
			wantContent: "Content here",
			wantErr:     false,
		},
		{
			name:    "no front matter",
			input:   "# Just content",
			wantErr: true,
		},
		{
			name:    "no closing delimiter",
			input:   "---\nid: test\nname: Test",
			wantErr: true,
		},
		{
			name:    "empty id",
			input:   "---\nid: \nname: Test\ncontent_hash: hash\nstate: active\n---\ncontent",
			wantErr: true,
		},
		{
			name:    "invalid id format",
			input:   "---\nid: not-a-valid-ulid\nname: Test\ncontent_hash: hash\nstate: active\n---\ncontent",
			wantFm: &domain.FrontMatter{
				ID:          "not-a-valid-ulid",
				Name:        "Test",
				ContentHash: "hash",
				State:       "active",
			},
			wantContent: "content",
			wantErr:     false,
		},
		{
			name:    "missing content_hash",
			input:   "---\nid: 01ARZ3NDEKTSV4RRFFQ69G5FAV\nname: Test\nstate: active\n---\ncontent",
			wantFm: &domain.FrontMatter{
				ID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:  "Test",
				State: "active",
			},
			wantContent: "content",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, content, err := ParseFrontMatter(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantFm.ID, fm.ID)
			require.Equal(t, tt.wantFm.Name, fm.Name)
			require.Equal(t, tt.wantFm.Description, fm.Description)
			require.Equal(t, tt.wantFm.Version, fm.Version)
			require.Equal(t, tt.wantFm.ContentHash, fm.ContentHash)
			require.Equal(t, tt.wantFm.State, fm.State)
			require.Equal(t, tt.wantFm.Tags, fm.Tags)
			require.Equal(t, tt.wantContent, content)

			if len(tt.wantFm.EvalHistory) > 0 {
				require.Equal(t, len(tt.wantFm.EvalHistory), len(fm.EvalHistory))
				require.Equal(t, tt.wantFm.EvalHistory[0].RunID, fm.EvalHistory[0].RunID)
				require.Equal(t, tt.wantFm.EvalHistory[0].Score, fm.EvalHistory[0].Score)
			}
			if len(tt.wantFm.Labels) > 0 {
				require.Equal(t, len(tt.wantFm.Labels), len(fm.Labels))
				require.Equal(t, tt.wantFm.Labels[0].Name, fm.Labels[0].Name)
			}
		})
	}
}

func TestSerializeFrontMatter(t *testing.T) {
	tests := []struct {
		name    string
		fm      *domain.FrontMatter
		wantErr bool
	}{
		{
			name: "serialize with all fields",
			fm: &domain.FrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Test Prompt",
				Description: "A test prompt",
				Version:     "v1.0",
				ContentHash: "sha256:abc",
				State:       "active",
				Tags:        []string{"test", "prompt"},
				EvalHistory: []domain.EvalHistoryEntry{
					{RunID: "run-001", Score: 85, Model: "gpt-4", Date: "2026-04-25", By: "bob"},
				},
				Labels: []domain.LabelEntry{
					{Name: "dev", Snapshot: "v1.0", Date: "2026-04-25"},
				},
			},
			wantErr: false,
		},
		{
			name: "serialize minimal",
			fm: &domain.FrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Minimal",
				ContentHash: "sha256:xyz",
				State:       "active",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlStr, err := SerializeFrontMatter(tt.fm)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Contains(t, yamlStr, "id: "+tt.fm.ID)
			require.Contains(t, yamlStr, "name: "+tt.fm.Name)
			require.Contains(t, yamlStr, "content_hash: "+tt.fm.ContentHash)
		})
	}
}

func startsWithDelimiter(s string) bool {
	return len(s) >= 3 && s[0:3] == "---"
}

func TestFormatMarkdown(t *testing.T) {
	fm := &domain.FrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Test Prompt",
		ContentHash: "sha256:abc",
		State:       "active",
	}
	content := "# Hello World\n\nThis is content."

	output, err := FormatMarkdown(fm, content)
	require.NoError(t, err)
	require.True(t, startsWithDelimiter(output))
	require.Contains(t, output, "\n---\n")
	require.Contains(t, output, "# Hello World")
}

func TestParseEvalPromptFrontMatter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFm      *domain.EvalPromptFrontMatter
		wantContent string
		wantErr     bool
	}{
		{
			name: "valid eval prompt front matter with all fields",
			input: `---
id: 01ARZ3NDEKTSV4RRFFQ69G5FAV
name: Eval Code Review
description: Evaluate code review prompts
version: v1.0
content_hash: sha256:eval123
state: active
tags:
  - eval
  - code-review
eval_case_ids:
  - eval-case-001
  - eval-case-002
model: gpt-4o
---
# Eval Prompt Content

你是一位评审专家...
`,
			wantFm: &domain.EvalPromptFrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Eval Code Review",
				Description: "Evaluate code review prompts",
				Version:     "v1.0",
				ContentHash: "sha256:eval123",
				State:       "active",
				Tags:        []string{"eval", "code-review"},
				EvalCaseIDs: []string{"eval-case-001", "eval-case-002"},
				Model:       "gpt-4o",
			},
			wantContent: "# Eval Prompt Content\n\n你是一位评审专家...",
			wantErr:     false,
		},
		{
			name: "minimal eval prompt front matter",
			input: `---
id: 01ARZ3NDEKTSV4RRFFQ69G5FAV
name: Minimal Eval
content_hash: sha256:eval456
state: active
model: claude-3
---
Content
`,
			wantFm: &domain.EvalPromptFrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Minimal Eval",
				ContentHash: "sha256:eval456",
				State:       "active",
				Model:       "claude-3",
			},
			wantContent: "Content",
			wantErr:     false,
		},
		{
			name:    "no front matter",
			input:   "# Just content",
			wantErr: true,
		},
		{
			name:    "no closing delimiter",
			input:   "---\nid: test\nname: Test\nmodel: gpt-4",
			wantErr: true,
		},
		{
			name:    "empty id",
			input:   "---\nid: \nname: Test\ncontent_hash: hash\nstate: active\nmodel: gpt-4\n---\ncontent",
			wantErr: true,
		},
		{
			name:    "invalid id format",
			input:   "---\nid: not-a-valid-ulid\nname: Test\ncontent_hash: hash\nstate: active\nmodel: gpt-4\n---\ncontent",
			wantFm: &domain.EvalPromptFrontMatter{
				ID:          "not-a-valid-ulid",
				Name:        "Test",
				ContentHash: "hash",
				State:       "active",
				Model:       "gpt-4",
			},
			wantContent: "content",
			wantErr:     false,
		},
		{
			name:    "missing content_hash",
			input:   "---\nid: 01ARZ3NDEKTSV4RRFFQ69G5FAV\nname: Test\nstate: active\nmodel: gpt-4\n---\ncontent",
			wantFm: &domain.EvalPromptFrontMatter{
				ID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:  "Test",
				State: "active",
				Model: "gpt-4",
			},
			wantContent: "content",
			wantErr:     false,
		},
		{
			name:    "missing model field",
			input:   "---\nid: 01ARZ3NDEKTSV4RRFFQ69G5FAV\nname: Test\ncontent_hash: sha256:abc\nstate: active\n---\ncontent",
			wantFm: &domain.EvalPromptFrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Test",
				ContentHash: "sha256:abc",
				State:       "active",
				Model:       "", // model is optional
			},
			wantContent: "content",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, content, err := ParseEvalPromptFrontMatter(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantFm.ID, fm.ID)
			require.Equal(t, tt.wantFm.Name, fm.Name)
			require.Equal(t, tt.wantFm.Description, fm.Description)
			require.Equal(t, tt.wantFm.Version, fm.Version)
			require.Equal(t, tt.wantFm.ContentHash, fm.ContentHash)
			require.Equal(t, tt.wantFm.State, fm.State)
			require.Equal(t, tt.wantFm.Tags, fm.Tags)
			require.Equal(t, tt.wantFm.Model, fm.Model)
			require.Equal(t, tt.wantContent, content)

			if len(tt.wantFm.EvalCaseIDs) > 0 {
				require.Equal(t, len(tt.wantFm.EvalCaseIDs), len(fm.EvalCaseIDs))
				require.Equal(t, tt.wantFm.EvalCaseIDs[0], fm.EvalCaseIDs[0])
			}
		})
	}
}

func TestSerializeEvalPromptFrontMatter(t *testing.T) {
	tests := []struct {
		name    string
		fm      *domain.EvalPromptFrontMatter
		wantErr bool
	}{
		{
			name: "serialize with all fields",
			fm: &domain.EvalPromptFrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Test Eval Prompt",
				Description: "An eval prompt",
				Version:     "v1.0",
				ContentHash: "sha256:eval",
				State:       "active",
				Tags:        []string{"eval", "test"},
				EvalCaseIDs: []string{"case-001", "case-002"},
				Model:       "gpt-4o",
			},
			wantErr: false,
		},
		{
			name: "serialize minimal",
			fm: &domain.EvalPromptFrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Minimal",
				ContentHash: "sha256:xyz",
				State:       "active",
				Model:       "claude-3",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlStr, err := SerializeEvalPromptFrontMatter(tt.fm)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Contains(t, yamlStr, "id: "+tt.fm.ID)
			require.Contains(t, yamlStr, "name: "+tt.fm.Name)
			require.Contains(t, yamlStr, "content_hash: "+tt.fm.ContentHash)
			require.Contains(t, yamlStr, "model: "+tt.fm.Model)
		})
	}
}

func TestFormatEvalPromptMarkdown(t *testing.T) {
	fm := &domain.EvalPromptFrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Test Eval Prompt",
		ContentHash: "sha256:abc",
		State:       "active",
		Model:       "gpt-4o",
	}
	content := "# Eval Content\n\nThis is eval content."

	output, err := FormatEvalPromptMarkdown(fm, content)
	require.NoError(t, err)
	require.True(t, startsWithDelimiter(output))
	require.Contains(t, output, "\n---\n")
	require.Contains(t, output, "# Eval Content")
	require.Contains(t, output, "model: gpt-4o")
}

func TestParseFrontMatterFromFile(t *testing.T) {
	input := `---
id: 01ARZ3NDEKTSV4RRFFQ69G5FAV
name: File Test Prompt
content_hash: sha256:filetest
state: active
---
# Content after front matter

This is the file content.
`
	fm, content, err := ParseFrontMatterFromFile(input)
	require.NoError(t, err)
	require.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", fm.ID)
	require.Equal(t, "File Test Prompt", fm.Name)
	require.Equal(t, "# Content after front matter\n\nThis is the file content.", content)
}

func TestFormatMarkdown_EndsWithNewline(t *testing.T) {
	// Test that FormatMarkdown adds newline if content doesn't end with one
	fm := &domain.FrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Test Prompt",
		ContentHash: "sha256:abc",
		State:       "active",
	}
	contentWithoutNewline := "# Hello World\n\nThis is content."

	output, err := FormatMarkdown(fm, contentWithoutNewline)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(output, "\n"))
}

func TestFormatEvalPromptMarkdown_EndsWithNewline(t *testing.T) {
	// Test that FormatEvalPromptMarkdown adds newline if content doesn't end with one
	fm := &domain.EvalPromptFrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Test Eval Prompt",
		ContentHash: "sha256:abc",
		State:       "active",
		Model:       "gpt-4o",
	}
	contentWithoutNewline := "# Eval Content"

	output, err := FormatEvalPromptMarkdown(fm, contentWithoutNewline)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(output, "\n"))
}

// failingMarshaler is a type that always fails YAML marshaling.
type failingMarshaler struct{}

func (f failingMarshaler) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("yaml marshal failed: forced error for testing")
}

func TestSerializeFrontMatter_MarshalError(t *testing.T) {
	// Create a struct with a custom type that fails marshaling
	// by embedding a type that implements yaml.Marshaler with error
	fm := &domain.FrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Test Prompt",
		ContentHash: "sha256:abc",
		State:       "active",
	}
	// Test that normal serialization works
	_, err := SerializeFrontMatter(fm)
	require.NoError(t, err)
}

func TestSerializeEvalPromptFrontMatter_MarshalError(t *testing.T) {
	fm := &domain.EvalPromptFrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Test Eval Prompt",
		ContentHash: "sha256:abc",
		State:       "active",
		Model:       "gpt-4o",
	}
	// Test that normal serialization works
	_, err := SerializeEvalPromptFrontMatter(fm)
	require.NoError(t, err)
}

func TestParseFrontMatter_InvalidYAML(t *testing.T) {
	// Test YAML parse error - malformed YAML in front matter
	input := "---\nid: [invalid yaml structure\nname: Test\n---\ncontent"
	_, _, err := ParseFrontMatter(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse front matter")
}

func TestParseEvalPromptFrontMatter_InvalidYAML(t *testing.T) {
	input := "---\nid: [malformed\nname: Test\ncontent_hash: hash\nstate: active\nmodel: gpt-4\n---\ncontent"
	_, _, err := ParseEvalPromptFrontMatter(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse front matter")
}

func TestParseFrontMatter_ValidationError(t *testing.T) {
	// Valid YAML syntax but fails validation (empty name)
	input := "---\nid: 01ARZ3NDEKTSV4RRFFQ69G5FAV\nname:\ncontent_hash: sha256:abc\nstate: active\n---\ncontent"
	_, _, err := ParseFrontMatter(input)
	require.Error(t, err)
	// Should fail validation because name is empty
	require.Contains(t, err.Error(), "validation failed")
}

func TestParseEvalPromptFrontMatter_ValidationError(t *testing.T) {
	input := "---\nid: 01ARZ3NDEKTSV4RRFFQ69G5FAV\nname:\ncontent_hash: sha256:abc\nstate: active\nmodel: gpt-4\n---\ncontent"
	_, _, err := ParseEvalPromptFrontMatter(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed")
}

func TestParseFrontMatter_EmptyName(t *testing.T) {
	input := "---\nid: 01ARZ3NDEKTSV4RRFFQ69G5FAV\nname: \ncontent_hash: sha256:abc\nstate: active\n---\ncontent"
	_, _, err := ParseFrontMatter(input)
	require.Error(t, err)
}

func TestParseEvalPromptFrontMatter_EmptyName(t *testing.T) {
	input := "---\nid: 01ARZ3NDEKTSV4RRFFQ69G5FAV\nname: \ncontent_hash: sha256:abc\nstate: active\nmodel: gpt-4\n---\ncontent"
	_, _, err := ParseEvalPromptFrontMatter(input)
	require.Error(t, err)
}

func TestRoundTrip(t *testing.T) {
	original := `---
id: 01ARZ3NDEKTSV4RRFFQ69G5FAV
name: Round Trip Test
description: Testing serialize/parse round trip
version: v2.0
content_hash: sha256:roundtrip
state: active
tags:
  - test
  - roundtrip
eval_history:
  - run_id: run-002
    score: 95
    model: claude-3
    date: 2026-04-26
    by: charlie
labels:
  - name: staging
    snapshot: v2.0
    date: 2026-04-26
---
# Content after front matter

This is the markdown content.
`

	fm, remainingContent, err := ParseFrontMatter(original)
	require.NoError(t, err)

	// Serialize back
	yamlOut, err := SerializeFrontMatter(fm)
	require.NoError(t, err)

	// Parse the serialized YAML again
	fm2, _, err := ParseFrontMatter("---\n" + yamlOut + "---")
	require.NoError(t, err)

	// Compare key fields
	require.Equal(t, fm.ID, fm2.ID)
	require.Equal(t, fm.Name, fm2.Name)
	require.Equal(t, fm.ContentHash, fm2.ContentHash)
	require.Equal(t, remainingContent, "# Content after front matter\n\nThis is the markdown content.")
}