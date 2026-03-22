package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Model   string `yaml:"model"`
	AutoRun bool   `yaml:"auto_run"`
}

func Load() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}
	}

	data, err := os.ReadFile(filepath.Join(home, ".genie.yaml"))
	if err != nil {
		return Config{}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	return cfg
}
