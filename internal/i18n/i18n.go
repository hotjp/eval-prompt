package i18n

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flosch/pongo2/v6"
)

//go:embed locales/*.json
var fs embed.FS

const (
	EnvLang     = "EP_LANG"
	DefaultLang = "en-US"
)

var (
	locales  map[string]map[string]string
	current  = DefaultLang
	initOnce sync.Once
)

// Init loads locales from embedded files (i18n/locales/).
// Language priority: EP_LANG > system LANG > DefaultLang
func Init() error {
	var err error
	initOnce.Do(func() {
		locales = make(map[string]map[string]string)

		// Try embedded files first
		entries, e := fs.ReadDir("locales")
		if e == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}
				lang := strings.TrimSuffix(entry.Name(), ".json")
				data, e := fs.ReadFile("locales/" + entry.Name())
				if e != nil {
					continue
				}
				var msgs map[string]string
				if e := json.Unmarshal(data, &msgs); e == nil {
					locales[lang] = msgs
				}
			}
		}

		// Fallback: try loading from disk (for development)
		if len(locales) == 0 {
			dir := findLocalesDir()
			if dir != "" {
				loadLocalesFromDir(dir)
			}
		}

		// Determine initial language: EP_LANG > LANG > DefaultLang
		if lang := os.Getenv(EnvLang); lang != "" {
			if _, ok := locales[lang]; ok {
				current = lang
			}
		} else if sysLang := parseSystemLang(os.Getenv("LANG")); sysLang != "" {
			if _, ok := locales[sysLang]; ok {
				current = sysLang
			}
		}

		if _, ok := locales[current]; !ok {
			// Fall back to en-US if current lang not found
			for lang := range locales {
				current = lang
				break
			}
		}
	})
	return err
}

func loadLocalesFromDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		lang := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var msgs map[string]string
		if err := json.Unmarshal(data, &msgs); err != nil {
			continue
		}
		locales[lang] = msgs
	}
}

func findLocalesDir() string {
	// Relative to module root
	modRoot, err := findModuleRoot()
	if err == nil {
		dir := filepath.Join(modRoot, "i18n", "locales")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	// Relative to current working directory
	cwd, err := os.Getwd()
	if err == nil {
		dir := filepath.Join(cwd, "i18n", "locales")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return ""
}

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
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

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

// SetLang sets the current language.
func SetLang(lang string) error {
	if _, ok := locales[lang]; !ok {
		return os.ErrNotExist
	}
	current = lang
	return nil
}

// SetLangIfNotEnv sets language only if EP_LANG is not set.
func SetLangIfNotEnv(lang string) error {
	if os.Getenv(EnvLang) != "" {
		return nil
	}
	return SetLang(lang)
}

// GetLang returns the current language.
func GetLang() string {
	return current
}

// T returns the localized string for the given key,
// using pongo2 template syntax for parameter substitution.
//
// Usage:
//
//	i18n.T("asset_create_success", pongo2.Context{"id": id})
//	i18n.T("sync_reconcile_done", pongo2.Context{"count": 5})
//
// If the key is not found, returns the key itself.
func T(key string, args pongo2.Context) string {
	msg, ok := locales[current][key]
	if !ok {
		// Try English fallback
		if current != DefaultLang {
			if msg, ok := locales[DefaultLang][key]; ok {
				return render(msg, args)
			}
		}
		return key
	}
	return render(msg, args)
}

// render applies pongo2 template rendering if args are provided.
func render(msg string, args pongo2.Context) string {
	if len(args) == 0 {
		return msg
	}
	tpl, err := pongo2.FromString(msg)
	if err != nil {
		return msg
	}
	result, err := tpl.Execute(args)
	if err != nil {
		return msg
	}
	return result
}

// AvailableLangs returns the list of loaded languages.
func AvailableLangs() []string {
	langs := make([]string, 0, len(locales))
	for lang := range locales {
		langs = append(langs, lang)
	}
	return langs
}
