package authz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSandboxGuard_ValidatePath_TraversalBoundaryCases(t *testing.T) {
	guard := NewSandboxGuard("/sandbox")

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "path traversal with ../etc/passwd",
			path:    "/sandbox/../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path traversal attempt to escape sandbox",
			path:    "../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "deep path traversal",
			path:    "/sandbox/foo/bar/../../../../../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path outside sandbox /etc/passwd",
			path:    "/etc/passwd",
			wantErr: ErrOutsideSandbox,
		},
		{
			name:    "path completely outside sandbox using symlink equivalent",
			path:    "/usr/share/secrets",
			wantErr: ErrOutsideSandbox,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := guard.ValidatePath(tt.path)
			require.Error(t, err)
			require.Equal(t, tt.wantErr, err)
		})
	}
}

func TestSandboxGuard_ValidatePath(t *testing.T) {
	rootDir := "/sandbox"

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "normal path within sandbox",
			path:    "/sandbox/prompts/test.txt",
			wantErr: nil,
		},
		{
			name:    "relative path within sandbox",
			path:    "prompts/test.txt",
			wantErr: nil,
		},
		{
			name:    "path traversal attempt",
			path:    "/sandbox/../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path traversal with clean",
			path:    "/sandbox/prompts/../../secrets",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path outside sandbox",
			path:    "/etc/passwd",
			wantErr: ErrOutsideSandbox,
		},
		{
			name:    "path completely outside sandbox",
			path:    "/home/user/file",
			wantErr: ErrOutsideSandbox,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: nil, // filepath.Clean("") returns "."
		},
		{
			name:    "current directory reference",
			path:    "/sandbox/./prompts/./test.txt",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard := NewSandboxGuard(rootDir)
			err := guard.ValidatePath(tt.path)
			if err != tt.wantErr {
				t.Errorf("ValidatePath(%q) = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestSandboxGuard_IsPathSafe(t *testing.T) {
	rootDir := "/sandbox"

	tests := []struct {
		name   string
		path   string
		isSafe bool
	}{
		{
			name:   "safe path",
			path:   "/sandbox/prompts/test.txt",
			isSafe: true,
		},
		{
			name:   "unsafe path traversal",
			path:   "/sandbox/../etc",
			isSafe: false,
		},
		{
			name:   "unsafe path outside sandbox",
			path:   "/etc/passwd",
			isSafe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard := NewSandboxGuard(rootDir)
			if got := guard.IsPathSafe(tt.path); got != tt.isSafe {
				t.Errorf("IsPathSafe(%q) = %v, want %v", tt.path, got, tt.isSafe)
			}
		})
	}
}

func TestSandboxGuard_ValidatePathWithContext(t *testing.T) {
	guard := NewSandboxGuard("/sandbox")
	ctx := context.Background()

	err := guard.ValidatePathWithContext(ctx, "/sandbox/prompts/test.txt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSandboxGuard_RootDir(t *testing.T) {
	guard := NewSandboxGuard("/sandbox")
	if got := guard.RootDir(); got != "/sandbox" {
		t.Errorf("RootDir() = %v, want /sandbox", got)
	}
}

func TestSandboxGuard_CleanRemovesTraversal(t *testing.T) {
	rootDir := "/sandbox"

	tests := []struct {
		name    string
		input   string
		cleaned string
	}{
		{
			name:    "removes single dot",
			input:   "/sandbox/./prompts",
			cleaned: "/sandbox/prompts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard := NewSandboxGuard(rootDir)
			err := guard.ValidatePath(tt.input)
			// Single dot paths are allowed
			if err != nil {
				t.Errorf("ValidatePath(%q) returned unexpected error: %v", tt.input, err)
			}
		})
	}
}
