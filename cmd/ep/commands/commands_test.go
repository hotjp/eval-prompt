package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// buildBinary builds the ep binary to a temp location and returns the path
func buildBinary(t *testing.T) string {
	wd, _ := os.Getwd()
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "ep")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join(repoRoot, "cmd", "ep")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("build output: %s, repoRoot: %s, wd: %s", string(out), repoRoot, wd)
	}
	require.NoError(t, err, "should build ep binary")
	err = os.Chmod(binaryPath, 0755)
	require.NoError(t, err, "should make binary executable")
	return binaryPath
}

func TestInitCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	cmd := exec.Command(ep, "init", tmpDir)
	out, err := cmd.CombinedOutput()
	t.Logf("output: %s", string(out))
	require.NoError(t, err, "ep init should succeed")

	promptsDir := filepath.Join(tmpDir, "prompts")
	_, err = os.Stat(promptsDir)
	require.NoError(t, err, "prompts directory should be created")
}

func TestServeCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "serve", "--port", "18080", "--no-browser")
	cmd.Dir = tmpDir

	err := cmd.Start()
	require.NoError(t, err, "ep serve should start")

	time.Sleep(2 * time.Second)

	if cmd.Process != nil {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	}
}

func TestSyncReconcile(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "sync", "reconcile")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("sync reconcile output: %s", string(out))
	require.NoError(t, err, "ep sync reconcile should succeed")
}

func TestInvalidCommand(t *testing.T) {
	ep := buildBinary(t)
	cmd := exec.Command(ep, "nonexistent-command")
	out, err := cmd.CombinedOutput()
	t.Logf("invalid command output: %s", string(out))
	require.Error(t, err, "unknown command should fail")
}

func TestInitMissingPath(t *testing.T) {
	ep := buildBinary(t)
	cmd := exec.Command(ep, "init")
	out, err := cmd.CombinedOutput()
	t.Logf("init missing path output: %s", string(out))
	require.Error(t, err, "ep init without path should fail")
}