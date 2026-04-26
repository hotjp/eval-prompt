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
// This parses content prompt front matter (with eval_history, labels fields).
func ParseFrontMatter(content string) (*domain.FrontMatter, string, error) {
	fm := &domain.FrontMatter{}
	markdownContent, err := parseFrontMatterInto(fm, content)
	if err != nil {
		return nil, "", err
	}
	return fm, markdownContent, nil
}

// ParseEvalPromptFrontMatter parses YAML front matter from an eval prompt .md file.
// It expects the content to have a YAML front matter block delimited by ---.
// Returns the parsed EvalPromptFrontMatter and the content after the front matter.
func ParseEvalPromptFrontMatter(content string) (*domain.EvalPromptFrontMatter, string, error) {
	fm := &domain.EvalPromptFrontMatter{}
	markdownContent, err := parseFrontMatterInto(fm, content)
	if err != nil {
		return nil, "", err
	}
	return fm, markdownContent, nil
}

// parseFrontMatterInto parses YAML front matter into the given struct.
func parseFrontMatterInto(target interface{}, content string) (string, error) {
	// Check for front matter delimiters
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("no front matter found: expected --- at start")
	}

	// Skip the opening --- (possibly with trailing newline)
	rest := content[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	idx := strings.Index(rest, "---")
	if idx < 0 {
		return "", fmt.Errorf("no closing --- found")
	}

	yamlContent := strings.TrimSpace(rest[:idx])
	markdownContent := strings.TrimSpace(rest[idx+3:])

	if err := yaml.Unmarshal([]byte(yamlContent), target); err != nil {
		return "", fmt.Errorf("failed to parse front matter: %w", err)
	}

	// Validate based on target type
	switch fm := target.(type) {
	case *domain.FrontMatter:
		if err := fm.Validate(); err != nil {
			return "", fmt.Errorf("front matter validation failed: %w", err)
		}
	case *domain.EvalPromptFrontMatter:
		if err := fm.Validate(); err != nil {
			return "", fmt.Errorf("eval prompt front matter validation failed: %w", err)
		}
	}

	return markdownContent, nil
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

// SerializeEvalPromptFrontMatter serializes an EvalPromptFrontMatter struct to YAML format.
// It does NOT include the --- delimiters.
func SerializeEvalPromptFrontMatter(fm *domain.EvalPromptFrontMatter) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("failed to serialize eval prompt front matter: %w", err)
	}
	return string(data), nil
}

// NormalizeBody normalizes markdown body content for consistent hashing.
// - Converts CRLF to LF
// - Removes trailing whitespace on each line
// - Ensures single trailing newline
// - Trims leading/trailing blank lines
func NormalizeBody(body string) string {
	// Convert CRLF to LF
	body = strings.ReplaceAll(body, "\r\n", "\n")
	// Remove \r not followed by \n (old Mac-style)
	body = strings.ReplaceAll(body, "\r", "\n")

	lines := strings.Split(body, "\n")
	var cleaned []string
	for _, line := range lines {
		// Remove trailing whitespace on each line
		cleaned = append(cleaned, strings.TrimRight(line, " \t"))
	}

	// Remove leading blank lines
	start := 0
	for start < len(cleaned) && cleaned[start] == "" {
		start++
	}

	// Remove trailing blank lines
	end := len(cleaned)
	for end > start && cleaned[end-1] == "" {
		end--
	}

	body = strings.Join(cleaned[start:end], "\n")

	// Ensure single trailing newline
	return strings.TrimRight(body, "\n") + "\n"
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

// FormatEvalPromptMarkdown formats a complete eval prompt .md file with front matter and content.
// The front matter is wrapped with --- delimiters.
func FormatEvalPromptMarkdown(fm *domain.EvalPromptFrontMatter, content string) (string, error) {
	yamlContent, err := SerializeEvalPromptFrontMatter(fm)
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
