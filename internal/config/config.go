package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	AutoRun   bool   `yaml:"auto_run"`
	Clipboard *bool  `yaml:"clipboard"`
}

// Path returns the config file location, ~/.genie.yaml.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".genie.yaml")
}

// Load reads the config at path. A missing file is not an error; a file
// that can't be parsed is, so typos don't silently fall back to defaults.
func Load(path string) (Config, error) {
	var cfg Config
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// ClipboardEnabled reports whether to copy commands; it defaults to true.
func (c Config) ClipboardEnabled() bool {
	return c.Clipboard == nil || *c.Clipboard
}
