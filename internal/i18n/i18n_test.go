package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load(t *testing.T) {
	// Create a temporary directory with test locale files
	tmpDir := t.TempDir()

	// Write test locale files
	zhCN := `
asset_create_success: "资产已创建: %s"
common_cancel: "取消"
`
	enUS := `
asset_create_success: "Asset created: %s"
common_cancel: "Cancel"
`
	err := os.WriteFile(filepath.Join(tmpDir, "zh-CN.yaml"), []byte(zhCN), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "en-US.yaml"), []byte(enUS), 0644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)
	err = loader.Load()
	require.NoError(t, err)

	// Test GetMessage
	msg, ok := loader.GetMessage("zh-CN", "asset_create_success")
	assert.True(t, ok)
	assert.Equal(t, "资产已创建: %s", msg)

	msg, ok = loader.GetMessage("en-US", "common_cancel")
	assert.True(t, ok)
	assert.Equal(t, "Cancel", msg)

	// Test non-existent key
	_, ok = loader.GetMessage("en-US", "non_existent")
	assert.False(t, ok)

	// Test non-existent language
	_, ok = loader.GetMessage("fr-FR", "asset_create_success")
	assert.False(t, ok)

	// Test AvailableLangs
	langs := loader.AvailableLangs()
	assert.ElementsMatch(t, []string{"zh-CN", "en-US"}, langs)
}

func TestI18n_New(t *testing.T) {
	// Save and clear LANG to ensure deterministic test
	oldLang := os.Getenv("LANG")
	os.Unsetenv("LANG")
	defer func() {
		if oldLang != "" {
			os.Setenv("LANG", oldLang)
		}
	}()

	tmpDir := t.TempDir()

	zhCN := `
asset_create_success: "资产已创建: %s"
common_error: "错误"
`
	enUS := `
asset_create_success: "Asset created: %s"
common_error: "Error"
`
	err := os.WriteFile(filepath.Join(tmpDir, "zh-CN.yaml"), []byte(zhCN), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "en-US.yaml"), []byte(enUS), 0644)
	require.NoError(t, err)

	i18n, err := New(tmpDir)
	require.NoError(t, err)

	// Default language should be en-US
	assert.Equal(t, "en-US", i18n.lang)
}

func TestI18n_SetLang(t *testing.T) {
	tmpDir := t.TempDir()

	zhCN := `
asset_create_success: "资产已创建: %s"
`
	enUS := `
asset_create_success: "Asset created: %s"
`
	err := os.WriteFile(filepath.Join(tmpDir, "zh-CN.yaml"), []byte(zhCN), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "en-US.yaml"), []byte(enUS), 0644)
	require.NoError(t, err)

	i18n, err := New(tmpDir)
	require.NoError(t, err)

	// Use global instance
	origGlobal := global
	global = i18n
	defer func() { global = origGlobal }()

	// Set to Chinese
	err = SetLang("zh-CN")
	require.NoError(t, err)
	assert.Equal(t, "zh-CN", GetLang())

	// Set back to English
	err = SetLang("en-US")
	require.NoError(t, err)
	assert.Equal(t, "en-US", GetLang())

	// Try to set unsupported language
	err = SetLang("fr-FR")
	assert.Error(t, err)
}

