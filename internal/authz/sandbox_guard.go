// Package authz implements L3-Authz layer: permission checks (RBAC/OpenFGA),
// rate limiting, and identity verification.
package authz

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
)

// ErrPathTraversal is returned when a path traversal attempt is detected.
var ErrPathTraversal = errors.New("path traversal detected")

// ErrOutsideSandbox is returned when a path is outside the sandbox directory.
var ErrOutsideSandbox = errors.New("path outside sandbox directory")

// SandboxGuard validates that file paths are within a sandbox directory and not vulnerable to path traversal.
type SandboxGuard struct {
	rootDir string
}

// NewSandboxGuard creates a new SandboxGuard with the specified root sandbox directory.
func NewSandboxGuard(rootDir string) *SandboxGuard {
	return &SandboxGuard{
		rootDir: filepath.Clean(rootDir),
	}
}

// ValidatePath checks if a path is safe within the sandbox.
// Returns nil if the path is safe, ErrPathTraversal or ErrOutsideSandbox otherwise.
func (g *SandboxGuard) ValidatePath(path string) error {
	// Check for ".." in the original path before cleaning - this catches traversal attempts
	if strings.Contains(path, "..") {
		return ErrPathTraversal
	}

	// Clean the path to resolve any "." components and normalize
	cleanPath := filepath.Clean(path)

	// Make the path absolute if it isn't already
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(g.rootDir, cleanPath)
		cleanPath = filepath.Clean(cleanPath)
	}

	// Check if the path is within the sandbox root
	if !strings.HasPrefix(cleanPath, g.rootDir) {
		return ErrOutsideSandbox
	}

	return nil
}

// ValidatePathWithContext is like ValidatePath but accepts a context (for future extensibility).
func (g *SandboxGuard) ValidatePathWithContext(ctx context.Context, path string) error {
	return g.ValidatePath(path)
}

// RootDir returns the configured sandbox root directory.
func (g *SandboxGuard) RootDir() string {
	return g.rootDir
}

// IsPathSafe returns true if the path is safe within the sandbox.
func (g *SandboxGuard) IsPathSafe(path string) bool {
	return g.ValidatePath(path) == nil
}
