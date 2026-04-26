package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/eval-prompt/internal/domain"
)

// Deprecated: Execution is stored in .evals/executions/*.json, not in database.
type ExecutionFileStore struct {
	baseDir string
	mu      sync.RWMutex
}

func NewExecutionFileStore(baseDir string) *ExecutionFileStore {
	return &ExecutionFileStore{baseDir: baseDir}
}

func (s *ExecutionFileStore) filePath(id string) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s.json", id))
}

func (s *ExecutionFileStore) Save(ctx context.Context, exec *domain.EvalExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(exec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath(exec.ID), data, 0644)
}

func (s *ExecutionFileStore) Get(ctx context.Context, id string) (*domain.EvalExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath(id))
	if err != nil {
		return nil, err
	}
	var exec domain.EvalExecution
	if err := json.Unmarshal(data, &exec); err != nil {
		return nil, err
	}
	return &exec, nil
}

func (s *ExecutionFileStore) ListByAsset(ctx context.Context, assetID string) ([]*domain.EvalExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var result []*domain.EvalExecution
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
		if err != nil {
			continue
		}
		var exec domain.EvalExecution
		if err := json.Unmarshal(data, &exec); err != nil {
			continue
		}
		if exec.AssetID == assetID {
			result = append(result, &exec)
		}
	}
	return result, nil
}

func (s *ExecutionFileStore) List(ctx context.Context, offset, limit int) ([]*domain.EvalExecution, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, 0, err
	}

	var all []*domain.EvalExecution
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
		if err != nil {
			continue
		}
		var exec domain.EvalExecution
		if err := json.Unmarshal(data, &exec); err != nil {
			continue
		}
		all = append(all, &exec)
	}

	total := len(all)

	if offset >= len(all) {
		return []*domain.EvalExecution{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (s *ExecutionFileStore) UpdateStatus(ctx context.Context, id string, status domain.ExecutionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, err := s.getWithoutLock(id)
	if err != nil {
		return err
	}
	exec.Status = status
	data, err := json.MarshalIndent(exec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath(id), data, 0644)
}

func (s *ExecutionFileStore) UpdateProgress(ctx context.Context, id string, completed, failed, cancelled int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, err := s.getWithoutLock(id)
	if err != nil {
		return err
	}
	exec.CompletedRuns = completed
	exec.FailedRuns = failed
	exec.CancelledRuns = cancelled
	data, err := json.MarshalIndent(exec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath(id), data, 0644)
}

func (s *ExecutionFileStore) Archive(ctx context.Context, id string) error {
	return nil
}

func (s *ExecutionFileStore) IsArchived(ctx context.Context, id string) bool {
	return false
}

func (s *ExecutionFileStore) getWithoutLock(id string) (*domain.EvalExecution, error) {
	data, err := os.ReadFile(s.filePath(id))
	if err != nil {
		return nil, err
	}
	var exec domain.EvalExecution
	if err := json.Unmarshal(data, &exec); err != nil {
		return nil, err
	}
	return &exec, nil
}
