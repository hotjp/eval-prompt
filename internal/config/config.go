package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	goyaml "gopkg.in/yaml.v3"
)

// DefaultConfigPath is the default path to the config file.
const DefaultConfigPath = "config.yaml"

// Config is the root configuration structure.
type Config struct {
	Path         string             // path to config file (not marshaled)
	Server       ServerConfig       `koanf:"server"`
	Database     DatabaseConfig     `koanf:"database"`
	Telemetry    TelemetryConfig    `koanf:"telemetry"`
	Sandbox      SandboxConfig      `koanf:"sandbox"`
	Plugins      PluginsConfig      `koanf:"plugins"`
	PromptAssets PromptAssetsConfig `koanf:"prompt_assets"`
	Taxonomy     TaxonomyConfig     `koanf:"taxonomy"`
}

// TaxonomyConfig holds asset_type and tag taxonomy definitions.
type TaxonomyConfig struct {
	AssetTypes []AssetTypeConfig `koanf:"asset_types"`
	Tags     []TagConfig     `koanf:"tags"`
}

// AssetTypeConfig holds a single asset_type definition.
type AssetTypeConfig struct {
	Name        string `koanf:"name"`
	Description string `koanf:"description"`
	Color       string `koanf:"color"`
	BuiltIn     bool   `koanf:"-"` // not serialized, marks built-in items
}

