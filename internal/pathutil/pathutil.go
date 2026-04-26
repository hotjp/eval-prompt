// Package pathutil provides path validation utilities for secure file operations.
package pathutil

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrPathTraversal is returned when a path traversal attempt is detected.
var ErrPathTraversal = errors.New("path traversal not allowed")

// ErrEmptyPath is returned when the path is empty.
var ErrEmptyPath = errors.New("path cannot be empty")

// ValidateInDir validates that the given path is safe to use within baseDir.
// It prevents path traversal attacks by checking for ".." components.
// Returns nil if the path is safe, ErrPathTraversal or ErrEmptyPath otherwise.
//
// Usage:
//   if err := ValidateInDir(path, baseDir); err != nil {
//       return err
//   }
func ValidateInDir(path, baseDir string) error {
	if path == "" {
		return ErrEmptyPath
	}

	// Check for ".." in the path before cleaning - catches traversal attempts
	if strings.Contains(path, "..") {
		return ErrPathTraversal
	}

	// Clean the path to resolve any "." components and normalize
	cleanPath := filepath.Clean(path)

	// Make absolute if relative
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(baseDir, cleanPath)
		cleanPath = filepath.Clean(cleanPath)
	}

	// Verify the resolved path is within baseDir
	cleanBaseDir := filepath.Clean(baseDir)
	if !strings.HasPrefix(cleanPath, cleanBaseDir) {
		return ErrPathTraversal
	}

	return nil
}

// ValidateID validates that an ID doesn't contain path traversal characters.
// IDs should be simple alphanumeric strings (ULID, etc).
func ValidateID(id string) error {
	if id == "" {
		return ErrEmptyPath
	}
	if strings.Contains(id, "..") {
		return ErrPathTraversal
	}
	if strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return ErrPathTraversal
	}
	return nil
}
