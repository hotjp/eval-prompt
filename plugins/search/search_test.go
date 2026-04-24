package search

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewIndexer(t *testing.T) {
	indexer := NewIndexer()
	require.NotNil(t, indexer)
}

func TestIndexer_Reconcile(t *testing.T) {
	indexer := NewIndexer()
	report, err := indexer.Reconcile(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, report.Added)
	require.Equal(t, 0, report.Updated)
	require.Equal(t, 0, report.Deleted)
	require.Nil(t, report.Errors)
}

func TestIndexer_Search(t *testing.T) {
	indexer := NewIndexer()

	// Add test assets
	assets := []service.Asset{
		{
			ID:          "asset-1",
			Name:        "Test Asset One",
			Description: "This is a test asset for unit testing",
			BizLine:     "ml",
			Tags:        []string{"test", "unit"},
			State:       "created",
		},
		{
			ID:          "asset-2",
			Name:        "Production Prompt",
			Description: "A prompt for production use",
			BizLine:     "ml",
			Tags:        []string{"prod"},
			State:       "evaluated",
		},
		{
			ID:          "asset-3",
			Name:        "Another Asset",
			Description: "Different content",
			BizLine:     "data",
			Tags:        []string{"data"},
			State:       "created",
		},
	}

	for _, asset := range assets {
		err := indexer.Save(context.Background(), asset)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		query          string
		filters        service.SearchFilters
		expectedCount  int
		expectedIDs    []string
	}{
		{
			name:          "search all with empty query",
			query:         "",
			expectedCount: 3,
		},
		{
			name:          "search by name match",
			query:         "Test",
			expectedCount: 1,
			expectedIDs:   []string{"asset-1"},
		},
		{
			name:          "search by description match",
			query:         "production",
			expectedCount: 1,
			expectedIDs:   []string{"asset-2"},
		},
		{
			name:          "search case insensitive",
			query:         "TEST",
			expectedCount: 1,
			expectedIDs:   []string{"asset-1"},
		},
		{
			name:          "search no match",
			query:         "nonexistent",
			expectedCount: 0,
		},
		{
			name:          "filter by biz line",
			query:         "",
			filters:       service.SearchFilters{BizLine: "ml"},
			expectedCount: 2,
		},
		{
			name:          "filter by state",
			query:         "",
			filters:       service.SearchFilters{State: "evaluated"},
			expectedCount: 1,
			expectedIDs:   []string{"asset-2"},
		},
		{
			name:          "filter by biz line no match",
			query:         "",
			filters:       service.SearchFilters{BizLine: "hr"},
			expectedCount: 0,
		},
		{
			name:          "filter by tags",
			query:         "",
			filters:       service.SearchFilters{Tags: []string{"test"}},
			expectedCount: 1,
			expectedIDs:   []string{"asset-1"},
		},
		{
			name:          "filter by multiple tags - first tag matches",
			query:         "",
			filters:       service.SearchFilters{Tags: []string{"test", "data"}},
			expectedCount: 2,
		},
		{
			name:          "filter by single tag that exists",
			query:         "",
			filters:       service.SearchFilters{Tags: []string{"data"}},
			expectedCount: 1,
			expectedIDs:   []string{"asset-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := indexer.Search(context.Background(), tt.query, tt.filters)
			require.NoError(t, err)
			require.Len(t, results, tt.expectedCount)

			if tt.expectedIDs != nil {
				ids := make([]string, len(results))
				for i, r := range results {
					ids[i] = r.ID
				}
				require.ElementsMatch(t, tt.expectedIDs, ids)
			}
		})
	}
}

func TestIndexer_Search_EmptyIndex(t *testing.T) {
	indexer := NewIndexer()
	results, err := indexer.Search(context.Background(), "test", service.SearchFilters{})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestIndexer_GetByID(t *testing.T) {
	indexer := NewIndexer()

	asset := service.Asset{
		ID:          "asset-1",
		Name:        "Test Asset",
		Description: "A test asset",
		BizLine:     "ml",
		Tags:        []string{"test"},
		State:       "created",
	}

	err := indexer.Save(context.Background(), asset)
	require.NoError(t, err)

	tests := []struct {
		name      string
		id        string
		wantErr   bool
		wantName  string
	}{
		{
			name:     "existing asset",
			id:       "asset-1",
			wantErr:  false,
			wantName: "Test Asset",
		},
		{
			name:    "non-existent asset",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail, err := indexer.GetByID(context.Background(), tt.id)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantName, detail.Name)
			}
		})
	}
}

