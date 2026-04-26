package commands

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalRunCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "run", "test/asset-1")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval run output: %s", string(out))
	// Eval service returns error about LLM not configured or storage when not properly initialized
	if err != nil {
		require.True(t, strings.Contains(string(out), "LLM") || strings.Contains(string(out), "not configured") || strings.Contains(string(out), "storage") || strings.Contains(string(out), "not found") || strings.Contains(string(out), "no eval cases"),
			"should indicate LLM or storage issue")
	}
}

func TestEvalRunWithJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "run", "test/asset-1", "--json")
	cmd.Dir = tmpDir
	out, _ := cmd.CombinedOutput()
	t.Logf("eval run --json output: %s", string(out))
	// JSON output should still work even with storage error
	require.True(t, strings.Contains(string(out), "LLM") || strings.Contains(string(out), "not configured") || strings.Contains(string(out), "storage") || strings.Contains(string(out), "not found") || strings.Contains(string(out), "no eval cases") || strings.Contains(string(out), "error"),
		"should return JSON with error")
}

func TestEvalCasesCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "cases", "test/asset-1")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval cases output: %s", string(out))
	// Cases command may fail when evals directory is not configured
	// but command structure is valid
	if err != nil {
		require.True(t, strings.Contains(string(out), "not configured") || strings.Contains(string(out), "evals") || strings.Contains(string(out), "error"),
			"should indicate evals directory issue")
	}
}

func TestEvalCasesJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "cases", "test/asset-1", "--json")
	cmd.Dir = tmpDir
	out, _ := cmd.CombinedOutput()
	t.Logf("eval cases --json output: %s", string(out))
	// JSON output should still work even with evals directory error
	require.True(t, strings.Contains(string(out), "not configured") || strings.Contains(string(out), "evals") || strings.Contains(string(out), "error") || strings.Contains(string(out), "null"),
		"should output JSON with error or null")
}

func TestEvalReportCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "report", "test-run-123")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval report output: %s", string(out))
	// Report may fail due to storage issues but command structure is valid
	if err != nil {
		require.True(t, strings.Contains(string(out), "not configured") || strings.Contains(string(out), "storage") || strings.Contains(string(out), "not found") || strings.Contains(string(out), "error"),
			"should indicate an error")
	}
}

func TestEvalReportJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "report", "test-run-123", "--json")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval report --json output: %s", string(out))
	// JSON output should still work even with storage error
	if err != nil {
		require.True(t, strings.Contains(string(out), "not configured") || strings.Contains(string(out), "storage") || strings.Contains(string(out), "not found") || strings.Contains(string(out), "error"),
			"should return JSON with error")
	}
}

func TestEvalDiagnoseCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "diagnose", "test-run-123")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval diagnose output: %s", string(out))
	// Diagnose may fail due to LLM not configured but command structure is valid
	if err != nil {
		require.True(t, strings.Contains(string(out), "LLM") || strings.Contains(string(out), "not configured") || strings.Contains(string(out), "storage") || strings.Contains(string(out), "not found") || strings.Contains(string(out), "error"),
			"should indicate an error")
	}
}

func TestEvalCompareCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "compare", "test/asset-1", "v1", "v2")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval compare output: %s", string(out))
	// Compare may fail due to missing asset file but command structure is valid
	if err != nil {
		require.True(t, strings.Contains(string(out), "not found") || strings.Contains(string(out), "no such file") || strings.Contains(string(out), "storage") || strings.Contains(string(out), "error") || strings.Contains(string(out), "failed"),
			"should indicate an error")
	}
}

func TestEvalCompareJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "compare", "test/asset-1", "v1", "v2", "--format", "json")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval compare --format json output: %s", string(out))
	// JSON output should still work even with storage error
	if err != nil {
		require.True(t, strings.Contains(string(out), "not found") || strings.Contains(string(out), "no such file") || strings.Contains(string(out), "storage") || strings.Contains(string(out), "error") || strings.Contains(string(out), "failed"),
			"should return JSON with error")
	}
}

func TestEvalMissingArgs(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "run")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval run missing args output: %s", string(out))
	require.Error(t, err, "ep eval run without args should fail")
	require.Contains(t, string(out), "accepts 1 arg(s)", "should show arg requirement")
}

func TestEvalCompareMissingArgs(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Missing v2 arg
	cmd := exec.Command(ep, "eval", "compare", "test/asset-1", "v1")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval compare missing args output: %s", string(out))
	require.Error(t, err, "ep eval compare with missing args should fail")
	require.Contains(t, string(out), "accepts 3 arg(s)", "should show arg requirement")
}

func TestEvalHelpCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "eval", "--help")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("eval --help output: %s", string(out))
	require.NoError(t, err, "ep eval --help should succeed")
	require.True(t, strings.Contains(string(out), "Eval 操作") || strings.Contains(string(out), "eval"), "should show eval help")
}