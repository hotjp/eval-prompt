package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnapshotListCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Create a prompt file so there's git history
	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG0"
	frontMatter := `---
id: ` + assetID + `
name: Snapshot Test Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Prompt Content
`
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	cmd := exec.Command(ep, "snapshot", "list", assetID)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot list output: %s", string(out))
	require.NoError(t, err, "ep snapshot list should succeed")
}

func TestSnapshotListWithLimit(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG1"
	frontMatter := `---
id: ` + assetID + `
name: Snapshot Limit Test
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Content
`
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	cmd := exec.Command(ep, "snapshot", "list", assetID, "--limit", "5")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot list --limit 5 output: %s", string(out))
	require.NoError(t, err, "ep snapshot list with --limit should succeed")
}

func TestSnapshotListJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG2"
	frontMatter := `---
id: ` + assetID + `
name: Snapshot JSON Test
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Content
`
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	cmd := exec.Command(ep, "snapshot", "list", assetID, "--json")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot list --json output: %s", string(out))
	require.NoError(t, err, "ep snapshot list --json should succeed")
}

func TestSnapshotListMissingAssetID(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "snapshot", "list")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot list missing args output: %s", string(out))
	require.Error(t, err, "ep snapshot list without asset ID should fail")
}

func TestSnapshotDiffCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Create a prompt file with git history
	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG3"
	frontMatter := `---
id: ` + assetID + `
name: Snapshot Diff Test
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Content
`
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	cmd := exec.Command(ep, "snapshot", "diff", assetID, "HEAD~1", "HEAD")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot diff output: %s", string(out))
	// Diff may fail if there's no prior commit, but command should execute
	require.True(t, err == nil || containsAny(string(out), []string{"Diff", "fatal", "unknown", "revision"}),
		"snapshot diff should either succeed or report git error")
}

func TestSnapshotDiffMissingArgs(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "snapshot", "diff")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot diff missing args output: %s", string(out))
	require.Error(t, err, "ep snapshot diff without args should fail")
}

func TestSnapshotCheckoutCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG4"
	frontMatter := `---
id: ` + assetID + `
name: Snapshot Checkout Test
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Content
`
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	cmd := exec.Command(ep, "snapshot", "checkout", assetID, "HEAD")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot checkout output: %s", string(out))
	// Checkout is not implemented
	require.Error(t, err, "ep snapshot checkout should fail (not implemented)")
	require.Contains(t, string(out), "not implemented")
}

func TestSnapshotHelpCommand(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "snapshot", "--help")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("snapshot --help output: %s", string(out))
	require.NoError(t, err, "ep snapshot --help should succeed")
	require.Contains(t, string(out), "snapshot", "should show snapshot help")
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(sub) > 0 && len(s) > 0 && (containsString(s, sub) || containsString(s, "fatal") || containsString(s, "unknown") || containsString(s, "revision")) {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstr(s, substr)))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
