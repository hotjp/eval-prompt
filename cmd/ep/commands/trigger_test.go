package commands

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTriggerMatchCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "trigger", "match", "hello world")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("trigger match output: %s", string(out))
	require.NoError(t, err, "ep trigger match should succeed")
}

func TestTriggerMatchJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "trigger", "match", "hello world", "--json")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("trigger match --json output: %s", string(out))
	require.NoError(t, err, "ep trigger match --json should succeed")
}

func TestTriggerMatchWithTop(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "trigger", "match", "hello world", "--top", "3")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("trigger match --top 3 output: %s", string(out))
	require.NoError(t, err, "ep trigger match --top should succeed")
}

func TestTriggerMatchMissingInput(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "trigger", "match")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("trigger match missing input output: %s", string(out))
	require.Error(t, err, "ep trigger match without input should fail")
}

func TestTriggerMatchWithPrompt(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Test with longer prompt text
	cmd := exec.Command(ep, "trigger", "match", "code review for API endpoint")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("trigger match with prompt output: %s", string(out))
	require.NoError(t, err, "ep trigger match should succeed with prompt input")
}

func TestTriggerMatchTableOutput(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "trigger", "match", "test prompt")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("trigger match table output: %s", string(out))
	require.NoError(t, err, "ep trigger match should output table format")
	require.Contains(t, string(out), "ID", "should contain table header")
}

func TestTriggerMatchEmptyIndex(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Match against empty index should return empty results
	cmd := exec.Command(ep, "trigger", "match", "nonexistent query")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("trigger match empty index output: %s", string(out))
	require.NoError(t, err, "ep trigger match should succeed even with empty index")
}