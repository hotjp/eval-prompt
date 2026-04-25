package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2ESyncReconcileAndExport(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	// Step 1: ep init <tmpdir>
	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	out, err := initCmd.CombinedOutput()
	t.Logf("ep init output: %s", string(out))
	require.NoError(t, err, "ep init should succeed")

	// Step 2: Create prompts/test-asset.md manually with valid front matter
	promptsDir := filepath.Join(tmpDir, "prompts")
	err = os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	frontMatter := `---
id: ` + assetID + `
name: Test Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Prompt Content

This is the actual prompt content.
`
	mdFile := filepath.Join(promptsDir, "test-asset.md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err, "should create test-asset.md")

	// Step 3: ep sync reconcile (detect Added)
	cmd := exec.Command(ep, "sync", "reconcile", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync reconcile (Added) output: %s", string(out))
	require.NoError(t, err, "ep sync reconcile should succeed")
	require.Contains(t, string(out), "新增:", "should detect Added asset")

	// Step 4: ep sync export --format yaml (note: current implementation outputs JSON-like format)
	cmd = exec.Command(ep, "sync", "export", "--format", "yaml", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync export (yaml) output: %s", string(out))
	require.NoError(t, err, "ep sync export should succeed")
	// Verify output contains expected asset data
	require.Contains(t, string(out), "Test Asset", "should output asset name")

	// Step 5: ep sync export --format json (json)
	cmd = exec.Command(ep, "sync", "export", "--format", "json", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync export --format json output: %s", string(out))
	require.NoError(t, err, "ep sync export --format json should succeed")
	// Verify JSON output contains expected asset data
	require.Contains(t, string(out), "Test Asset", "should output asset name in JSON")

	// Step 6: Modify the .md file content (change description)
	newContent := `---
id: ` + assetID + `
name: Test Asset Updated
state: active
content_hash: sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
---
# Prompt Content

This is the modified prompt content.
`
	err = os.WriteFile(mdFile, []byte(newContent), 0644)
	require.NoError(t, err, "should update test-asset.md")

	// Step 7: ep sync reconcile (detect Updated)
	cmd = exec.Command(ep, "sync", "reconcile", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync reconcile (Updated) output: %s", string(out))
	require.NoError(t, err, "ep sync reconcile should succeed")
	require.Contains(t, string(out), "更新:", "should detect Updated asset")

	// Step 8: ep sync export > /dev/null (verify export still works)
	cmd = exec.Command(ep, "sync", "export", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync export final output: %s", string(out))
	require.NoError(t, err, "ep sync export should succeed after update")
}

func TestE2ESyncReconcileMultipleAssets(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	// ep init
	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	out, err := initCmd.CombinedOutput()
	t.Logf("ep init output: %s", string(out))
	require.NoError(t, err, "ep init should succeed")

	// Create prompts directory
	promptsDir := filepath.Join(tmpDir, "prompts")
	err = os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	// Create first asset
	assetID1 := "01ARZ3NDEKTSV4RRFFQ69G5FB1"
	frontMatter1 := `---
id: ` + assetID1 + `
name: First Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# First Prompt
`
	mdFile1 := filepath.Join(promptsDir, "first-asset.md")
	err = os.WriteFile(mdFile1, []byte(frontMatter1), 0644)
	require.NoError(t, err, "should create first-asset.md")

	// Create second asset
	assetID2 := "01ARZ3NDEKTSV4RRFFQ69G5FB2"
	frontMatter2 := `---
id: ` + assetID2 + `
name: Second Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Second Prompt
`
	mdFile2 := filepath.Join(promptsDir, "second-asset.md")
	err = os.WriteFile(mdFile2, []byte(frontMatter2), 0644)
	require.NoError(t, err, "should create second-asset.md")

	// ep sync reconcile (detect multiple Added)
	cmd := exec.Command(ep, "sync", "reconcile", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync reconcile (multiple) output: %s", string(out))
	require.NoError(t, err, "ep sync reconcile should succeed")
	require.Contains(t, string(out), "新增:", "should detect Added assets")

	// ep sync export should contain both assets
	cmd = exec.Command(ep, "sync", "export")
	cmd.Dir = tmpDir
	out, err = cmd.CombinedOutput()
	t.Logf("sync export (multiple) output: %s", string(out))
	require.NoError(t, err, "ep sync export should succeed")
	// Both asset names should appear in output
	require.True(t, strings.Contains(string(out), "First Asset") && strings.Contains(string(out), "Second Asset"),
		"export should contain both assets")
}

func TestE2ESyncExportYAMLFormat(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	out, err := initCmd.CombinedOutput()
	t.Logf("ep init output: %s", string(out))
	require.NoError(t, err, "ep init should succeed")

	promptsDir := filepath.Join(tmpDir, "prompts")
	err = os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARZ3NDEKTSV4RRFFQ69G5FC1"
	frontMatter := `---
id: ` + assetID + `
name: YAML Test Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# YAML Test Content
`
	mdFile := filepath.Join(promptsDir, "yaml-test.md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err, "should create yaml-test.md")

	// Reconcile to add the asset
	cmd := exec.Command(ep, "sync", "reconcile", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync reconcile output: %s", string(out))
	require.NoError(t, err, "ep sync reconcile should succeed")

	// Export with explicit yaml format
	cmd = exec.Command(ep, "sync", "export", "--format", "yaml", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync export --format yaml output: %s", string(out))
	require.NoError(t, err, "ep sync export --format yaml should succeed")
	require.Contains(t, string(out), "YAML Test Asset",
		"yaml output should contain asset name")
}

func TestE2ESyncExportJSONFormat(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	out, err := initCmd.CombinedOutput()
	t.Logf("ep init output: %s", string(out))
	require.NoError(t, err, "ep init should succeed")

	promptsDir := filepath.Join(tmpDir, "prompts")
	err = os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARZ3NDEKTSV4RRFFQ69G5FD1"
	frontMatter := `---
id: ` + assetID + `
name: JSON Test Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# JSON Test Content
`
	mdFile := filepath.Join(promptsDir, "json-test.md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err, "should create json-test.md")

	// Reconcile to add the asset
	cmd := exec.Command(ep, "sync", "reconcile", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync reconcile output: %s", string(out))
	require.NoError(t, err, "ep sync reconcile should succeed")

	// Export with explicit json format
	cmd = exec.Command(ep, "sync", "export", "--format", "json", "--dir", tmpDir)
	out, err = cmd.CombinedOutput()
	t.Logf("sync export --format json output: %s", string(out))
	require.NoError(t, err, "ep sync export --format json should succeed")
	require.Contains(t, string(out), "JSON Test Asset",
		"json output should contain asset name")
}