func TestT_Translate(t *testing.T) {
	// Save and clear LANG to ensure deterministic test
	oldLang := os.Getenv("LANG")
	os.Unsetenv("LANG")
	defer func() {
		if oldLang != "" {
			os.Setenv("LANG", oldLang)
		}
	}()

	tmpDir := t.TempDir()

	zhCN := `
asset_create_success: "资产已创建: %s"
common_cancel: "取消"
common_loading: "加载中..."
`
	enUS := `
asset_create_success: "Asset created: %s"
common_cancel: "Cancel"
common_loading: "Loading..."
`
	err := os.WriteFile(filepath.Join(tmpDir, "zh-CN.yaml"), []byte(zhCN), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "en-US.yaml"), []byte(enUS), 0644)
	require.NoError(t, err)

	i18n, err := New(tmpDir)
	require.NoError(t, err)

	// Use global instance
	origGlobal := global
	global = i18n
	defer func() { global = origGlobal }()

	// Test English (default)
	result := T("asset_create_success", "01ABC")
	assert.Equal(t, "Asset created: 01ABC", result)

	result = T("common_cancel")
	assert.Equal(t, "Cancel", result)

	// Switch to Chinese
	err = SetLang("zh-CN")
	require.NoError(t, err)

	result = T("asset_create_success", "01ABC")
	assert.Equal(t, "资产已创建: 01ABC", result)

	result = T("common_cancel")
	assert.Equal(t, "取消", result)

	// Test fallback - non-existent key returns key
	result = T("non_existent_key")
	assert.Equal(t, "non_existent_key", result)
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		params   []any
		expected string
	}{
		{
			name:     "no params",
			msg:      "Hello World",
			params:   []any{},
			expected: "Hello World",
		},
		{
			name:     "single param",
			msg:      "Hello %s",
			params:   []any{"Alice"},
			expected: "Hello Alice",
		},
		{
			name:     "multiple params",
			msg:      "Hello %s, you have %d messages",
			params:   []any{"Bob", 5},
			expected: "Hello Bob, you have 5 messages",
		},
		{
			name:     "param in middle",
			msg:      "Error: %s (code %d)",
			params:   []any{"Not Found", 404},
			expected: "Error: Not Found (code 404)",
		},
		{
			name:     "more params than placeholders",
			msg:      "Hello %s",
			params:   []any{"Alice", "extra"},
			expected: "Hello Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMessage(tt.msg, tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestT_EnvLang(t *testing.T) {
	// Save and clear LANG to ensure deterministic test (EP_LANG overrides LANG anyway)
	oldLang := os.Getenv("LANG")
	os.Unsetenv("LANG")
	defer func() {
		if oldLang != "" {
			os.Setenv("LANG", oldLang)
		}
	}()

	tmpDir := t.TempDir()

	zhCN := `
asset_create_success: "资产已创建: %s"
`
	enUS := `
asset_create_success: "Asset created: %s"
`
	err := os.WriteFile(filepath.Join(tmpDir, "zh-CN.yaml"), []byte(zhCN), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "en-US.yaml"), []byte(enUS), 0644)
	require.NoError(t, err)

	// Set EP_LANG environment variable
	oldEPLang := os.Getenv(EnvLang)
	defer func() {
		if oldEPLang != "" {
			os.Setenv(EnvLang, oldEPLang)
		} else {
			os.Unsetenv(EnvLang)
		}
	}()
	os.Setenv(EnvLang, "zh-CN")

	i18n, err := New(tmpDir)
	require.NoError(t, err)

	// Should default to zh-CN from EP_LANG
	assert.Equal(t, "zh-CN", i18n.lang)
}

func TestAvailableLangs(t *testing.T) {
	// Save and clear LANG to ensure deterministic test
	oldLang := os.Getenv("LANG")
	os.Unsetenv("LANG")
	defer func() {
		if oldLang != "" {
			os.Setenv("LANG", oldLang)
		}
	}()

	tmpDir := t.TempDir()

	zhCN := `asset_create_success: "资产已创建: %s"`
	enUS := `asset_create_success: "Asset created: %s"`
	deDE := `asset_create_success: "Asset erstellt: %s"`

	err := os.WriteFile(filepath.Join(tmpDir, "zh-CN.yaml"), []byte(zhCN), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "en-US.yaml"), []byte(enUS), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "de-DE.yaml"), []byte(deDE), 0644)
	require.NoError(t, err)

	i18n, err := New(tmpDir)
	require.NoError(t, err)

	// Use global instance
	origGlobal := global
	global = i18n
	defer func() { global = origGlobal }()

	langs := AvailableLangs()
	assert.ElementsMatch(t, []string{"zh-CN", "en-US", "de-DE"}, langs)
}
