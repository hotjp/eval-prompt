package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// EnvLang is the environment variable for setting the language.
	EnvLang = "EP_LANG"

	// DefaultLang is the default language when none is set.
	DefaultLang = "en-US"

	// DefaultLocalesDir is the default directory for language packs.
	DefaultLocalesDir = "i18n/locales"
)

// global holds the global i18n state.
var (
	global     *I18n
	globalOnce sync.Once
)

// I18n provides internationalization support.
type I18n struct {
	mu         sync.RWMutex
	lang       string
	loader     *Loader
	fallback   map[string]string // fallback messages when key not found in current lang
}

// current returns the global I18n instance.
func current() *I18n {
	return global
}

// Init initializes the global i18n with the default locales directory.
func Init() error {
	dir, err := findLocalesDir()
	if err != nil {
		return err
	}
	return InitWithDir(dir)
}

// InitWithDir initializes the global i18n with the specified locales directory.
func InitWithDir(localesDir string) error {
	var err error
	globalOnce.Do(func() {
		global, err = New(localesDir)
	})
	return err
}

// New creates a new I18n instance with the specified locales directory.
// Language priority: EP_LANG > system LANG > DefaultLang (en-US)
func New(localesDir string) (*I18n, error) {
	loader := NewLoader(localesDir)
	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("failed to load locales: %w", err)
	}

	i := &I18n{
		lang:     DefaultLang,
		loader:   loader,
		fallback: getFallbackMessages(),
	}

	// Priority 1: EP_LANG environment variable (explicit override)
	if lang := os.Getenv(EnvLang); lang != "" {
		if loader.HasLang(lang) {
			i.lang = lang
		}
	}

	// Priority 2: System LANG environment variable (e.g., "en_US.UTF-8")
	// Only used if no EP_LANG was set
	if os.Getenv(EnvLang) == "" {
		if sysLang := parseSystemLang(os.Getenv("LANG")); sysLang != "" {
			if loader.HasLang(sysLang) {
				i.lang = sysLang
			}
		}
	}

	return i, nil
}

// parseSystemLang converts system LANG to i18n language code.
// Examples: "en_US.UTF-8" -> "en-US", "zh_CN.UTF-8" -> "zh-CN"
func parseSystemLang(lang string) string {
	if lang == "" {
		return ""
	}
	// Strip encoding suffix (.UTF-8, .utf8, etc.)
	if idx := strings.Index(lang, "."); idx != -1 {
		lang = lang[:idx]
	}
	// Convert en_US -> en-US
	return strings.ReplaceAll(lang, "_", "-")
}

// InitWithLang initializes with explicit language setting.
// Use this when you want to set a language from config before Init,
// but still want EP_LANG to override if set.
// Usage:
//   i18n.Init()                           // EP_LANG > system LANG > en-US
//   i18n.SetLangIfNotEnv(cfg.Lang)       // if no EP_LANG, use config lang
func InitWithLang(localesDir, lang string) error {
	globalOnce.Do(func() {
		var err error
		global, err = New(localesDir)
		if err != nil {
			return
		}
		// This will be overridden by SetLangIfNotEnv if EP_LANG is set
		if lang != "" && global.loader.HasLang(lang) {
			global.mu.Lock()
			global.lang = lang
			global.mu.Unlock()
		}
	})
	return nil
}

// findLocalesDir looks for the locales directory.
func findLocalesDir() (string, error) {
	// Check EP_LOCALES_DIR environment variable first
	if dir := os.Getenv("EP_LOCALES_DIR"); dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		}
	}

	// Try relative to current working directory
	cwd, err := os.Getwd()
	if err == nil {
		dir := filepath.Join(cwd, DefaultLocalesDir)
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		}
	}

	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), DefaultLocalesDir)
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		}
	}

	// Try relative to module root (for development)
	modRoot, err := findModuleRoot()
	if err == nil {
		dir := filepath.Join(modRoot, DefaultLocalesDir)
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		}
	}

	return "", fmt.Errorf("locales directory not found")
}

// findModuleRoot finds the Go module root directory.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

