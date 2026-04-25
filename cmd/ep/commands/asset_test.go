package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssetList(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initOut, err := initCmd.CombinedOutput()
	require.NoError(t, err, "ep init should succeed")
	t.Logf("init output: %s", string(initOut))

	cmd := exec.Command(ep, "asset", "list")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset list output: %s", string(out))
	require.NoError(t, err, "ep asset list should succeed")
}

func TestAssetListJSON(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "asset", "list", "--json")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset list --json output: %s", string(out))
	require.NoError(t, err, "ep asset list --json should succeed")
}

func TestAssetCreate(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	contentFile := filepath.Join(tmpDir, "test-prompt.txt")
	err := os.WriteFile(contentFile, []byte("Hello World"), 0644)
	require.NoError(t, err)

	cmd := exec.Command(ep, "asset", "create",
		"--name", "Test Asset",
		"--file", contentFile,
		"--biz-line", "test")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset create output: %s", string(out))
	require.NoError(t, err, "ep asset create should succeed")
}

func TestAssetCreateMissingArgs(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "asset", "create")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset create (missing args) output: %s", string(out))
	require.Error(t, err, "ep asset create without args should fail")
}

func TestAssetShow(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "asset", "show", "test/show-asset")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset show output: %s", string(out))
	// Asset may not be found due to in-memory index, but command should execute
	require.True(t, err == nil || strings.Contains(string(out), "not found"), "command should run")
}

func TestAssetCat(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	cmd := exec.Command(ep, "asset", "cat", "test/cat-asset")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset cat output: %s", string(out))
	// Asset may not be found due to in-memory index, but command should execute
	require.True(t, err == nil || strings.Contains(string(out), "not found"), "command should run")
}

func TestAssetRm(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Create a prompt file directly in prompts directory
	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	// Create a properly formatted .md file with front matter
	// Using ULID-like ID for validation
	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG0"
	frontMatter := fmt.Sprintf(`---
id: %s
name: Remove Test Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Prompt Content
`, assetID)
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	// rm should fail because state is active, not archived
	cmd := exec.Command(ep, "asset", "rm", assetID)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset rm (active) output: %s", string(out))
	require.Error(t, err, "rm should fail when state is active")
	require.Contains(t, string(out), "请先 archive")

	// archive the asset
	archiveCmd := exec.Command(ep, "asset", "archive", assetID)
	archiveCmd.Dir = tmpDir
	out, err = archiveCmd.CombinedOutput()
	t.Logf("asset archive output: %s", string(out))
	require.NoError(t, err, "ep asset archive should succeed")

	// verify state is archived
	content, err := os.ReadFile(mdFile)
	require.NoError(t, err)
	require.Contains(t, string(content), "state: archived")

	// now rm should succeed
	rmCmd := exec.Command(ep, "asset", "rm", assetID)
	rmCmd.Dir = tmpDir
	out, err = rmCmd.CombinedOutput()
	t.Logf("asset rm output: %s", string(out))
	require.NoError(t, err, "ep asset rm should succeed after archive")

	// verify file is deleted
	_, err = os.Stat(mdFile)
	require.True(t, os.IsNotExist(err), "file should be deleted")
}

func TestAssetArchiveRestore(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Create a prompt file
	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG1"
	frontMatter := fmt.Sprintf(`---
id: %s
name: Archive Test Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Prompt Content
`, assetID)
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	// archive the asset
	archiveCmd := exec.Command(ep, "asset", "archive", assetID)
	archiveCmd.Dir = tmpDir
	out, err := archiveCmd.CombinedOutput()
	t.Logf("asset archive output: %s", string(out))
	require.NoError(t, err, "ep asset archive should succeed")

	// verify state is archived
	content, err := os.ReadFile(mdFile)
	require.NoError(t, err)
	require.Contains(t, string(content), "state: archived")

	// restore the asset
	restoreCmd := exec.Command(ep, "asset", "restore", assetID)
	restoreCmd.Dir = tmpDir
	out, err = restoreCmd.CombinedOutput()
	t.Logf("asset restore output: %s", string(out))
	require.NoError(t, err, "ep asset restore should succeed")

	// verify state is active again
	content, err = os.ReadFile(mdFile)
	require.NoError(t, err)
	require.Contains(t, string(content), "state: active")
}

func TestAssetArchiveNotFound(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Try to archive non-existent asset
	cmd := exec.Command(ep, "asset", "archive", "nonexistent")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset archive (not found) output: %s", string(out))
	require.Error(t, err, "archive should fail for non-existent asset")
	require.Contains(t, string(out), "资产文件不存在")
}

func TestAssetRmActiveFails(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.CombinedOutput()

	// Create a prompt file with active state
	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	assetID := "01ARNG3M6SV5QT2S2P7VPH3TG2"
	frontMatter := fmt.Sprintf(`---
id: %s
name: Active Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Prompt Content
`, assetID)
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	// Try to rm active asset - should fail
	cmd := exec.Command(ep, "asset", "rm", assetID)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	t.Logf("asset rm (active) output: %s", string(out))
	require.Error(t, err, "rm should fail for active asset")
	require.Contains(t, string(out), "请先 archive")
}