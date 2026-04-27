package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetYAML_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ay      *AssetYAML
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal asset",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test Prompt",
				Main:      "prompts/test/prompt.md",
			},
			wantErr: false,
		},
		{
			name: "valid full asset",
			ay: &AssetYAML{
				AssetType:   "skill",
				Name:        "Calculator",
				Main:        "skills/calculator/handler.py",
				Description: "A calculator skill",
				Tags:        []string{"math", "tool"},
				Category:    "content",
				State:       "published",
				MainFunction: "process",
				Files: []FileEntry{
					{Path: "skills/calculator/handler.py", Role: "main"},
					{Path: "skills/calculator/requirements.txt", Role: "config"},
				},
				External: []FileEntry{
					{Path: "shared-utils/common.py", Role: "lib"},
				},
				Version: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing asset_type",
			ay: &AssetYAML{
				Name: "Test",
				Main: "prompts/test/prompt.md",
			},
			wantErr: true,
			errMsg:  "asset_type is required",
		},
		{
			name: "invalid asset_type",
			ay: &AssetYAML{
				AssetType: "invalid",
				Name:      "Test",
				Main:      "prompts/test/prompt.md",
			},
			wantErr: true,
			errMsg:  "asset_type must be one of",
		},
		{
			name: "missing name",
			ay: &AssetYAML{
				AssetType: "prompt",
				Main:      "prompts/test/prompt.md",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing main",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test",
			},
			wantErr: true,
			errMsg:  "main is required",
		},
		{
			name: "invalid state",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test",
				Main:      "prompts/test/prompt.md",
				State:     "invalid",
			},
			wantErr: true,
			errMsg:  "state must be one of",
		},
		{
			name: "invalid category",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test",
				Main:      "prompts/test/prompt.md",
				Category:  "invalid",
			},
			wantErr: true,
			errMsg:  "category must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ay.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAssetYAML_DefaultState(t *testing.T) {
	tests := []struct {
		name      string
		ay        *AssetYAML
		wantState string
	}{
		{
			name: "empty state defaults to draft",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test",
				Main:      "prompts/test.md",
			},
			wantState: "draft",
		},
		{
			name: "published state preserved",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test",
				Main:      "prompts/test.md",
				State:     "published",
			},
			wantState: "published",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantState, tt.ay.DefaultState())
		})
	}
}

func TestAssetYAML_DefaultCategory(t *testing.T) {
	tests := []struct {
		name        string
		ay          *AssetYAML
		wantCategory string
	}{
		{
			name: "empty category defaults to content",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test",
				Main:      "prompts/test.md",
			},
			wantCategory: "content",
		},
		{
			name: "eval category preserved",
			ay: &AssetYAML{
				AssetType: "prompt",
				Name:      "Test",
				Main:      "prompts/test.md",
				Category:  "eval",
			},
			wantCategory: "eval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantCategory, tt.ay.DefaultCategory())
		})
	}
}

func TestAssetYAML_ResolveMain(t *testing.T) {
	tests := []struct {
		name        string
		ay          *AssetYAML
		repoPath    string
		wantPath    string
		wantExt     bool
		wantErr     bool
	}{
		{
			name: "repo relative path",
			ay: &AssetYAML{
				Main: "prompts/test/prompt.md",
			},
			repoPath: "/Users/king/repo",
			wantPath: "prompts/test/prompt.md",
			wantExt:  false,
			wantErr:  false,
		},
		{
			name: "absolute path is external",
			ay: &AssetYAML{
				Main: "/Users/king/.local/skills/handler.py",
			},
			repoPath: "/Users/king/repo",
			wantPath: "/Users/king/.local/skills/handler.py",
			wantExt:  true,
			wantErr:  false,
		},
		{
			name: "tilde path is external",
			ay: &AssetYAML{
				Main: "~/skills/handler.py",
			},
			repoPath: "/Users/king/repo",
			wantExt:  true,
			wantErr:  false,
		},
		{
			name: "empty main path error",
			ay: &AssetYAML{
				Main: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, isExt, err := tt.ay.ResolveMain(tt.repoPath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExt, isExt)
				if !tt.wantErr && tt.wantPath != "" {
					assert.Equal(t, tt.wantPath, path)
				}
			}
		})
	}
}

func TestAssetYAML_HasFile(t *testing.T) {
	ay := &AssetYAML{
		AssetType: "skill",
		Name:      "Calculator",
		Main:      "skills/calculator/handler.py",
		Files: []FileEntry{
			{Path: "skills/calculator/handler.py", Role: "main"},
			{Path: "skills/calculator/requirements.txt", Role: "config"},
		},
		External: []FileEntry{
			{Path: "shared-utils/common.py", Role: "lib"},
		},
	}

	assert.True(t, ay.HasFile("skills/calculator/handler.py"))
	assert.True(t, ay.HasFile("skills/calculator/requirements.txt"))
	assert.False(t, ay.HasFile("skills/calculator/other.py"))
}

func TestAssetYAML_HasExternal(t *testing.T) {
	ay := &AssetYAML{
		AssetType: "skill",
		Name:      "Calculator",
		Main:      "skills/calculator/handler.py",
		External: []FileEntry{
			{Path: "shared-utils/common.py", Role: "lib"},
			{Path: "shared-utils/helpers.py", Role: "lib"},
		},
	}

	assert.True(t, ay.HasExternal("shared-utils/common.py"))
	assert.False(t, ay.HasExternal("other/path/file.py"))
}

