package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncExportCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "sync", "export")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("sync export output: %s", string(out))
	require.NoError(t, err, "ep sync export should succeed")
}

func TestSyncExportJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "sync", "export", "--format", "json")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("sync export --format json output: %s", string(out))
	require.NoError(t, err, "ep sync export --format json should succeed")
	// JSON output should contain json-like structure
	require.True(t, strings.Contains(string(out), "{") || strings.Contains(string(out), "null") || strings.Contains(string(out), "[]"),
		"should output JSON format")
}

func TestSyncExportYAML(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "sync", "export", "--format", "yaml")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("sync export --format yaml output: %s", string(out))
	require.NoError(t, err, "ep sync export --format yaml should succeed")
}

func TestSyncExportToFile(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	outputFile := filepath.Join(tmpDir, "export.json")
	cmd := exec.Command(ep, "sync", "export", "--output", outputFile)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("sync export --output output: %s", string(out))
	require.NoError(t, err, "ep sync export --output should succeed")

	// Verify file was created and has content
	info, err := os.Stat(outputFile)
	require.NoError(t, err, "export file should be created")
	require.Greater(t, info.Size(), int64(0), "export file should have content")
}

func TestSyncReconcileAgain(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "sync", "reconcile")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("sync reconcile output: %s", string(out))
	require.NoError(t, err, "ep sync reconcile should succeed")
	require.Contains(t, string(out), "对账完成", "should show reconcile completion")
}

func TestSyncHelpCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "sync", "--help")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("sync --help output: %s", string(out))
	require.NoError(t, err, "ep sync --help should succeed")
	require.Contains(t, string(out), "sync", "should show sync help")
}
