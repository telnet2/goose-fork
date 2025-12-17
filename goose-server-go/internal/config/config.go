package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds the server configuration
type Config struct {
	// Server settings
	Port      int    `yaml:"port"`
	SecretKey string `yaml:"-"` // Never serialize secret key

	// Paths
	DataDir    string `yaml:"data_dir"`
	ConfigFile string `yaml:"config_file"`

	// Provider settings
	DefaultProvider string `yaml:"default_provider"`
	DefaultModel    string `yaml:"default_model"`
}

// DefaultDataDir returns the default data directory based on OS
func DefaultDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "goose")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "goose")
	default: // linux and others
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			xdgData = filepath.Join(homeDir, ".local", "share")
		}
		return filepath.Join(xdgData, "goose")
	}
}

// DefaultConfigDir returns the default config directory based on OS
func DefaultConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "goose")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "goose")
	default: // linux and others
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(xdgConfig, "goose")
	}
}

// Load loads the configuration from environment variables and config file
func Load() (*Config, error) {
	cfg := &Config{
		Port:    3000,
		DataDir: DefaultDataDir(),
	}

	// Load from environment variables
	if portStr := os.Getenv("GOOSE_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Port = port
		}
	}

	if dataDir := os.Getenv("GOOSE_PATH_ROOT"); dataDir != "" {
		cfg.DataDir = dataDir
	}

	cfg.SecretKey = os.Getenv("GOOSE_SERVER__SECRET_KEY")

	cfg.DefaultProvider = os.Getenv("GOOSE_PROVIDER")
	cfg.DefaultModel = os.Getenv("GOOSE_MODEL")

	// Determine config file location
	configDir := DefaultConfigDir()
	cfg.ConfigFile = filepath.Join(configDir, "config.yaml")

	// Load from config file if exists
	if err := cfg.loadFromFile(); err != nil {
		// Config file is optional, ignore errors
		_ = err
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFromFile loads configuration from the config file
func (c *Config) loadFromFile() error {
	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return err
	}

	var fileCfg struct {
		Port            int    `yaml:"port"`
		DefaultProvider string `yaml:"default_provider"`
		DefaultModel    string `yaml:"default_model"`
	}

	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return err
	}

	// Only override if not set by environment
	if c.Port == 3000 && fileCfg.Port != 0 {
		c.Port = fileCfg.Port
	}
	if c.DefaultProvider == "" && fileCfg.DefaultProvider != "" {
		c.DefaultProvider = fileCfg.DefaultProvider
	}
	if c.DefaultModel == "" && fileCfg.DefaultModel != "" {
		c.DefaultModel = fileCfg.DefaultModel
	}

	return nil
}

// SessionsDBPath returns the path to the sessions database
func (c *Config) SessionsDBPath() string {
	return filepath.Join(c.DataDir, "sessions.db")
}

// ExtensionsDir returns the path to the extensions directory
func (c *Config) ExtensionsDir() string {
	return filepath.Join(c.DataDir, "extensions")
}

// RecipesDir returns the path to the recipes directory
func (c *Config) RecipesDir() string {
	return filepath.Join(c.DataDir, "recipes")
}
