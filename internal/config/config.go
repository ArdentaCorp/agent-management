package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// AIToolConfig defines an AI tool type and its possible skill directory paths.
type AIToolConfig struct {
	Type      string   `json:"type"`
	SkillDirs []string `json:"skillDirs"`
}

// Config represents the global skm configuration file.
type Config struct {
	System   string         `json:"system"`
	Registry string         `json:"registry,omitempty"`
	AITools  []AIToolConfig `json:"aiTools,omitempty"`
}

// Manager handles configuration paths and operations.
type Manager struct {
	homeDir    string
	repoDir    string
	configFile string
}

// NewManager creates a new config manager and ensures directories exist.
func NewManager() *Manager {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	homeDir := filepath.Join(home, ".agent-management")
	repoDir := filepath.Join(homeDir, "repo")
	configFile := filepath.Join(homeDir, "config.json")

	// Ensure directories exist
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot create directory %s: %v\n", homeDir, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot create directory %s: %v\n", repoDir, err)
		os.Exit(1)
	}

	// Create config file if it doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		cfg := Config{System: runtime.GOOS}
		data, _ := json.MarshalIndent(cfg, "", "  ")
		os.WriteFile(configFile, data, 0644)
	}

	return &Manager{
		homeDir:    homeDir,
		repoDir:    repoDir,
		configFile: configFile,
	}
}

// GetHomeDir returns the skm home directory path.
func (m *Manager) GetHomeDir() string {
	return m.homeDir
}

// GetRepoDir returns the global skill repository directory path.
func (m *Manager) GetRepoDir() string {
	return m.repoDir
}

// GetSafeName converts a skill ID to a filesystem-safe directory name.
// e.g. "github:user/repo/path" -> "github__user__repo__path"
func (m *Manager) GetSafeName(id string) string {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) < 2 {
		return strings.ReplaceAll(id, "/", "__")
	}
	safeRest := strings.ReplaceAll(parts[1], "/", "__")
	return parts[0] + "__" + safeRest
}

// ParseSafeName converts a filesystem-safe directory name back to a skill ID.
// e.g. "github__user__repo__path" -> "github:user/repo/path"
func (m *Manager) ParseSafeName(safeName string) string {
	parts := strings.SplitN(safeName, "__", 2)
	if len(parts) < 2 {
		return safeName
	}
	rest := strings.ReplaceAll(parts[1], "__", "/")
	return parts[0] + ":" + rest
}

// GetLinkName returns a clean name for symlinks in project directories.
// e.g. "local:figma-mcp" -> "figma-mcp", "github:user/repo/my-skill" -> "my-skill"
func (m *Manager) GetLinkName(id string) string {
	parts := strings.SplitN(id, ":", 2)
	name := id
	if len(parts) == 2 {
		name = parts[1]
	}
	// Use the last path segment
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

// GetRepoPath returns the on-disk path for a skill's cloned repository.
func (m *Manager) GetRepoPath(id string) string {
	return filepath.Join(m.repoDir, m.GetSafeName(id))
}

// GetAITools returns user-configured AI tools, or nil if not configured.
func (m *Manager) GetAITools() []AIToolConfig {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	if len(cfg.AITools) > 0 {
		return cfg.AITools
	}
	return nil
}

// LoadConfig reads and returns the full configuration.
func (m *Manager) LoadConfig() (*Config, error) {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetRegistry returns the configured registry URL, or empty string if not set.
func (m *Manager) GetRegistry() string {
	cfg, err := m.LoadConfig()
	if err != nil {
		return ""
	}
	return cfg.Registry
}

// SetRegistry saves the registry URL to config.
func (m *Manager) SetRegistry(url string) error {
	cfg, err := m.LoadConfig()
	if err != nil {
		cfg = &Config{System: runtime.GOOS}
	}
	cfg.Registry = url
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(m.configFile, data, 0644)
}

// GetRegistryDir returns the path where the registry repo is cloned.
func (m *Manager) GetRegistryDir() string {
	return filepath.Join(m.homeDir, "registry")
}