func TestAssetYAML_GetFileRole(t *testing.T) {
	ay := &AssetYAML{
		AssetType: "skill",
		Name:      "Calculator",
		Main:      "skills/calculator/handler.py",
		Files: []FileEntry{
			{Path: "skills/calculator/handler.py", Role: "main"},
			{Path: "skills/calculator/requirements.txt", Role: "config"},
		},
		External: []FileEntry{
			{Path: "shared-utils/common.py", Role: "lib"},
		},
	}

	assert.Equal(t, "main", ay.GetFileRole("skills/calculator/handler.py"))
	assert.Equal(t, "config", ay.GetFileRole("skills/calculator/requirements.txt"))
	assert.Equal(t, "lib", ay.GetFileRole("shared-utils/common.py"))
	assert.Equal(t, "", ay.GetFileRole("nonexistent/path.py"))
}

func TestParseAssetYAML(t *testing.T) {
	yamlContent := `
asset_type: skill
name: Calculator
main: skills/calculator/handler.py
description: A calculator skill
tags:
  - math
  - tool
category: content
state: published
main_function: process
files:
  - path: skills/calculator/handler.py
    role: main
  - path: skills/calculator/requirements.txt
    role: config
external:
  - path: shared-utils/common.py
    role: lib
version: "1.0.0"
`

	ay, err := ParseAssetYAML(yamlContent)
	require.NoError(t, err)
	assert.Equal(t, "skill", ay.AssetType)
	assert.Equal(t, "Calculator", ay.Name)
	assert.Equal(t, "skills/calculator/handler.py", ay.Main)
	assert.Equal(t, "A calculator skill", ay.Description)
	assert.Equal(t, []string{"math", "tool"}, ay.Tags)
	assert.Equal(t, "content", ay.Category)
	assert.Equal(t, "published", ay.State)
	assert.Equal(t, "process", ay.MainFunction)
	assert.Len(t, ay.Files, 2)
	assert.Len(t, ay.External, 1)
	assert.Equal(t, "1.0.0", ay.Version)
}

func TestSerializeAssetYAML(t *testing.T) {
	ay := &AssetYAML{
		AssetType:   "prompt",
		Name:        "Test Prompt",
		Main:        "prompts/test/prompt.md",
		Description: "A test prompt",
		Tags:        []string{"test"},
		State:       "draft",
		Category:    "content",
	}

	yamlStr, err := SerializeAssetYAML(ay)
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "asset_type: prompt")
	assert.Contains(t, yamlStr, "name: Test Prompt")
	assert.Contains(t, yamlStr, "main: prompts/test/prompt.md")
	assert.Contains(t, yamlStr, "description: A test prompt")
	assert.Contains(t, yamlStr, "tags:")
	assert.Contains(t, yamlStr, "- test")
}

func TestNewAssetYAML(t *testing.T) {
	ay := NewAssetYAML("prompt", "Test Prompt", "prompts/test/prompt.md")

	assert.Equal(t, "prompt", ay.AssetType)
	assert.Equal(t, "Test Prompt", ay.Name)
	assert.Equal(t, "prompts/test/prompt.md", ay.Main)
	assert.Equal(t, "draft", ay.State)
	assert.Equal(t, "content", ay.Category)
	assert.NotNil(t, ay.Metadata)
	assert.False(t, ay.Metadata.CreatedAt.IsZero())
}

func TestAssetYAMLFromFrontMatter(t *testing.T) {
	fm := &FrontMatter{
		ID:          "test-id",
		Name:        "Test Prompt",
		Description: "A test prompt",
		AssetType:   "prompt",
		Tags:        []string{"test", "demo"},
		Category:    "eval",
		State:       "published",
		Version:     "2.0.0",
	}

	ay := AssetYAMLFromFrontMatter(fm, "prompts/test/prompt.md")

	assert.Equal(t, "prompt", ay.AssetType)
	assert.Equal(t, "Test Prompt", ay.Name)
	assert.Equal(t, "prompts/test/prompt.md", ay.Main)
	assert.Equal(t, "A test prompt", ay.Description)
	assert.Equal(t, []string{"test", "demo"}, ay.Tags)
	assert.Equal(t, "eval", ay.Category)
	assert.Equal(t, "published", ay.State)
	assert.Equal(t, "2.0.0", ay.Version)
}

func TestAssetYAML_FileEntry_Roles(t *testing.T) {
	validRoles := []string{"main", "script", "config", "doc", "data", "lib", "template"}
	for _, role := range validRoles {
		ay := &AssetYAML{
			AssetType: "skill",
			Name:      "Test",
			Main:      "test/main.py",
			Files: []FileEntry{
				{Path: "test/main.py", Role: role},
			},
		}
		err := ay.Validate()
		assert.NoError(t, err, "Role %s should be valid", role)
	}
}

func TestAssetYAML_AssetTypes(t *testing.T) {
	validTypes := []string{"prompt", "skill", "agent", "mcp", "workflow", "knowledge"}
	for _, at := range validTypes {
		ay := &AssetYAML{
			AssetType: at,
			Name:      "Test",
			Main:      "test/main.md",
		}
		err := ay.Validate()
		assert.NoError(t, err, "AssetType %s should be valid", at)
	}
}

func TestAssetYAML_States(t *testing.T) {
	validStates := []string{"draft", "published", "archived", "deleted", "unavailable", ""}
	for _, state := range validStates {
		ay := &AssetYAML{
			AssetType: "prompt",
			Name:      "Test",
			Main:      "test/main.md",
			State:     state,
		}
		err := ay.Validate()
		assert.NoError(t, err, "State %q should be valid", state)
	}
}

func TestAssetYAML_Categories(t *testing.T) {
	validCategories := []string{"content", "eval", "metric", ""}
	for _, cat := range validCategories {
		ay := &AssetYAML{
			AssetType: "prompt",
			Name:      "Test",
			Main:      "test/main.md",
			Category:  cat,
		}
		err := ay.Validate()
		assert.NoError(t, err, "Category %q should be valid", cat)
	}
}
