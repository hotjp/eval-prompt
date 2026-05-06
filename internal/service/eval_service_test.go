package service

import (
	"context"
	"testing"
)

func TestEvalService_RunEval_NotConfigured(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.RunEval(ctx, &RunEvalRequest{AssetID: "asset-id", SnapshotVersion: "v1.0.0"})
	if err == nil {
		t.Error("expected error for not configured")
	}
	// RunEval now checks for LLM invoker first
	if err != nil && err.Error() == "LLM invoker not configured" {
		return // expected (LLM not configured in test)
	}
	t.Errorf("expected 'LLM invoker not configured', got %v", err)
}

func TestEvalService_GetEvalRun_NotConfigured(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.GetEvalRun(ctx, "run-id")
	if err == nil {
		t.Error("expected error for not configured")
	}
	// GetEvalRun requires prompts directory to exist
	if err != nil && err.Error() == "failed to read prompts directory: open prompts: no such file or directory" {
		return // expected (prompts directory doesn't exist in test)
	}
	t.Errorf("expected 'failed to read prompts directory', got %v", err)
}

func TestEvalService_ListEvalRuns_NotConfigured(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	runs, err := svc.ListEvalRuns(ctx, "asset-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When asset file does not exist, ListEvalRuns returns an empty slice
	if len(runs) != 0 {
		t.Errorf("expected empty runs for non-existent asset, got %d", len(runs))
	}
}

func TestEvalService_CompareEval_NotConfigured(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.CompareEval(ctx, "asset-id", "v1.0.0", "v2.0.0")
	if err == nil {
		t.Error("expected error for not configured")
	}
	// CompareEval requires prompts directory and asset file to exist
	if err != nil && err.Error() == "failed to read asset file: open prompts/asset-id.md: no such file or directory" {
		return // expected (asset file doesn't exist in test)
	}
	t.Errorf("expected 'failed to read asset file', got %v", err)
}

func TestEvalService_GenerateReport_NotConfigured(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.GenerateReport(ctx, "run-id")
	if err == nil {
		t.Error("expected error for not configured")
	}
	// GenerateReport now tries execution store and prompts directory
	if err != nil && (err.Error() == "prompts directory not found: open prompts: no such file or directory" || err.Error() == "eval run not found: run-id") {
		return // expected
	}
	t.Errorf("expected prompts directory not found or run not found, got %v", err)
}

func TestEvalService_DiagnoseEval_NotConfigured(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.DiagnoseEval(ctx, "run-id")
	if err == nil {
		t.Error("expected error for not configured")
	}
	// DiagnoseEval checks for LLM invoker first
	if err != nil && err.Error() == "LLM invoker not available" {
		return // expected (LLM not configured in test)
	}
	if err != nil && err.Error() == "diagnose eval requires file-based storage implementation" {
		return // expected
	}
	t.Errorf("expected 'LLM invoker not available' or 'diagnose eval requires file-based storage implementation', got %v", err)
}

func TestNotImplementedError(t *testing.T) {
	err := &NotImplementedError{Method: "TestMethod"}
	if err.Error() != "not implemented: TestMethod" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestErrNotImplemented(t *testing.T) {
	err := ErrNotImplemented
	if err.Method != "EvalService method" {
		t.Errorf("unexpected method: %s", err.Method)
	}
}