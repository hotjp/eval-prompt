package service

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/eval-prompt/internal/pathutil"
)

// LLMCall represents a single LLM call record stored in JSON Lines format.
type LLMCall struct {
	RunID           string    `json:"run_id"`
	ExecutionID     string    `json:"execution_id"`
	AssetID         string    `json:"asset_id"`
	SnapshotID      string    `json:"snapshot_id"`
	CaseID          string    `json:"case_id"`
	RunNumber       int       `json:"run_number"`
	Status          string    `json:"status"` // completed/failed/cancelled
	Model           string    `json:"model"`
	Temperature     float64   `json:"temperature"`
	TokensIn        int       `json:"tokens_in"`
	TokensOut       int       `json:"tokens_out"`
	LatencyMs       int64     `json:"latency_ms"`
	ResponseContent string    `json:"response_content"`
	Error           string    `json:"error"`
	Timestamp       time.Time `json:"timestamp"`
}

// LLMCallFileStore handles persistent storage of LLM calls in JSON Lines format.
// File path: .evals/calls/{execution_id}/calls.jsonl
type LLMCallFileStore struct {
	baseDir string
	mu      sync.Mutex // protects concurrent writes to the same execution
}

func NewLLMCallFileStore(baseDir string) *LLMCallFileStore {
	return &LLMCallFileStore{baseDir: baseDir}
}

func (s *LLMCallFileStore) filePath(executionID string) (string, error) {
	if err := pathutil.ValidateID(executionID); err != nil {
		return "", err
	}
	return filepath.Join(s.baseDir, executionID, "calls.jsonl"), nil
}

// Append appends a single LLM call to the execution's calls.jsonl file.
func (s *LLMCallFileStore) Append(ctx context.Context, executionID string, call *LLMCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath, err := s.filePath(executionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(call)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// AppendBatch appends multiple LLM calls to the execution's calls.jsonl file.
func (s *LLMCallFileStore) AppendBatch(ctx context.Context, executionID string, calls []*LLMCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath, err := s.filePath(executionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, call := range calls {
		data, err := json.Marshal(call)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// ListByExecution reads all LLM calls for a given execution.
func (s *LLMCallFileStore) ListByExecution(ctx context.Context, executionID string) ([]*LLMCall, error) {
	s.mu.Lock() // use mutex for read as well to prevent reading while writing
	defer s.mu.Unlock()

	filePath, err := s.filePath(executionID)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var calls []*LLMCall
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var call LLMCall
		if err := json.Unmarshal(scanner.Bytes(), &call); err != nil {
			slog.Warn("failed to unmarshal LLM call", "error", err)
			continue
		}
		calls = append(calls, &call)
	}
	return calls, scanner.Err()
}

// ListByExecutionPaginated reads LLM calls with pagination.
func (s *LLMCallFileStore) ListByExecutionPaginated(ctx context.Context, executionID string, offset, limit int) ([]*LLMCall, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath, err := s.filePath(executionID)
	if err != nil {
		return nil, 0, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var all []*LLMCall
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var call LLMCall
		if err := json.Unmarshal(scanner.Bytes(), &call); err != nil {
			slog.Warn("failed to unmarshal LLM call", "error", err)
			continue
		}
		all = append(all, &call)
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	total := len(all)

	if offset >= total {
		return []*LLMCall{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// GetCompletedRunIDs returns a map of run_id to true for all completed calls.
func (s *LLMCallFileStore) GetCompletedRunIDs(ctx context.Context, executionID string) (map[string]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath, err := s.filePath(executionID)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var call LLMCall
		if err := json.Unmarshal(scanner.Bytes(), &call); err != nil {
			slog.Warn("failed to unmarshal LLM call", "error", err)
			continue
		}
		if call.Status == "completed" {
			result[call.RunID] = true
		}
	}
	return result, scanner.Err()
}