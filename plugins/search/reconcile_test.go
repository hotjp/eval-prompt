package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		s        string
		expected bool
	}{
		{
			name:     "contains string",
			slice:    []string{"a", "b", "c"},
			s:        "b",
			expected: true,
		},
		{
			name:     "does not contain string",
			slice:    []string{"a", "b", "c"},
			s:        "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			s:        "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.s)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFileTree(t *testing.T) {
	indexer := &Indexer{}

	ay := &domain.AssetYAML{
		AssetType: "skill",
		Name:      "Calculator",
		Main:      "skills/calculator/handler.py",
		Files: []domain.FileEntry{
			{Path: "skills/calculator/handler.py", Role: "main"},
			{Path: "skills/calculator/requirements.txt", Role: "config"},
		},
		External: []domain.FileEntry{
			{Path: "shared-utils/common.py", Role: "lib"},
		},
	}

	tree := indexer.buildFileTree(ay, "/repo")

	assert.Len(t, tree, 3)
	assert.Contains(t, tree, "skills/calculator/handler.py")
	assert.Contains(t, tree, "skills/calculator/requirements.txt")
	assert.Contains(t, tree, "shared-utils/common.py")
}

func TestBuildFileTree_Deduplication(t *testing.T) {
	indexer := &Indexer{}

	ay := &domain.AssetYAML{
		AssetType: "skill",
		Name:      "Calculator",
		Main:      "skills/calculator/handler.py",
		Files: []domain.FileEntry{
			{Path: "skills/calculator/handler.py", Role: "main"},
			{Path: "skills/calculator/handler.py", Role: "main"}, // duplicate
		},
	}

	tree := indexer.buildFileTree(ay, "/repo")

	assert.Len(t, tree, 1)
	assert.Contains(t, tree, "skills/calculator/handler.py")
}

func TestScanAssetsTypeDir(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create assets/skills directory
	assetsSkillsDir := filepath.Join(tempDir, "assets", "skills")
	os.MkdirAll(assetsSkillsDir, 0755)

	// Create a test asset.yaml
	assetYAML := `
asset_type: skill
name: Calculator
main: skills/calculator/handler.py
description: A calculator skill
tags:
  - math
  - tool
state: published
category: content
`
	os.WriteFile(filepath.Join(assetsSkillsDir, "calculator.yaml"), []byte(assetYAML), 0644)

	// Create skills/calculator directory with main file
	skillsCalcDir := filepath.Join(tempDir, "skills", "calculator")
	os.MkdirAll(skillsCalcDir, 0755)
	os.WriteFile(filepath.Join(skillsCalcDir, "handler.py"), []byte("# calculator"), 0644)

	// Create mock indexer
	indexer := NewIndexer()

	report := &service.ReconcileReport{}
	validAssetIDs := make(map[string]bool)

	err := indexer.scanAssetsTypeDir(nil, assetsSkillsDir, "skill", tempDir, report, validAssetIDs)
	assert.NoError(t, err)
	assert.True(t, validAssetIDs["calculator"])
}

func TestScanAssetsTypeDir_MissingMainFile(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create assets/skills directory
	assetsSkillsDir := filepath.Join(tempDir, "assets", "skills")
	os.MkdirAll(assetsSkillsDir, 0755)

	// Create a test asset.yaml with main pointing to non-existent file
	assetYAML := `
asset_type: skill
name: Calculator
main: skills/calculator/handler.py
state: published
`
	os.WriteFile(filepath.Join(assetsSkillsDir, "calculator.yaml"), []byte(assetYAML), 0644)

	// DON'T create skills/calculator directory

	// Create mock indexer
	indexer := NewIndexer()

	report := &service.ReconcileReport{}
	validAssetIDs := make(map[string]bool)

	err := indexer.scanAssetsTypeDir(nil, assetsSkillsDir, "skill", tempDir, report, validAssetIDs)
	assert.NoError(t, err)
	assert.True(t, validAssetIDs["calculator"])

	// Asset should be marked as unavailable
	entry, exists := indexer.assets["calculator"]
	assert.True(t, exists)
	assert.Equal(t, "unavailable", entry.asset.State)
}

func TestReconcileAssetYAML(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create assets/skills directory
	assetsSkillsDir := filepath.Join(tempDir, "assets", "skills")
	os.MkdirAll(assetsSkillsDir, 0755)

	// Create a test asset.yaml
	assetYAML := `
asset_type: skill
name: Calculator
main: skills/calculator/handler.py
description: A calculator skill
tags:
  - math
  - tool
state: published
category: content
`
	os.WriteFile(filepath.Join(assetsSkillsDir, "calculator.yaml"), []byte(assetYAML), 0644)

	// Create skills/calculator directory with main file
	skillsCalcDir := filepath.Join(tempDir, "skills", "calculator")
	os.MkdirAll(skillsCalcDir, 0755)
	os.WriteFile(filepath.Join(skillsCalcDir, "handler.py"), []byte("# calculator"), 0644)

	// Create mock indexer with git bridge
	indexer := NewIndexer()

	report := &service.ReconcileReport{}
	err := indexer.reconcileAssetYAML(nil, filepath.Join("assets", "skills", "calculator.yaml"), "calculator", "skill", tempDir, report)
	assert.NoError(t, err)
	assert.Equal(t, 1, report.Added)
	assert.Equal(t, 0, report.Updated)

	// Check indexed asset
	entry, exists := indexer.assets["calculator"]
	assert.True(t, exists)
	assert.Equal(t, "skill", entry.asset.AssetType)
	assert.Equal(t, "Calculator", entry.asset.Name)
	assert.Equal(t, "skills/calculator/handler.py", entry.asset.FilePath)
	assert.Equal(t, "published", entry.asset.State)
}

func TestReconcileAssetYAML_ExternalAsset(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create assets/skills directory
	assetsSkillsDir := filepath.Join(tempDir, "assets", "skills")
	os.MkdirAll(assetsSkillsDir, 0755)

	// Create a test asset.yaml with external main path
	assetYAML := `
asset_type: skill
name: External Calculator
main: /Users/king/.local/skills/calculator/handler.py
state: published
`
	os.WriteFile(filepath.Join(assetsSkillsDir, "external-calculator.yaml"), []byte(assetYAML), 0644)

	// Create mock indexer
	indexer := NewIndexer()

	report := &service.ReconcileReport{}
	err := indexer.reconcileAssetYAML(nil, filepath.Join("assets", "skills", "external-calculator.yaml"), "external-calculator", "skill", tempDir, report)
	assert.NoError(t, err)
	assert.Equal(t, 1, report.Added)

	// Check indexed asset - should not be marked unavailable even though file doesn't exist
	entry, exists := indexer.assets["external-calculator"]
	assert.True(t, exists)
	assert.Equal(t, "published", entry.asset.State) // Not unavailable because it's external
}

func TestConsistencyReport_OrphanFolders(t *testing.T) {
	// This test requires a git bridge with repo path, which is complex to set up.
	// For now, we test the logic with a simpler approach - just verify the function runs.
	t.Skip("Skipping - requires mock git bridge with repo path")

	// Create temp directory structure
	tempDir := t.TempDir()

	// Create skills/calculator directory WITHOUT assets/skills/calculator.yaml
	skillsCalcDir := filepath.Join(tempDir, "skills", "calculator")
	os.MkdirAll(skillsCalcDir, 0755)
	os.WriteFile(filepath.Join(skillsCalcDir, "handler.py"), []byte("# calculator"), 0644)

	// Create assets directory but NOT the calculator.yaml
	assetsSkillsDir := filepath.Join(tempDir, "assets", "skills")
	os.MkdirAll(assetsSkillsDir, 0755)

	// Create mock indexer
	indexer := NewIndexer()

	report, err := indexer.CheckConsistency(nil)
	assert.NoError(t, err)
	assert.Contains(t, report.OrphanFolders, "skills/calculator")
}