func TestIndexer_Save(t *testing.T) {
	indexer := NewIndexer()

	asset := service.Asset{
		ID:          "asset-1",
		Name:        "Test Asset",
		Description: "Description",
		BizLine:     "ml",
		Tags:        []string{"test"},
		State:       "created",
	}

	err := indexer.Save(context.Background(), asset)
	require.NoError(t, err)

	// Verify it can be retrieved
	detail, err := indexer.GetByID(context.Background(), "asset-1")
	require.NoError(t, err)
	require.Equal(t, "Test Asset", detail.Name)
	require.Equal(t, "Description", detail.Description)
	require.Equal(t, "ml", detail.BizLine)
}

func TestIndexer_Save_UpdatesExisting(t *testing.T) {
	indexer := NewIndexer()

	asset1 := service.Asset{
		ID:          "asset-1",
		Name:        "Original Name",
		Description: "Original Description",
		BizLine:     "ml",
		Tags:        []string{"test"},
		State:       "created",
	}

	err := indexer.Save(context.Background(), asset1)
	require.NoError(t, err)

	asset2 := service.Asset{
		ID:          "asset-1",
		Name:        "Updated Name",
		Description: "Updated Description",
		BizLine:     "ml",
		Tags:        []string{"test", "updated"},
		State:       "evaluated",
	}

	err = indexer.Save(context.Background(), asset2)
	require.NoError(t, err)

	detail, err := indexer.GetByID(context.Background(), "asset-1")
	require.NoError(t, err)
	require.Equal(t, "Updated Name", detail.Name)
	require.Equal(t, "evaluated", detail.State)
}

func TestIndexer_Delete(t *testing.T) {
	indexer := NewIndexer()

	asset := service.Asset{
		ID:          "asset-1",
		Name:        "Test Asset",
		Description: "A test asset",
		BizLine:     "ml",
		Tags:        []string{"test"},
		State:       "created",
	}

	err := indexer.Save(context.Background(), asset)
	require.NoError(t, err)

	// Verify it exists
	_, err = indexer.GetByID(context.Background(), "asset-1")
	require.NoError(t, err)

	// Delete it
	err = indexer.Delete(context.Background(), "asset-1")
	require.NoError(t, err)

	// Verify it's gone
	_, err = indexer.GetByID(context.Background(), "asset-1")
	require.Error(t, err)
}

func TestIndexer_Delete_NonExistent(t *testing.T) {
	indexer := NewIndexer()
	err := indexer.Delete(context.Background(), "non-existent")
	require.NoError(t, err) // Delete should not error for non-existent
}

func TestIndexer_ImplementsInterface(t *testing.T) {
	indexer := NewIndexer()
	var _ service.AssetIndexer = indexer
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "simple match",
			s:        "Hello World",
			substr:   "World",
			expected: true,
		},
		{
			name:     "case insensitive match",
			s:        "Hello World",
			substr:   "world",
			expected: true,
		},
		{
			name:     "no match",
			s:        "Hello World",
			substr:   "foo",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "Hello World",
			substr:   "",
			expected: true,
		},
		{
			name:     "substring longer than string",
			s:        "Hi",
			substr:   "Hello",
			expected: false,
		},
		{
			name:     "exact match",
			s:        "Hello",
			substr:   "Hello",
			expected: true,
		},
		{
			name:     "match at start",
			s:        "Hello World",
			substr:   "Hello",
			expected: true,
		},
		{
			name:     "match at end",
			s:        "Hello World",
			substr:   "World",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.s, tt.substr)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestEqualIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "equal strings",
			a:        "Hello",
			b:        "Hello",
			expected: true,
		},
		{
			name:     "case insensitive equal",
			a:        "Hello",
			b:        "hello",
			expected: true,
		},
		{
			name:     "different strings",
			a:        "Hello",
			b:        "World",
			expected: false,
		},
		{
			name:     "different lengths",
			a:        "Hello",
			b:        "Hello!",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalIgnoreCase(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}