// TagConfig holds a single tag definition.
type TagConfig struct {
	Name    string `koanf:"name"`
	Color   string `koanf:"color"`
	BuiltIn bool   `koanf:"-"` // not serialized, marks built-in items
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         int           `koanf:"port"`
	Host         string        `koanf:"host"`
	MetricsPort  int           `koanf:"metrics_port"`
	PprofPort    int           `koanf:"pprof_port"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	DSN         string        `koanf:"dsn"`
	MaxOpen     int           `koanf:"max_open"`
	MaxIdle     int           `koanf:"max_idle"`
	MaxLifetime time.Duration `koanf:"max_lifetime"`
}

// TelemetryConfig holds observability settings.
type TelemetryConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Endpoint string `koanf:"endpoint"`
}

// SandboxConfig holds sandbox execution limits.
type SandboxConfig struct {
	AllowedCommands   []string      `koanf:"allowed_commands"`
	ForbiddenPatterns []string      `koanf:"forbidden_patterns"`
	MaxExecutionTime  time.Duration `koanf:"max_execution_time"`
	MaxFileSize       int64         `koanf:"max_file_size"`
	MaxFileCount      int           `koanf:"max_file_count"`
}

// PromptAssetsConfig holds prompt asset repository settings.
type PromptAssetsConfig struct {
	RepoPath      string `koanf:"repo_path"`
	AssetsDir     string `koanf:"assets_dir"`
	EvalsDir      string `koanf:"evals_dir"`
	TracesDir     string `koanf:"traces_dir"`
	EvalThreshold int    `koanf:"eval_threshold"`
	Concurrency   int    `koanf:"concurrency"`
}

// PluginsConfig holds plugin-specific settings.
type PluginsConfig struct {
	LLM    []LLMProviderConfig `koanf:"llm"`
	Search SearchPluginConfig   `koanf:"search"`
}

// LLMProviderConfig holds settings for a single LLM provider.
type LLMProviderConfig struct {
	Name         string `koanf:"name" yaml:"name"`
	Provider     string `koanf:"provider" yaml:"provider"` // openai | claude | ollama
	APIKey       string `koanf:"api_key" yaml:"api_key"`
	Endpoint     string `koanf:"endpoint" yaml:"endpoint"`
	DefaultModel string `koanf:"default_model" yaml:"default_model"`
	Default      bool   `koanf:"default" yaml:"default"` // if true, this is the default provider
	PingPath     string `koanf:"ping_path" yaml:"ping_path"` // lightweight health check path, e.g. "/v1/models"
}

// LLMPluginConfig holds LLM plugin settings (legacy single-provider format).
// Deprecated: Use []LLMProviderConfig instead.
type LLMPluginConfig struct {
	Enabled      bool   `koanf:"enabled"`
	Provider     string `koanf:"provider"` // openai | claude | ollama
	APIKey       string `koanf:"api_key"`
	Endpoint     string `koanf:"endpoint"`
	DefaultModel string `koanf:"default_model"`
}

// SearchPluginConfig holds search plugin settings.
type SearchPluginConfig struct {
	Enabled bool   `koanf:"enabled"`
	Type    string `koanf:"type"` // meilisearch | basic
	URL     string `koanf:"url"`
	APIKey  string `koanf:"api_key"`
}

// envKeyTransform transforms environment variable names to koanf key paths.
// It strips the prefix and replaces underscores with dots.
// For example: APP_SERVER_PORT -> server.port
func envKeyTransform(s string) string {
	// Strip the APP_ prefix
	s = strings.TrimPrefix(s, "APP_")
	// Replace double underscores (__) with dots
	s = strings.ReplaceAll(s, "__", ".")
	// Replace single underscores (_) with dots for nested keys
	// But be careful not to break values that contain underscores
	// We only replace underscores that are likely delimiters
	return strings.ToLower(s)
}

// Load reads configuration from the given file path and applies environment variable overrides.
// Environment variables with the APP_ prefix override YAML values using koanf's dot notation.
func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Use default config path if none provided
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	// Set all defaults first
	for key, value := range getDefaults() {
		k.Set(key, value)
	}

	// Load from YAML file if it exists
	if fileExists(configPath) {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configPath, err)
		}
	}

	// Load environment variables with APP_ prefix, override existing keys
	if err := k.Load(env.Provider("APP_", ".", envKeyTransform), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	cfg.Path = configPath

	return &cfg, nil
}

// MustLoad loads the configuration and panics on error.
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getDefaults returns the default configuration as a flat map.
func getDefaults() map[string]interface{} {
	return map[string]interface{}{
		// Server defaults
		"server.port":          8080,
		"server.host":          "127.0.0.1",
		"server.metrics_port":  9090,
		"server.pprof_port":    6060,
		"server.read_timeout":  30 * time.Second,
		"server.write_timeout": 30 * time.Second,

		// Database defaults
		"database.dsn":          "",
		"database.max_open":     25,
		"database.max_idle":     5,
		"database.max_lifetime": 5 * time.Minute,

		// Telemetry defaults
		"telemetry.enabled":  false,
		"telemetry.endpoint": "localhost:4317",

		// Sandbox defaults
		"sandbox.allowed_commands": []string{
			"npm", "go", "python", "python3", "node",
			"git", "curl", "mkdir", "cp", "mv",
		},
		"sandbox.forbidden_patterns": []string{
			"rm -rf /", "rm -rf /*", "> /etc/", "| sh", "| bash", "curl .* | sh",
		},
		"sandbox.max_execution_time": 60 * time.Second,
		"sandbox.max_file_size":      int64(10 * 1024 * 1024), // 10MB
		"sandbox.max_file_count":     1000,

		// PromptAssets defaults
		"prompt_assets.repo_path":      "./prompt-assets",
		"prompt_assets.assets_dir":     "prompts",
		"prompt_assets.evals_dir":      ".evals",
		"prompt_assets.traces_dir":     ".traces",
		"prompt_assets.eval_threshold": 80,
		"prompt_assets.concurrency":    4,

		"plugins.search.enabled": false,
		"plugins.search.type":    "basic",
		"plugins.search.url":     "",
		"plugins.search.api_key": "",

		// Taxonomy defaults
		"taxonomy.asset_types": []map[string]string{
			{"name": "prompt", "description": "提示词", "color": "blue"},
			{"name": "agent", "description": "Agent 描述文件", "color": "purple"},
			{"name": "skill", "description": "Skill", "color": "green"},
			{"name": "knowledge", "description": "知识库", "color": "orange"},
			{"name": "system", "description": "系统配置", "color": "red"},
			{"name": "workflow", "description": "工作流", "color": "cyan"},
			{"name": "tool", "description": "工具描述", "color": "geekblue"},
		},
		"taxonomy.tags": []map[string]string{
			{"name": "prod", "color": "green"},
			{"name": "draft", "color": "orange"},
			{"name": "llm", "color": "blue"},
			{"name": "rag", "color": "purple"},
			{"name": "agent", "color": "cyan"},
			{"name": "security", "color": "red"},
			{"name": "ops", "color": "geekblue"},
			{"name": "go", "color": "lime"},
			{"name": "review", "color": "gold"},
		},
	}
}

// LoadFromEnv loads configuration purely from environment variables without a config file.
// This is useful for containerized deployments.
func LoadFromEnv() (*Config, error) {
	k := koanf.New(".")

	// Set all defaults first
	for key, value := range getDefaults() {
		k.Set(key, value)
	}

	// Load environment variables with APP_ prefix
	if err := k.Load(env.Provider("APP_", ".", envKeyTransform), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.MetricsPort <= 0 || c.Server.MetricsPort > 65535 {
		return fmt.Errorf("invalid metrics port: %d", c.Server.MetricsPort)
	}
	if c.Server.PprofPort <= 0 || c.Server.PprofPort > 65535 {
		return fmt.Errorf("invalid pprof port: %d", c.Server.PprofPort)
	}
	return nil
}

// Save writes the current config back to the YAML file.
// Note: This will overwrite the config file and may lose comments/formatting.
func (c *Config) Save() error {
	if c.Path == "" {
		return fmt.Errorf("config path is not set, cannot save")
	}

	data, err := goyaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.Path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultTaxonomyPath is the default path to the taxonomy config file.
const DefaultTaxonomyPath = "config/taxonomy.yaml"

// LoadTaxonomy loads taxonomy from the given YAML file path.
// Returns merged taxonomy (built-in + user config) if file exists.
// Built-in items are marked with BuiltIn=true, user items with BuiltIn=false.
func LoadTaxonomy(path string) (*TaxonomyConfig, error) {
	// Start with built-in taxonomy (all items marked BuiltIn=true)
	merged := defaultTaxonomy()

	if path == "" {
		path = DefaultTaxonomyPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No user config, return built-in only
			return merged, nil
		}
		return nil, fmt.Errorf("failed to read taxonomy file: %w", err)
	}

	var userTaxonomy TaxonomyConfig
	if err := goyaml.Unmarshal(data, &userTaxonomy); err != nil {
		return nil, fmt.Errorf("failed to parse taxonomy YAML: %w", err)
	}

	// Merge user config: override built-in by name, append new user items
	merged.AssetTypes = mergeAssetTypes(merged.AssetTypes, userTaxonomy.AssetTypes)
	merged.Tags = mergeTags(merged.Tags, userTaxonomy.Tags)

	return merged, nil
}

// mergeAssetTypes merges user asset_types into built-in ones.
// User items override built-in by name (case-sensitive).
func mergeAssetTypes(builtIn, user []AssetTypeConfig) []AssetTypeConfig {
	result := make([]AssetTypeConfig, 0, len(builtIn)+len(user))
	seen := make(map[string]bool)

	// First add all built-in
	for _, b := range builtIn {
		b.BuiltIn = true
		result = append(result, b)
		seen[b.Name] = true
	}

	// Then add user items (override or append)
	for _, u := range user {
		if overridden, exists := findAssetTypeByName(result, u.Name); exists {
			// Override: keep BuiltIn=true but update description/color
			overridden.Description = u.Description
			overridden.Color = u.Color
		} else {
			// Append new user item
			u.BuiltIn = false
			result = append(result, u)
		}
		seen[u.Name] = true
	}

	return result
}

// mergeTags merges user tags into built-in ones.
func mergeTags(builtIn, user []TagConfig) []TagConfig {
	result := make([]TagConfig, 0, len(builtIn)+len(user))

	// First add all built-in
	for _, b := range builtIn {
		b.BuiltIn = true
		result = append(result, b)
	}

	// Then add user items (override or append)
	for _, u := range user {
		if exists, idx := findTagByName(result, u.Name); exists {
			// Override: keep BuiltIn=true but update color
			result[idx].Color = u.Color
		} else {
			// Append new user item
			u.BuiltIn = false
			result = append(result, u)
		}
	}

	return result
}

func findAssetTypeByName(list []AssetTypeConfig, name string) (*AssetTypeConfig, bool) {
	for i := range list {
		if list[i].Name == name {
			return &list[i], true
		}
	}
	return nil, false
}

func findTagByName(list []TagConfig, name string) (bool, int) {
	for i := range list {
		if list[i].Name == name {
			return true, i
		}
	}
	return false, -1
}

// SaveTaxonomy saves taxonomy to the given YAML file path.
func SaveTaxonomy(path string, taxonomy *TaxonomyConfig) error {
	if path == "" {
		path = DefaultTaxonomyPath
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create taxonomy directory: %w", err)
	}

	data, err := goyaml.Marshal(taxonomy)
	if err != nil {
		return fmt.Errorf("failed to marshal taxonomy: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write taxonomy file: %w", err)
	}

	return nil
}

// defaultTaxonomy returns the default taxonomy configuration.
func defaultTaxonomy() *TaxonomyConfig {
	return &TaxonomyConfig{
		AssetTypes: []AssetTypeConfig{
			{Name: "prompt", Description: "提示词", Color: "blue", BuiltIn: true},
			{Name: "agent", Description: "Agent 描述文件", Color: "purple", BuiltIn: true},
			{Name: "skill", Description: "Skill", Color: "green", BuiltIn: true},
			{Name: "knowledge", Description: "知识库", Color: "orange", BuiltIn: true},
			{Name: "system", Description: "系统配置", Color: "red", BuiltIn: true},
			{Name: "workflow", Description: "工作流", Color: "cyan", BuiltIn: true},
			{Name: "tool", Description: "工具描述", Color: "geekblue", BuiltIn: true},
		},
		Tags: []TagConfig{
			{Name: "prod", Color: "green", BuiltIn: true},
			{Name: "draft", Color: "orange", BuiltIn: true},
			{Name: "llm", Color: "blue", BuiltIn: true},
			{Name: "rag", Color: "purple", BuiltIn: true},
			{Name: "agent", Color: "cyan", BuiltIn: true},
			{Name: "security", Color: "red", BuiltIn: true},
			{Name: "ops", Color: "geekblue", BuiltIn: true},
			{Name: "go", Color: "lime", BuiltIn: true},
			{Name: "review", Color: "gold", BuiltIn: true},
		},
	}
}

// DefaultLLMConfigPath is the default path to the LLM config file.
const DefaultLLMConfigPath = "config/llm.yaml"

// LoadLLMConfig loads LLM provider configs from the given YAML file path.
func LoadLLMConfig(path string) ([]LLMProviderConfig, error) {
	if path == "" {
		path = DefaultLLMConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read LLM config file: %w", err)
	}

	var configs []LLMProviderConfig
	if err := goyaml.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse LLM config YAML: %w", err)
	}

	return configs, nil
}

// SaveLLMConfig saves LLM provider configs to the given YAML file path.
func SaveLLMConfig(path string, configs []LLMProviderConfig) error {
	if path == "" {
		path = DefaultLLMConfigPath
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create LLM config directory: %w", err)
	}

	data, err := goyaml.Marshal(configs)
	if err != nil {
		return fmt.Errorf("failed to marshal LLM config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write LLM config file: %w", err)
	}

	return nil
}

// SaveLLMConfigToMain saves LLM provider configs to the main config.yaml file.
// This updates the plugins.llm section in the main config file.
func SaveLLMConfigToMain(configPath string, configs []LLMProviderConfig) error {
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, create it with just the LLM config
		cfg := map[string]interface{}{
			"plugins": map[string]interface{}{
				"llm": configs,
			},
		}
		out, err := goyaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		return os.WriteFile(configPath, out, 0644)
	}

	// Parse existing config
	var doc map[string]interface{}
	if err := goyaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse existing config: %w", err)
	}

	// Update plugins.llm section
	if doc["plugins"] == nil {
		doc["plugins"] = map[string]interface{}{}
	}
	plugins := doc["plugins"].(map[string]interface{})
	plugins["llm"] = configs

	// Write back
	out, err := goyaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, out, 0644)
}
