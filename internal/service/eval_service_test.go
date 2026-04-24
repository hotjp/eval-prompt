package service

import (
	"context"
	"errors"
	"testing"
)

func TestEvalService_RunEval_NotImplemented(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.RunEval(ctx, "asset-id", "v1.0.0", nil)
	if err == nil {
		t.Error("expected error for not implemented")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestEvalService_GetEvalRun_NotImplemented(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.GetEvalRun(ctx, "run-id")
	if err == nil {
		t.Error("expected error for not implemented")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestEvalService_ListEvalRuns_NotImplemented(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.ListEvalRuns(ctx, "asset-id")
	if err == nil {
		t.Error("expected error for not implemented")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestEvalService_CompareEval_NotImplemented(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.CompareEval(ctx, "asset-id", "v1.0.0", "v2.0.0")
	if err == nil {
		t.Error("expected error for not implemented")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestEvalService_GenerateReport_NotImplemented(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.GenerateReport(ctx, "run-id")
	if err == nil {
		t.Error("expected error for not implemented")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestEvalService_DiagnoseEval_NotImplemented(t *testing.T) {
	svc := NewEvalService()
	ctx := context.Background()

	_, err := svc.DiagnoseEval(ctx, "run-id")
	if err == nil {
		t.Error("expected error for not implemented")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
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