// getFallbackMessages returns default fallback messages.
func getFallbackMessages() map[string]string {
	return map[string]string{
		MsgAssetCreateSuccess:   "Asset created: %s",
		MsgAssetArchiveSuccess:  "Asset archived: %s",
		MsgAssetRestoreSuccess:  "Asset restored: %s",
		MsgAssetDeleteSuccess:   "Asset deleted: %s",
		MsgAssetNotFound:         "Asset not found: %s",
		MsgAssetStateConflict:   "Asset state conflict: %s",
		MsgEvalRunStarted:       "Evaluation started: %s",
		MsgEvalCancelSuccess:    "Evaluation cancelled: %s",
		MsgEvalCompareTitle:     "Evaluation Comparison",
		MsgEvalScoreDelta:       "Score delta: %+.2f",
		MsgServeStarting:        "Starting server...",
		MsgServeStarted:         "Server started",
		MsgServeAPIEndpoint:     "API endpoint: %s",
		MsgServeSSEEndpoint:     "SSE endpoint: %s",
		MsgServeOpeningBrowser:  "Opening browser...",
		MsgInitTitle:            "Initialize Prompt Management System",
		MsgInitGitComplete:      "Git repository initialized",
		MsgInitLockAdded:        "Lock file added",
		MsgInitComplete:         "Initialization complete",
		MsgInitServeHint:        "Run 'ep serve' to start the server",
		MsgSyncReconcileDone:    "Sync complete: %d changes",
		MsgSyncAdded:            "Added: %s",
		MsgSyncUpdated:          "Updated: %s",
		MsgSyncDeleted:          "Deleted: %s",
		MsgSyncError:            "Sync error: %s",
		MsgCommonCancel:         "Cancel",
		MsgCommonConfirm:        "Confirm",
		MsgCommonError:          "Error",
		MsgCommonLoading:        "Loading...",
		MsgCommonSuccess:        "Success",
		MsgCommonWarning:        "Warning",
		MsgErrAssetNotFound:     "Asset not found",
		MsgErrInvalidID:         "Invalid ID: %s",
		MsgErrGitNotInitialized: "Git repository not initialized",
		MsgErrStorageNotConfigured: "Storage not configured",
	}
}

// SetLang sets the current language.
func SetLang(lang string) error {
	if global == nil {
		return fmt.Errorf("i18n not initialized")
	}
	if !global.loader.HasLang(lang) {
		return fmt.Errorf("language not supported: %s", lang)
	}
	global.mu.Lock()
	defer global.mu.Unlock()
	global.lang = lang
	return nil
}

// SetLangIfNotEnv sets the language only if EP_LANG is not already set.
// This allows config.yaml to set a default language while still respecting
// the EP_LANG environment variable override.
// Returns error if lang is not supported.
func SetLangIfNotEnv(lang string) error {
	if os.Getenv(EnvLang) != "" {
		return nil // EP_LANG takes precedence
	}
	return SetLang(lang)
}

// GetLang returns the current language.
func GetLang() string {
	if global == nil {
		return DefaultLang
	}
	global.mu.RLock()
	defer global.mu.RUnlock()
	return global.lang
}

// T translates a message key with optional parameters.
//
// Usage:
//   i18n.T("asset_create_success", id)  // "Asset created: 01ABC"
//   i18n.T("common_error")               // "Error"
func T(key string, params ...any) string {
	if global == nil {
		return key
	}
	return global.translate(key, params...)
}

// translate performs the translation.
func (i *I18n) translate(key string, params ...any) string {
	i.mu.RLock()
	lang := i.lang
	i.mu.RUnlock()

	// Try current language first
	if msg, ok := i.loader.GetMessage(lang, key); ok {
		return formatMessage(msg, params)
	}

	// Fall back to English
	if lang != DefaultLang {
		if msg, ok := i.loader.GetMessage(DefaultLang, key); ok {
			return formatMessage(msg, params)
		}
	}

	// Fall back to hardcoded defaults
	if msg, ok := i.fallback[key]; ok {
		return formatMessage(msg, params)
	}

	// Return key as last resort
	return key
}

// formatMessage formats a message with parameters using sequential filling.
func formatMessage(msg string, params []any) string {
	if len(params) == 0 {
		return msg
	}

	// Find and replace format specifiers sequentially
	// Supported: %s, %d, %f, %+.2f, etc.
	result := msg
	paramIdx := 0
	for paramIdx < len(params) {
		// Find the next % followed by a format specifier
		idx := strings.Index(result, "%")
		if idx == -1 {
			break
		}

		// Check if there's a valid format specifier after %
		if idx+1 >= len(result) {
			break
		}

		// Parse format specifier (look for chars like s, d, f, etc.)
		specIdx := idx + 1
		var specLen int
		for specIdx < len(result) && (result[specIdx] == '+' || result[specIdx] == '-' || result[specIdx] == ' ' || result[specIdx] == '0' || result[specIdx] == '.') {
			specIdx++
		}
		if specIdx < len(result) && (result[specIdx] == 's' || result[specIdx] == 'd' || result[specIdx] == 'f' || result[specIdx] == 'v') {
			specLen = specIdx - idx + 1
		} else {
			// Not a format specifier, continue searching
			idx = strings.Index(result[idx+1:], "%")
			if idx == -1 {
				break
			}
			continue
		}

		// Replace the format specifier with the parameter value
		formatted := fmt.Sprintf("%v", params[paramIdx])
		result = result[:idx] + formatted + result[idx+specLen:]
		paramIdx++
	}
	return result
}

// AvailableLangs returns a list of available languages.
func AvailableLangs() []string {
	if global == nil {
		return []string{DefaultLang}
	}
	return global.loader.AvailableLangs()
}
