package service

import (
	"context"
	"errors"
	"testing"
)

// mockSyncIndexer is a local mock implementation of AssetIndexer for sync service testing.
type mockSyncIndexer struct {
	SearchFunc    func(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error)
	GetByIDFunc   func(ctx context.Context, id string) (*AssetDetail, error)
	SaveFunc      func(ctx context.Context, asset Asset) error
	DeleteFunc    func(ctx context.Context, id string) error
	ReconcileFunc func(ctx context.Context) (ReconcileReport, error)
}

func (m *mockSyncIndexer) Search(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, filters)
	}
	return nil, nil
}

func (m *mockSyncIndexer) GetByID(ctx context.Context, id string) (*AssetDetail, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockSyncIndexer) Save(ctx context.Context, asset Asset) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, asset)
	}
	return nil
}

func (m *mockSyncIndexer) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockSyncIndexer) Reconcile(ctx context.Context) (ReconcileReport, error) {
	if m.ReconcileFunc != nil {
		return m.ReconcileFunc(ctx)
	}
	return ReconcileReport{}, nil
}

func TestSyncService_Reconcile(t *testing.T) {
	mockIndexer := &mockSyncIndexer{
		ReconcileFunc: func(ctx context.Context) (ReconcileReport, error) {
			return ReconcileReport{
				Added:   2,
				Updated: 1,
				Deleted: 0,
				Errors:  nil,
			}, nil
		},
	}

	svc := NewSyncService(mockIndexer, nil)
	ctx := context.Background()

	report, err := svc.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Added != 2 {
		t.Errorf("expected Added=2, got %d", report.Added)
	}
	if report.Updated != 1 {
		t.Errorf("expected Updated=1, got %d", report.Updated)
	}
}

func TestSyncService_Reconcile_NilIndexer(t *testing.T) {
	svc := NewSyncService(nil, nil)
	ctx := context.Background()

	_, err := svc.Reconcile(ctx)
	if err == nil {
		t.Error("expected error for nil indexer")
	}
}

func TestSyncService_RebuildIndex(t *testing.T) {
	callCount := 0
	mockIndexer := &mockSyncIndexer{
		ReconcileFunc: func(ctx context.Context) (ReconcileReport, error) {
			callCount++
			return ReconcileReport{
				Added:   1,
				Updated: 0,
				Deleted: 0,
				Errors:  nil,
			}, nil
		},
	}

	svc := NewSyncService(mockIndexer, nil)
	ctx := context.Background()

	err := svc.RebuildIndex(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 reconcile call, got %d", callCount)
	}
}

func TestSyncService_RebuildIndex_NilIndexer(t *testing.T) {
	svc := NewSyncService(nil, nil)
	ctx := context.Background()

	err := svc.RebuildIndex(ctx)
	if err == nil {
		t.Error("expected error for nil indexer")
	}
}

func TestSyncService_RebuildIndex_ReconcileErrors(t *testing.T) {
	mockIndexer := &mockSyncIndexer{
		ReconcileFunc: func(ctx context.Context) (ReconcileReport, error) {
			return ReconcileReport{
				Added:   0,
				Updated: 0,
				Deleted: 0,
				Errors:  []string{"error 1", "error 2"},
			}, nil
		},
	}

	svc := NewSyncService(mockIndexer, nil)
	ctx := context.Background()

	err := svc.RebuildIndex(ctx)
	if err == nil {
		t.Error("expected error when reconcile has errors")
	}
}

func TestSyncService_Export_JSON(t *testing.T) {
	mockIndexer := &mockSyncIndexer{
		SearchFunc: func(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error) {
			return []AssetSummary{
				{
					ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
					Name:        "Test Asset",
					Description: "Test description",
					BizLine:     "test",
					Tags:        []string{"test"},
					State:       "created",
				},
			}, nil
		},
	}

	svc := NewSyncService(mockIndexer, nil)
	ctx := context.Background()

	data, err := svc.Export(ctx, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestSyncService_Export_YAML(t *testing.T) {
	mockIndexer := &mockSyncIndexer{
		SearchFunc: func(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error) {
			return []AssetSummary{
				{
					ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
					Name:        "Test Asset",
					Description: "Test description",
					BizLine:     "test",
					Tags:        []string{"test"},
					State:       "created",
				},
			}, nil
		},
	}

	svc := NewSyncService(mockIndexer, nil)
	ctx := context.Background()

	data, err := svc.Export(ctx, "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestSyncService_Export_UnsupportedFormat(t *testing.T) {
	svc := NewSyncService(nil, nil)
	ctx := context.Background()

	_, err := svc.Export(ctx, "xml")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestSyncService_Export_NilIndexer(t *testing.T) {
	svc := NewSyncService(nil, nil)
	ctx := context.Background()

	_, err := svc.Export(ctx, "json")
	if err == nil {
		t.Error("expected error for nil indexer")
	}
}

func TestSyncService_Export_IndexerError(t *testing.T) {
	mockIndexer := &mockSyncIndexer{
		SearchFunc: func(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error) {
			return nil, errors.New("search failed")
		},
	}

	svc := NewSyncService(mockIndexer, nil)
	ctx := context.Background()

	_, err := svc.Export(ctx, "json")
	if err == nil {
		t.Error("expected error when indexer search fails")
	}
}
