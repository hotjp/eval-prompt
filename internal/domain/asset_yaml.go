package domain

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AssetYAML represents the structure of an asset.yaml file.
// This is the registry format for folder-based assets.
type AssetYAML struct {
	// Required fields
	AssetType string `yaml:"asset_type"` // prompt | skill | agent | mcp | workflow | knowledge
	Name      string `yaml:"name"`
	Main      string `yaml:"main"` // entry file (repo relative path)

	// Basic info
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Category    string   `yaml:"category,omitempty"` // default "content"
	State       string   `yaml:"state,omitempty"`    // draft | published | archived | deleted | unavailable

	// Entry
	MainFunction string `yaml:"main_function,omitempty"` // entry function name for skill/agent

	// File associations
	Files    []FileEntry `yaml:"files,omitempty"`    // all files in asset
	External []FileEntry `yaml:"external,omitempty"` // related files outside main file

	// Version and upstream
	Version  string    `yaml:"version,omitempty"`
	Upstream *Upstream `yaml:"upstream,omitempty"`

	// Metadata (system-managed, user should not edit)
	Metadata *AssetMetadata `yaml:"metadata,omitempty"`
}

// FileEntry represents a file in the asset's files list.
type FileEntry struct {
	Path string `yaml:"path"` // repo relative path
	Role string `yaml:"role"` // main | script | config | doc | data | lib | template
}

// Upstream represents upstream repository information.
type Upstream struct {
	URL       string `yaml:"url,omitempty"`       // upstream Git repo URL
	Branch    string `yaml:"branch,omitempty"`    // upstream branch
	LastSync  string `yaml:"last_sync,omitempty"` // last sync time
}

// AssetMetadata contains system-managed metadata.
type AssetMetadata struct {
	CreatedAt  time.Time `yaml:"created_at,omitempty"`
	CreatedBy  string    `yaml:"created_by,omitempty"`
	UpdatedAt  time.Time `yaml:"updated_at,omitempty"`
	UpdatedBy  string    `yaml:"updated_by,omitempty"`
}

// Validate validates the AssetYAML structure.
func (a *AssetYAML) Validate() error {
	// Validate required fields
	if a.AssetType == "" {
		return NewDomainError(ErrAssetTypeRequired, "asset_type is required")
	}
	if !isValidAssetType(a.AssetType) {
		return NewDomainError(ErrAssetTypeInvalid, "asset_type must be one of: prompt, skill, agent, mcp, workflow, knowledge")
	}
	if a.Name == "" {
		return NewDomainError(ErrAssetNameEmpty, "name is required")
	}
	if a.Main == "" {
		return NewDomainError(ErrAssetMainRequired, "main is required")
	}

	// Validate state if provided
	if a.State != "" && !isValidState(a.State) {
		return NewDomainError(ErrAssetStateInvalid, "state must be one of: draft, published, archived, deleted, unavailable")
	}

	// Validate category if provided
	if a.Category != "" && !isValidCategory(a.Category) {
		return NewDomainError(ErrAssetCategoryInvalid, "category must be one of: content, eval, metric")
	}

	return nil
}

// isValidAssetType checks if the asset type is valid.
func isValidAssetType(t string) bool {
	validTypes := []string{"prompt", "skill", "agent", "mcp", "workflow", "knowledge"}
	for _, vt := range validTypes {
		if t == vt {
			return true
		}
	}
	return false
}

// isValidState checks if the state is valid.
func isValidState(s string) bool {
	validStates := []string{"draft", "published", "archived", "deleted", "unavailable"}
	for _, vs := range validStates {
		if s == vs {
			return true
		}
	}
	return false
}

// isValidCategory checks if the category is valid.
func isValidCategory(c string) bool {
	validCategories := []string{"content", "eval", "metric"}
	for _, vc := range validCategories {
		if c == vc {
			return true
		}
	}
	return false
}

// DefaultState returns the default state for a new asset.
func (a *AssetYAML) DefaultState() string {
	if a.State == "" {
		return "draft"
	}
	return a.State
}

// DefaultCategory returns the default category for a new asset.
func (a *AssetYAML) DefaultCategory() string {
	if a.Category == "" {
		return "content"
	}
	return a.Category
}

