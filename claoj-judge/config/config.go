// Package config handles judge configuration loading and management.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all judge configuration.
type Config struct {
	// Server connection
	ServerHost string `yaml:"server_host"`
	ServerPort int    `yaml:"server_port"`
	Secure     bool   `yaml:"secure"`

	// Authentication
	JudgeName string `yaml:"judge_name"`
	JudgeKey  string `yaml:"judge_key"`

	// API server
	APIHost string `yaml:"api_host"`
	APIPort int    `yaml:"api_port"`

	// Logging
	LogFile string `yaml:"log_file"`

	// Problem storage
	ProblemGlobs []string `yaml:"problem_globs"`

	// Runtime configuration
	Runtime map[string]interface{} `yaml:"runtime"`

	// Features
	NoWatchdog   bool `yaml:"no_watchdog"`
	SkipSelfTest bool `yaml:"skip_self_test"`

	// Resource limits
	TempDir          string `yaml:"tempdir"`
	MaxSubmissions   int    `yaml:"max_submissions"`
	MaxProcessCount  int    `yaml:"max_processes"`
	MemoryLimit      int64  `yaml:"memory_limit"`
	TimeLimit        int    `yaml:"time_limit"`
	CompilerTimeLimit int   `yaml:"compiler_time_limit"`
}

// DefaultConfig returns configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		ServerPort:        9999,
		APIHost:           "0.0.0.0",
		APIPort:           9998,
		TempDir:           os.TempDir(),
		ProblemGlobs:      []string{"/problems/*/"},
		MaxSubmissions:    10,
		MaxProcessCount:   32,
		MemoryLimit:       524288, // 512MB
		TimeLimit:         10,
		CompilerTimeLimit: 30,
		Runtime:           make(map[string]interface{}),
	}
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Support environment variable overrides
	if env := os.Getenv("CLAOJ_SERVER_HOST"); env != "" {
		cfg.ServerHost = env
	}
	if env := os.Getenv("CLAOJ_SERVER_PORT"); env != "" {
		// Parse port from env
	}
	if env := os.Getenv("CLAOJ_JUDGE_NAME"); env != "" {
		cfg.JudgeName = env
	}
	if env := os.Getenv("CLAOJ_JUDGE_KEY"); env != "" {
		cfg.JudgeKey = env
	}

	return cfg, nil
}

// SaveConfig saves configuration to a YAML file.
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetRuntimeConfig returns runtime-specific configuration.
func (c *Config) GetRuntimeConfig(language string) map[string]interface{} {
	if runtime, ok := c.Runtime[language].(map[string]interface{}); ok {
		return runtime
	}
	return make(map[string]interface{})
}
