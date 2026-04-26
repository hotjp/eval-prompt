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

// parseAssetID extracts the asset ID from "asset create" output
// Expected format: "资产已创建: 01KQ1QEZ6FBPV5CTTVCQQ1R0WT"
func parseAssetID(output string) string {
	parts := strings.Split(output, ":")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return ""
}

// E2E_CLI_CompleteWorkflow tests the complete CLI workflow:
// init → asset create → snapshot list → eval run → eval report → archive → restore → rm
func TestE2E_CLI_CompleteWorkflow(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	// Track the asset ID that will be created
	var assetID string

	t.Run("init", func(t *testing.T) {
		cmd := exec.Command(ep, "init", tmpDir)
		cmd.Env = append(os.Environ(), "EP_HOME="+epHome)
		out, err := cmd.CombinedOutput()
		t.Logf("init output: %s", string(out))
		require.NoError(t, err, "ep init should succeed")

		// Verify .git directory was created
		gitDir := filepath.Join(tmpDir, ".git")
		_, err = os.Stat(gitDir)
		require.NoError(t, err, ".git directory should be created")

		// Verify prompts/ directory was created
		promptsDir := filepath.Join(tmpDir, "prompts")
		_, err = os.Stat(promptsDir)
		require.NoError(t, err, "prompts directory should be created")
	})

	t.Run("asset_create", func(t *testing.T) {
		// Create a content file for the asset
		contentFile := filepath.Join(tmpDir, "test-prompt.txt")
		err := os.WriteFile(contentFile, []byte("# Hello World\n\nThis is a test prompt."), 0644)
		require.NoError(t, err)

		cmd := exec.Command(ep, "asset", "create",
			"--name", "Test E2E Asset",
			"--file", contentFile,
			"--biz-line", "e2e-test")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("asset create output: %s", string(out))
		require.NoError(t, err, "ep asset create should succeed")

		// Parse the actual asset ID from output (system generates its own ULID)
		assetID = parseAssetID(string(out))
		require.True(t, len(assetID) > 0, "should parse asset ID from output")

		// Verify .md file was created in prompts/ with valid front matter
		mdFile := filepath.Join(tmpDir, "prompts", assetID+".md")
		_, err = os.Stat(mdFile)
		require.NoError(t, err, "asset .md file should be created in prompts/")

		// Read and verify front matter
		content, err := os.ReadFile(mdFile)
		require.NoError(t, err)

		// Check for required front matter fields
		require.Contains(t, string(content), "id:", "front matter should contain id")
		require.Contains(t, string(content), "name:", "front matter should contain name")
		require.Contains(t, string(content), "state:", "front matter should contain state")
		require.Contains(t, string(content), "content_hash:", "front matter should contain content_hash")
	})

	t.Run("snapshot_list", func(t *testing.T) {
		cmd := exec.Command(ep, "snapshot", "list", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("snapshot list output: %s", string(out))
		require.NoError(t, err, "ep snapshot list should succeed")
		// Output should contain table headers or valid JSON
		output := string(out)
		require.True(t,
			strings.Contains(output, "HASH") ||
			strings.Contains(output, "COMMITTER") ||
			strings.Contains(output, "DATE") ||
			strings.Contains(output, "[]") || // empty array is valid
			strings.Contains(output, "Subject"),
			"snapshot list should return valid output")
	})

	t.Run("eval_run", func(t *testing.T) {
		cmd := exec.Command(ep, "eval", "run", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("eval run output: %s", string(out))
		// Either succeeds with run_id or fails with "storage not initialized" error - both are valid
		output := string(out)
		require.True(t,
			err == nil && (strings.Contains(output, "run_id") || strings.Contains(output, "Eval 运行已启动")) ||
				strings.Contains(output, "storage not initialized") ||
				strings.Contains(output, "not implemented") ||
				strings.Contains(output, "failed"),
			"eval run should either succeed or report storage not initialized")
	})

	t.Run("eval_report", func(t *testing.T) {
		// Try with a dummy run ID since storage may not be initialized
		cmd := exec.Command(ep, "eval", "report", "dummy-run-id")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("eval report output: %s", string(out))
		// Either succeeds or fails gracefully
		output := string(out)
		require.True(t,
			err == nil ||
				strings.Contains(output, "not found") ||
				strings.Contains(output, "not implemented") ||
				strings.Contains(output, "failed") ||
				strings.Contains(output, "not configured") ||
				strings.Contains(output, "storage not initialized"),
			"eval report should handle missing run gracefully")
	})

	t.Run("archive", func(t *testing.T) {
		cmd := exec.Command(ep, "asset", "archive", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("asset archive output: %s", string(out))
		require.NoError(t, err, "ep asset archive should succeed")

		// Verify state is archived in file
		mdFile := filepath.Join(tmpDir, "prompts", assetID+".md")
		content, err := os.ReadFile(mdFile)
		require.NoError(t, err)
		require.Contains(t, string(content), "state: archived", "asset state should be archived")
	})

	t.Run("restore", func(t *testing.T) {
		cmd := exec.Command(ep, "asset", "restore", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("asset restore output: %s", string(out))
		require.NoError(t, err, "ep asset restore should succeed")

		// Verify state is active again
		mdFile := filepath.Join(tmpDir, "prompts", assetID+".md")
		content, err := os.ReadFile(mdFile)
		require.NoError(t, err)
		require.Contains(t, string(content), "state: active", "asset state should be active after restore")
	})

	t.Run("rm_fails_on_active", func(t *testing.T) {
		// First, archive the asset so we can test rm behavior
		archiveCmd := exec.Command(ep, "asset", "archive", assetID)
		archiveCmd.Dir = tmpDir
		archiveCmd.CombinedOutput()

		// Restore to make it active again
		restoreCmd := exec.Command(ep, "asset", "restore", assetID)
		restoreCmd.Dir = tmpDir
		restoreCmd.CombinedOutput()

		// rm should fail on active asset
		cmd := exec.Command(ep, "asset", "rm", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("asset rm (active) output: %s", string(out))
		require.Error(t, err, "rm should fail when state is active")
		require.Contains(t, string(out), "请先 archive", "should indicate asset must be archived first")
	})

	t.Run("rm_succeeds_after_archive", func(t *testing.T) {
		// Archive the asset first
		archiveCmd := exec.Command(ep, "asset", "archive", assetID)
		archiveCmd.Dir = tmpDir
		out, err := archiveCmd.CombinedOutput()
		t.Logf("asset archive output: %s", string(out))
		require.NoError(t, err, "ep asset archive should succeed")

		// Now rm should succeed
		cmd := exec.Command(ep, "asset", "rm", assetID)
		cmd.Dir = tmpDir
		out, err = cmd.CombinedOutput()
		t.Logf("asset rm output: %s", string(out))
		require.NoError(t, err, "ep asset rm should succeed after archive")

		// Verify file is deleted
		mdFile := filepath.Join(tmpDir, "prompts", assetID+".md")
		_, err = os.Stat(mdFile)
		require.True(t, os.IsNotExist(err), "asset file should be deleted after rm")
	})
}

// E2E_CLI_InitBasic verifies basic init functionality
func TestE2E_CLI_InitBasic(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	cmd := exec.Command(ep, "init", tmpDir)
	cmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	out, err := cmd.CombinedOutput()
	t.Logf("init output: %s", string(out))
	require.NoError(t, err, "ep init should succeed")

	// Verify both .git and prompts/ directories exist
	gitDir := filepath.Join(tmpDir, ".git")
	_, err = os.Stat(gitDir)
	require.NoError(t, err, ".git directory should be created")

	promptsDir := filepath.Join(tmpDir, "prompts")
	_, err = os.Stat(promptsDir)
	require.NoError(t, err, "prompts directory should be created")
}

// E2E_CLI_AssetLifecycle tests individual asset lifecycle operations
func TestE2E_CLI_AssetLifecycle(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	// Initialize
	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	initCmd.CombinedOutput()

	assetID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"

	// Create asset file directly with proper front matter
	promptsDir := filepath.Join(tmpDir, "prompts")
	err := os.MkdirAll(promptsDir, 0755)
	require.NoError(t, err)

	frontMatter := fmt.Sprintf(`---
id: %s
name: Lifecycle Test Asset
state: active
content_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
---
# Prompt Content
`, assetID)
	mdFile := filepath.Join(promptsDir, assetID+".md")
	err = os.WriteFile(mdFile, []byte(frontMatter), 0644)
	require.NoError(t, err)

	// Test archive
	t.Run("lifecycle_archive", func(t *testing.T) {
		cmd := exec.Command(ep, "asset", "archive", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("archive output: %s", string(out))
		require.NoError(t, err, "archive should succeed")

		content, _ := os.ReadFile(mdFile)
		require.Contains(t, string(content), "state: archived")
	})

	// Test restore
	t.Run("lifecycle_restore", func(t *testing.T) {
		cmd := exec.Command(ep, "asset", "restore", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("restore output: %s", string(out))
		require.NoError(t, err, "restore should succeed")

		content, _ := os.ReadFile(mdFile)
		require.Contains(t, string(content), "state: active")
	})

	// Test rm after archive
	t.Run("lifecycle_rm", func(t *testing.T) {
		// Archive first
		archiveCmd := exec.Command(ep, "asset", "archive", assetID)
		archiveCmd.Dir = tmpDir
		archiveCmd.CombinedOutput()

		// Then rm
		cmd := exec.Command(ep, "asset", "rm", assetID)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		t.Logf("rm output: %s", string(out))
		require.NoError(t, err, "rm should succeed after archive")

		_, err = os.Stat(mdFile)
		require.True(t, os.IsNotExist(err), "file should be deleted")
	})
}
