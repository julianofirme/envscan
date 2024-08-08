package config

import (
	"os"

	"github.com/pelletier/go-toml"
)

type Rule struct {
	Description string   `toml:"description"`
	ID          string   `toml:"id"`
	Regex       string   `toml:"regex"`
	SecretGroup int      `toml:"secretGroup"`
	Keywords    []string `toml:"keywords"`
}

type Config struct {
	Rules  []Rule `toml:"rules"`
	Ignore Ignore `toml:"ignore"`
}

type Ignore struct {
	Files       []string `toml:"files"`
	Directories []string `toml:"directories"`
}

func LoadConfig(filePath string) (Config, error) {
	var config Config
	content, err := os.ReadFile(filePath)
	if err != nil {
		return config, err
	}
	err = toml.Unmarshal(content, &config)
	return config, err
}
