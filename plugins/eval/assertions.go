package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eval-prompt/internal/service"
)

// AssertionLibrary provides built-in assertions for deterministic evaluation.
type AssertionLibrary struct {
	checks map[string]AssertionChecker
}

// AssertionChecker checks a specific assertion type against trace events.
type AssertionChecker interface {
	Check(trace []service.TraceEvent, check service.DeterministicCheck) error
}

// NewAssertionLibrary creates a new assertion library with all built-in checks.
func NewAssertionLibrary() *AssertionLibrary {
	lib := &AssertionLibrary{
		checks: make(map[string]AssertionChecker),
	}

	// Register built-in assertions
	lib.checks["command_executed"] = &CommandExecutedChecker{}
	lib.checks["file_exists"] = &FileExistsChecker{}
	lib.checks["json_valid"] = &JSONValidChecker{}
	lib.checks["content_contains"] = &ContentContainsChecker{}
	lib.checks["json_path"] = &JSONPathChecker{}

	return lib
}

// Get returns an assertion checker by type name.
func (lib *AssertionLibrary) Get(typeName string) AssertionChecker {
	return lib.checks[typeName]
}

// CommandExecutedChecker verifies that a specific command was executed.
type CommandExecutedChecker struct{}

func (c *CommandExecutedChecker) Check(trace []service.TraceEvent, check service.DeterministicCheck) error {
	for _, event := range trace {
		if event.Type == "command_executed" {
			if cmd, ok := event.Data["command"].(string); ok {
				if strings.Contains(cmd, check.Expected) {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("command not found: %s", check.Expected)
}

// FileExistsChecker verifies that a file was created or exists.
type FileExistsChecker struct{}

func (c *FileExistsChecker) Check(trace []service.TraceEvent, check service.DeterministicCheck) error {
	for _, event := range trace {
		if event.Type == "file_created" || event.Type == "file_exists" {
			if path, ok := event.Data["path"].(string); ok {
				if matchesPath(path, check.Path) {
					return nil
				}
			}
		}
	}
	// Also check against actual filesystem
	if check.Path != "" {
		if _, err := os.Stat(check.Path); err == nil {
			return nil
		}
	}
	return fmt.Errorf("file not found: %s", check.Path)
}

// JSONValidChecker verifies that output is valid JSON.
type JSONValidChecker struct{}

func (c *JSONValidChecker) Check(trace []service.TraceEvent, check service.DeterministicCheck) error {
	for _, event := range trace {
		if event.Type == "llm_output" || event.Type == "output" {
			if content, ok := event.Data["content"].(string); ok {
				if err := json.Unmarshal([]byte(content), &map[string]any{}); err == nil {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("no valid JSON found in trace")
}

// ContentContainsChecker verifies that output contains expected content.
type ContentContainsChecker struct{}

func (c *ContentContainsChecker) Check(trace []service.TraceEvent, check service.DeterministicCheck) error {
	for _, event := range trace {
		if event.Type == "llm_output" || event.Type == "output" || event.Type == "content" {
			if content, ok := event.Data["content"].(string); ok {
				if strings.Contains(content, check.Expected) {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("content not found: %s", check.Expected)
}

// JSONPathChecker verifies a value at a specific JSON path.
type JSONPathChecker struct{}

func (c *JSONPathChecker) Check(trace []service.TraceEvent, check service.DeterministicCheck) error {
	for _, event := range trace {
		if event.Type == "llm_output" || event.Type == "output" {
			if content, ok := event.Data["content"].(string); ok {
				var jsonData any
				if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
					continue
				}

				value, err := getJSONPath(jsonData, check.JSONPath)
				if err == nil && fmt.Sprintf("%v", value) == check.Expected {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("json path %s with expected %s not found", check.JSONPath, check.Expected)
}

// matchesPath checks if a path matches the expected pattern (simple wildcard support).
func matchesPath(path, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return path == pattern
}

// getJSONPath extracts a value from JSON using a simple dot-notation path.
func getJSONPath(data any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		part = strings.TrimPrefix(part, "[")

		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			idxStr := strings.TrimSuffix(strings.TrimPrefix(part, "["), "]")
			idx := 0
			fmt.Sscanf(idxStr, "%d", &idx)

			if arr, ok := current.([]any); ok && idx < len(arr) {
				current = arr[idx]
			} else {
				return nil, fmt.Errorf("index out of bounds: %s", part)
			}
		} else if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return nil, fmt.Errorf("path not found: %s", part)
		}
	}

	return current, nil
}

// RunShellCommand runs a shell command for sandbox execution.
func RunShellCommand(cmd string, args ...string) (string, error) {
	execCmd := exec.Command(cmd, args...)
	output, err := execCmd.Output()
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}
	return string(output), nil
}

// ValidateFilePath validates that a path is within allowed directories.
func ValidateFilePath(path string, allowedDirs []string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, dir := range allowedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, absDir) {
			return true
		}
	}
	return false
}
