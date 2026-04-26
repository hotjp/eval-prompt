package i18n

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader handles loading language packs from YAML files.
type Loader struct {
	mu         sync.RWMutex
	localesDir string
	messages   map[string]map[string]string // lang -> key -> message
}

// NewLoader creates a new Loader with the specified locales directory.
func NewLoader(localesDir string) *Loader {
	return &Loader{
		localesDir: localesDir,
		messages:   make(map[string]map[string]string),
	}
}

// Load loads all language packs from the locales directory.
func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := os.ReadDir(l.localesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		lang := filepath.Base(entry.Name())
		lang = lang[:len(lang)-len(filepath.Ext(entry.Name()))] // strip .yaml

		if err := l.loadFile(filepath.Join(l.localesDir, entry.Name()), lang); err != nil {
			return err
		}
	}

	return nil
}

// loadFile loads a single language file.
func (l *Loader) loadFile(path, lang string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var messages map[string]string
	if err := yaml.Unmarshal(data, &messages); err != nil {
		return err
	}

	l.messages[lang] = messages
	return nil
}

// GetMessage returns the message for the given language and key.
func (l *Loader) GetMessage(lang, key string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if langMessages, ok := l.messages[lang]; ok {
		if msg, ok := langMessages[key]; ok {
			return msg, true
		}
	}
	return "", false
}

// HasLang checks if a language is loaded.
func (l *Loader) HasLang(lang string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.messages[lang]
	return ok
}

// AvailableLangs returns a list of available languages.
func (l *Loader) AvailableLangs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	langs := make([]string, 0, len(l.messages))
	for lang := range l.messages {
		langs = append(langs, lang)
	}
	return langs
}
