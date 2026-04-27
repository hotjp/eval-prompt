package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"calculator", "Calculator"},
		{"github-agent", "Github Agent"},
		{"my_prompt", "My Prompt"},
		{"system-design-guide", "System Design Guide"},
		{"abc", "Abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindMainFile(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "handler.py preferred",
			files:    []string{"run.py", "handler.py", "utils.py"},
			expected: "handler.py",
		},
		{
			name:     "overview.md preferred for prompts",
			files:    []string{"part1.md", "overview.md", "part2.md"},
			expected: "overview.md",
		},
		{
			name:     "workflow.yaml preferred",
			files:    []string{"config.yaml", "workflow.yaml"},
			expected: "workflow.yaml",
		},
		{
			name:     "defaults to first",
			files:    []string{"a.py", "b.py", "c.py"},
			expected: "a.py",
		},
		{
			name:     "empty files",
			files:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findMainFile(tt.files, "handler.py", "overview.md", "workflow.yaml")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasFile(t *testing.T) {
	files := []string{"handler.py", "utils.py", "main.py"}

	assert.True(t, hasFile(files, "handler.py"))
	assert.True(t, hasFile(files, "main.py"))
	assert.False(t, hasFile(files, "other.py"))
	assert.False(t, hasFile([]string{}, "handler.py"))
}

func TestDetectAssetType(t *testing.T) {
	// Create temp directories for testing
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		files           []string
		wantAssetType   string
		wantMainFile    string
		wantErr         bool
	}{
		{
			name:          "skill with handler.py",
			files:         []string{"handler.py", "requirements.txt"},
			wantAssetType: "skill",
			wantMainFile:  "handler.py",
			wantErr:       false,
		},
		{
			name:          "skill with __init__.py",
			files:         []string{"__init__.py", "utils.py"},
			wantAssetType: "skill",
			wantMainFile:  "__init__.py",
			wantErr:       false,
		},
		{
			name:          "workflow with yaml",
			files:         []string{"workflow.yaml", "config.yaml"},
			wantAssetType: "workflow",
			wantMainFile:  "workflow.yaml",
			wantErr:       false,
		},
		{
			name:          "prompt with md",
			files:         []string{"overview.md", "part1.md"},
			wantAssetType: "prompt",
			wantMainFile:  "overview.md",
			wantErr:       false,
		},
		{
			name:          "agent md preferred",
			files:         []string{"prompt.md", "agent.md"},
			wantAssetType: "prompt",
			wantMainFile:  "agent.md",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tempDir, tt.name)
			os.MkdirAll(testDir, 0755)
			defer os.RemoveAll(testDir)

			// Create test files
			for _, f := range tt.files {
				os.WriteFile(filepath.Join(testDir, f), []byte("test"), 0644)
			}

			indexer := &Indexer{}
			assetType, mainFile, err := indexer.detectAssetType(testDir)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantAssetType, assetType)
				assert.Equal(t, tt.wantMainFile, mainFile)
			}
		})
	}
}

func TestGenerateAssetYAML(t *testing.T) {
	indexer := &Indexer{}

	ay := indexer.generateAssetYAML("calculator", "skill", "handler.py")

	assert.Equal(t, "skill", ay.AssetType)
	assert.Equal(t, "Calculator", ay.Name)
	assert.Equal(t, "handler.py", ay.Main)
	assert.Equal(t, "draft", ay.State)
	assert.Equal(t, "content", ay.Category)
	assert.Len(t, ay.Files, 1)
	assert.Equal(t, "handler.py", ay.Files[0].Path)
	assert.Equal(t, "main", ay.Files[0].Role)
}

func TestGenerateAssetYAML_Agent(t *testing.T) {
	indexer := &Indexer{}

	ay := indexer.generateAssetYAML("github-agent", "agent", "agent.md")

	assert.Equal(t, "agent", ay.AssetType)
	assert.Equal(t, "Github Agent", ay.Name)
	assert.Equal(t, "agent.md", ay.Main)
}

func TestMoveDirectoryContents(t *testing.T) {
	tempDir := t.TempDir()

	// Create source directory with files
	srcDir := filepath.Join(tempDir, "src")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0644)

	// Create subdirectory
	subDir := filepath.Join(srcDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file3.txt"), []byte("content3"), 0644)

	// Create destination directory
	dstDir := filepath.Join(tempDir, "dst")

	indexer := &Indexer{}
	err := indexer.moveDirectoryContents(srcDir, dstDir)
	assert.NoError(t, err)

	// Verify files were moved
	assert.FileExists(t, filepath.Join(dstDir, "file1.txt"))
	assert.FileExists(t, filepath.Join(dstDir, "file2.txt"))
	assert.FileExists(t, filepath.Join(dstDir, "subdir", "file3.txt"))

	// Verify source was cleaned up
	_, err = os.Stat(srcDir)
	assert.True(t, os.IsNotExist(err))
}
