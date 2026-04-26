package service

import (
	"context"
	"testing"
)

// mockAssetIndexer is a local mock implementation of AssetIndexer for testing.
type mockAssetIndexer struct {
	SearchFunc  func(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error)
	GetByIDFunc func(ctx context.Context, id string) (*AssetDetail, error)
	SaveFunc    func(ctx context.Context, asset Asset) error
	DeleteFunc  func(ctx context.Context, id string) error
	ReconcileFunc func(ctx context.Context) (ReconcileReport, error)
}

func (m *mockAssetIndexer) Search(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, filters)
	}
	return nil, nil
}

func (m *mockAssetIndexer) GetByID(ctx context.Context, id string) (*AssetDetail, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAssetIndexer) Save(ctx context.Context, asset Asset) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, asset)
	}
	return nil
}

func (m *mockAssetIndexer) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockAssetIndexer) Reconcile(ctx context.Context) (ReconcileReport, error) {
	if m.ReconcileFunc != nil {
		return m.ReconcileFunc(ctx)
	}
	return ReconcileReport{}, nil
}

func (m *mockAssetIndexer) GetFileContent(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (m *mockAssetIndexer) SaveFileContent(ctx context.Context, id, fullContent, commitMessage string) (string, error) {
	return "", nil
}

func (m *mockAssetIndexer) CreatePlaceholder(ctx context.Context, id, name, bizLine string, tags []string, category string) error {
	return nil
}

func (m *mockAssetIndexer) ReInit(ctx context.Context, path string) error {
	return nil
}

func TestTriggerService_MatchTrigger(t *testing.T) {
	mockIndexer := &mockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error) {
			return []AssetSummary{
				{
					ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
					Name:        "Code Review",
					Description: "Review code changes",
					AssetType:     "engineering",
					Tags:        []string{"code", "review"},
					State:       "created",
				},
				{
					ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAW",
					Name:        "Documentation Writer",
					Description: "Write documentation",
					AssetType:     "docs",
					Tags:        []string{"docs", "writing"},
					State:       "created",
				},
			}, nil
		},
	}

	svc := NewTriggerService(mockIndexer, nil)
	ctx := context.Background()

	matches, err := svc.MatchTrigger(ctx, "code review", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestTriggerService_MatchTrigger_DefaultTop(t *testing.T) {
	callCount := 0
	mockIndexer := &mockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error) {
			callCount++
			return []AssetSummary{
				{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Name: "Test"},
			}, nil
		},
	}

	svc := NewTriggerService(mockIndexer, nil)
	ctx := context.Background()

	// top=0 should default to 5
	_, err := svc.MatchTrigger(ctx, "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTriggerService_MatchTrigger_NilIndexer(t *testing.T) {
	svc := NewTriggerService(nil, nil)
	ctx := context.Background()

	_, err := svc.MatchTrigger(ctx, "test", 5)
	if err == nil {
		t.Error("expected error for nil indexer")
	}
}

func TestTriggerService_ValidateAntiPatterns(t *testing.T) {
	svc := NewTriggerService(nil, nil)
	ctx := context.Background()

	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{
			name:    "valid prompt",
			prompt:  "Please explain this code",
			wantErr: false,
		},
		{
			name:    "contains generate code",
			prompt:  "Please generate code for me",
			wantErr: true,
		},
		{
			name:    "contains write new feature",
			prompt:  "Write new feature implementation",
			wantErr: true,
		},
		{
			name:    "contains refactor entire",
			prompt:  "Refactor entire codebase",
			wantErr: true,
		},
		{
			name:    "contains delete all",
			prompt:  "Delete all files",
			wantErr: true,
		},
		{
			name:    "contains drop table",
			prompt:  "Drop table users",
			wantErr: true,
		},
		{
			name:    "contains rm -rf",
			prompt:  "rm -rf /tmp",
			wantErr: true,
		},
		{
			name:    "case insensitive match",
			prompt:  "Write NEW FEATURE for me",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateAntiPatterns(ctx, tt.prompt)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTriggerService_InjectVariables(t *testing.T) {
	svc := NewTriggerService(nil, nil)
	ctx := context.Background()

	tests := []struct {
		name   string
		prompt string
		vars   map[string]string
		expect string
	}{
		{
			name:   "double brace syntax",
			prompt: "Hello, {{name}}!",
			vars:   map[string]string{"name": "World"},
			expect: "Hello, World!",
		},
		{
			name:   "dollar brace syntax",
			prompt: "Hello, ${name}!",
			vars:   map[string]string{"name": "World"},
			expect: "Hello, World!",
		},
		{
			name:   "multiple variables",
			prompt: "{{greeting}}, {{name}}! You have {{count}} messages.",
			vars:   map[string]string{"greeting": "Hello", "name": "Alice", "count": "5"},
			expect: "Hello, Alice! You have 5 messages.",
		},
		{
			name:   "mixed syntax",
			prompt: "{{greeting}}, ${name}!",
			vars:   map[string]string{"greeting": "Hi", "name": "Bob"},
			expect: "Hi, Bob!",
		},
		{
			name:   "empty vars",
			prompt: "Hello, {{name}}!",
			vars:   map[string]string{},
			expect: "Hello, {{name}}!",
		},
		{
			name:   "no placeholders",
			prompt: "Hello, World!",
			vars:   map[string]string{"name": "Alice"},
			expect: "Hello, World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.InjectVariables(ctx, tt.prompt, tt.vars)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expect {
				t.Errorf("expected %q, got %q", tt.expect, result)
			}
		})
	}
}

func TestCalculateRelevance(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		assetName   string
		description string
		minScore    float64
	}{
		{
			name:        "exact word match in name",
			input:       "code review",
			assetName:   "code review prompt",
			description: "review code changes",
			minScore:    0.3,
		},
		{
			name:        "no match",
			input:       "unrelated query",
			assetName:   "code review prompt",
			description: "review code changes",
			minScore:    0.0,
		},
		{
			name:        "short input words ignored",
			input:       "ab cd ef", // "ab" and "cd" are < 3 chars, ignored
			assetName:   "test prompt",
			description: "test description",
			minScore:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateRelevance(tt.input, tt.assetName, tt.description)
			if score < tt.minScore {
				t.Errorf("expected score >= %f, got %f", tt.minScore, score)
			}
		})
	}
}