// ResolveMain resolves the main path relative to repo root.
// If main starts with ~, it expands to user home directory.
// If main starts with /, it's an absolute path (external asset).
// Otherwise, it's relative to repo root.
func (a *AssetYAML) ResolveMain(repoPath string) (resolvedPath string, isExternal bool, err error) {
	main := strings.TrimSpace(a.Main)
	if main == "" {
		return "", false, fmt.Errorf("main path is empty")
	}

	// Handle ~ expansion
	if strings.HasPrefix(main, "~/") || main == "~" {
		homeDir, err := getHomeDir()
		if err != nil {
			return "", false, fmt.Errorf("expand home directory: %w", err)
		}
		if main == "~" {
			return homeDir, true, nil
		}
		return strings.Replace(main, "~", homeDir, 1), true, nil
	}

	// Absolute path = external asset
	if strings.HasPrefix(main, "/") {
		return main, true, nil
	}

	// Repo-relative path
	resolvedPath = main
	if repoPath != "" {
		resolvedPath = strings.TrimPrefix(main, repoPath)
		// Remove leading slash if present
		resolvedPath = strings.TrimPrefix(resolvedPath, "/")
	}
	return resolvedPath, false, nil
}

// GetMainFileRole returns the role of the main file.
// Returns "main" by default if not specified in files list.
func (a *AssetYAML) GetMainFileRole() string {
	for _, f := range a.Files {
		if f.Path == a.Main {
			return f.Role
		}
	}
	return "main"
}

// HasFile checks if a file path is in the files list.
func (a *AssetYAML) HasFile(path string) bool {
	for _, f := range a.Files {
		if f.Path == path {
			return true
		}
	}
	return false
}

// HasExternal checks if a path is in the external list.
func (a *AssetYAML) HasExternal(path string) bool {
	for _, e := range a.External {
		if e.Path == path {
			return true
		}
	}
	return false
}

// GetFileRole returns the role of a file, or empty string if not found.
func (a *AssetYAML) GetFileRole(path string) string {
	for _, f := range a.Files {
		if f.Path == path {
			return f.Role
		}
	}
	for _, e := range a.External {
		if e.Path == path {
			return e.Role
		}
	}
	return ""
}

// NewAssetYAML creates a new AssetYAML with default values.
func NewAssetYAML(assetType, name, main string) *AssetYAML {
	now := time.Now()
	return &AssetYAML{
		AssetType: assetType,
		Name:      name,
		Main:      main,
		State:     "draft",
		Category:  "content",
		Metadata: &AssetMetadata{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// ToMap converts AssetYAML to a map for YAML marshaling.
func (a *AssetYAML) ToMap() (map[string]interface{}, error) {
	if a == nil {
		return nil, nil
	}
	data, err := yaml.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("marshal asset yaml: %w", err)
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal asset yaml to map: %w", err)
	}
	return result, nil
}

// ParseAssetYAML parses an asset.yaml file content into an AssetYAML struct.
func ParseAssetYAML(content string) (*AssetYAML, error) {
	if content == "" {
		return nil, fmt.Errorf("empty content")
	}

	ay := &AssetYAML{}
	if err := yaml.Unmarshal([]byte(content), ay); err != nil {
		return nil, fmt.Errorf("parse asset yaml: %w", err)
	}

	// Set defaults
	if ay.State == "" {
		ay.State = "draft"
	}
	if ay.Category == "" {
		ay.Category = "content"
	}

	// Validate
	if err := ay.Validate(); err != nil {
		return nil, fmt.Errorf("validate asset yaml: %w", err)
	}

	return ay, nil
}

// SerializeAssetYAML serializes an AssetYAML struct to YAML format.
func SerializeAssetYAML(ay *AssetYAML) (string, error) {
	if ay == nil {
		return "", fmt.Errorf("nil asset yaml")
	}

	data, err := yaml.Marshal(ay)
	if err != nil {
		return "", fmt.Errorf("marshal asset yaml: %w", err)
	}
	return string(data), nil
}

// SerializeAssetYAMLCompact serializes an AssetYAML to YAML without empty optional fields.
func SerializeAssetYAMLCompact(ay *AssetYAML) (string, error) {
	if ay == nil {
		return "", fmt.Errorf("nil asset yaml")
	}

	// Use yaml.Marshal with strict formatting
	data, err := yaml.Marshal(ay)
	if err != nil {
		return "", fmt.Errorf("marshal asset yaml: %w", err)
	}
	return string(data), nil
}

// AssetYAMLFromFrontMatter converts a FrontMatter (old format) to AssetYAML (new format).
// This is used for migration from .md files with frontmatter to folder-based structure.
func AssetYAMLFromFrontMatter(fm *FrontMatter, mainPath string) *AssetYAML {
	ay := &AssetYAML{
		AssetType:   fm.AssetType,
		Name:        fm.Name,
		Main:        mainPath,
		Description: fm.Description,
		Tags:        fm.Tags,
		Category:    fm.Category,
		State:       fm.State,
		Version:     fm.Version,
		Metadata: &AssetMetadata{
			UpdatedAt: fm.UpdatedAt,
		},
	}
	if ay.State == "" {
		ay.State = "draft"
	}
	if ay.Category == "" {
		ay.Category = "content"
	}
	return ay
}

// getHomeDir returns the user's home directory.
func getHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home directory: %w", err)
	}
	return home, nil
}
