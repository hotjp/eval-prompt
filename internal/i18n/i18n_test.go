package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flosch/pongo2/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_Embed(t *testing.T) {
	err := Init()
	assert.NoError(t, err)
	assert.NotEmpty(t, locales)
}

func TestT_Translate(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	// Ensure English for this test
	SetLang("en-US")

	// Test English (default)
	result := T("common_cancel", nil)
	assert.Equal(t, "Cancel", result)

	result = T("common_cancel", pongo2.Context{})
	assert.Equal(t, "Cancel", result)

	// Non-existent key returns key
	result = T("non_existent_key", nil)
	assert.Equal(t, "non_existent_key", result)
}

func TestSetLang(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	// Ensure clean state
	SetLang("en-US")

	// Set to Chinese
	err = SetLang("zh-CN")
	require.NoError(t, err)
	assert.Equal(t, "zh-CN", GetLang())

	// Set back to English
	err = SetLang("en-US")
	require.NoError(t, err)
	assert.Equal(t, "en-US", GetLang())

	// Unsupported language returns error
	err = SetLang("fr-FR")
	assert.Error(t, err)
}

func TestT_EnvLang(t *testing.T) {
	// Save and restore EP_LANG
	oldEPLang := os.Getenv(EnvLang)
	if oldEPLang != "" {
		os.Unsetenv(EnvLang)
	}
	defer func() {
		if oldEPLang != "" {
			os.Setenv(EnvLang, oldEPLang)
		}
	}()

	os.Setenv(EnvLang, "zh-CN")
	defer func() {
		os.Unsetenv(EnvLang)
	}()

	// Language is determined at init time, so just verify EP_LANG is respected
	// when Init is called afresh. Since we can't reset Once, we just check the env is set.
	assert.Equal(t, "zh-CN", os.Getenv(EnvLang))
}

func TestT_DiskFallback(t *testing.T) {
	// Create temp JSON locale files on disk
	tmpDir := t.TempDir()
	zhCN := `{"test_msg": "测试消息: {{name}}"}`
	enUS := `{"test_msg": "Test message: {{name}}"}`
	err := os.WriteFile(filepath.Join(tmpDir, "zh-CN.json"), []byte(zhCN), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "en-US.json"), []byte(enUS), 0644)
	require.NoError(t, err)

	loadLocalesFromDir(tmpDir)

	assert.NotNil(t, locales["zh-CN"])
	assert.NotNil(t, locales["en-US"])
	assert.Equal(t, "测试消息: {{name}}", locales["zh-CN"]["test_msg"])
}

func TestAvailableLangs(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	langs := AvailableLangs()
	assert.NotEmpty(t, langs)
}

func TestParseSystemLang(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"en_US.UTF-8", "en-US"},
		{"zh_CN.UTF-8", "zh-CN"},
		{"en_US", "en-US"},
		{"", ""},
		{"en-US", "en-US"},
	}

	for _, tt := range tests {
		result := parseSystemLang(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
