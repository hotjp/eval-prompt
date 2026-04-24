package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// DefaultConfigPath is the default path to the config file.
const DefaultConfigPath = "config.yaml"

// Config is the root configuration structure.
type Config struct {
	Server       ServerConfig       `koanf:"server"`
	Database     DatabaseConfig     `koanf:"database"`
	Telemetry    TelemetryConfig    `koanf:"telemetry"`
	Sandbox      SandboxConfig      `koanf:"sandbox"`
	Plugins      PluginsConfig      `koanf:"plugins"`
	PromptAssets PromptAssetsConfig `koanf:"prompt_assets"`
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
}

// PluginsConfig holds plugin-specific settings.
type PluginsConfig struct {
	LLM    LLMPluginConfig    `koanf:"llm"`
	Search SearchPluginConfig `koanf:"search"`
}

// LLMPluginConfig holds LLM plugin settings.
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

	// Set all defaults first
	for key, value := range getDefaults() {
		k.Set(key, value)
	}

	// Load from YAML file if it exists
	if configPath != "" && fileExists(configPath) {
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
		"prompt_assets.repo_path":      "",
		"prompt_assets.assets_dir":     "prompts",
		"prompt_assets.evals_dir":      ".evals",
		"prompt_assets.traces_dir":     ".traces",
		"prompt_assets.eval_threshold": 80,

		// Plugin defaults
		"plugins.llm.enabled":       false,
		"plugins.llm.provider":      "openai",
		"plugins.llm.api_key":       "",
		"plugins.llm.endpoint":      "",
		"plugins.llm.default_model": "gpt-4o",

		"plugins.search.enabled": false,
		"plugins.search.type":    "basic",
		"plugins.search.url":     "",
		"plugins.search.api_key": "",
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
