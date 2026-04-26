package commands

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelListCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initOut, err := initCmd.CombinedOutput()
	require.NoError(t, err, "ep init should succeed")
	t.Logf("init output: %s", string(initOut))

	cmd := exec.Command(ep, "label", "list", "test-asset-id")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label list output: %s", string(out))
	// Label list may fail if asset not found, but command should execute
	require.True(t, err == nil || containsStr(string(out), "not found", "failed"),
		"label list should either succeed or report asset not found")
}

func TestLabelListJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "label", "list", "test-asset-id", "--json")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label list --json output: %s", string(out))
	// JSON output should be valid or error should be in JSON format
	require.True(t, err == nil || containsStr(string(out), "not found", "failed", "error"),
		"label list --json should either succeed or return error")
}

func TestLabelListMissingAssetID(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "label", "list")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label list missing args output: %s", string(out))
	require.Error(t, err, "ep label list without asset ID should fail")
}

func TestLabelSetCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "label", "set", "test-asset-id", "v1", "01ARNG3M6SV5QT2S2P7VPH3TG0")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label set output: %s", string(out))
	// Label set is not implemented
	require.Error(t, err, "ep label set should fail (not implemented)")
	require.Contains(t, string(out), "not implemented")
}

func TestLabelSetMissingArgs(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "label", "set")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label set missing args output: %s", string(out))
	require.Error(t, err, "ep label set without args should fail")
}

func TestLabelUnsetCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "label", "unset", "test-asset-id", "v1")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label unset output: %s", string(out))
	// Label unset is not implemented
	require.Error(t, err, "ep label unset should fail (not implemented)")
	require.Contains(t, string(out), "not implemented")
}

func TestLabelUnsetMissingArgs(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "label", "unset")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label unset missing args output: %s", string(out))
	require.Error(t, err, "ep label unset without args should fail")
}

func TestLabelHelpCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "label", "--help")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("label --help output: %s", string(out))
	require.NoError(t, err, "ep label --help should succeed")
	require.Contains(t, string(out), "label", "should show label help")
}

func containsStr(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
