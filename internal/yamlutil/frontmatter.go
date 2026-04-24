package yamlutil

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/eval-prompt/internal/domain"
	"gopkg.in/yaml.v3"
)

// ParseFrontMatter parses YAML front matter from a .md file content.
// It expects the content to have a YAML front matter block delimited by ---.
// Returns the parsed FrontMatter and the content after the front matter.
func ParseFrontMatter(content string) (*domain.FrontMatter, string, error) {
	// Check for front matter delimiters
	if !strings.HasPrefix(content, "---") {
		return nil, "", fmt.Errorf("no front matter found: expected --- at start")
	}

	// Skip the opening --- (possibly with trailing newline)
	rest := content[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	idx := strings.Index(rest, "---")
	if idx < 0 {
		return nil, "", fmt.Errorf("no closing --- found")
	}

	yamlContent := strings.TrimSpace(rest[:idx])
	markdownContent := strings.TrimSpace(rest[idx+3:])

	var fm domain.FrontMatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, "", fmt.Errorf("failed to parse front matter: %w", err)
	}

	if err := fm.Validate(); err != nil {
		return nil, "", fmt.Errorf("front matter validation failed: %w", err)
	}

	return &fm, markdownContent, nil
}

// SerializeFrontMatter serializes a FrontMatter struct to YAML format.
// It does NOT include the --- delimiters.
func SerializeFrontMatter(fm *domain.FrontMatter) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("failed to serialize front matter: %w", err)
	}
	return string(data), nil
}

// FormatMarkdown formats a complete .md file with front matter and content.
// The front matter is wrapped with --- delimiters.
func FormatMarkdown(fm *domain.FrontMatter, content string) (string, error) {
	yamlContent, err := SerializeFrontMatter(fm)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.WriteString(yamlContent)
	buf.WriteString("---\n")
	buf.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

// ParseFrontMatterFromFile parses front matter from a full .md file string.
// This is a convenience wrapper around ParseFrontMatter.
func ParseFrontMatterFromFile(fileContent string) (*domain.FrontMatter, string, error) {
	return ParseFrontMatter(fileContent)
}