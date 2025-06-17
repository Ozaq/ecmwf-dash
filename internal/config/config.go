package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub         GitHubConfig         `yaml:"github"`
	FetchIntervals FetchIntervalsConfig `yaml:"fetch_intervals"`
	Server         ServerConfig         `yaml:"server"`
}

type GitHubConfig struct {
	Organization string   `yaml:"organization"`
	Repositories []string `yaml:"repositories"`
}

type FetchIntervalsConfig struct {
	Issues       time.Duration `yaml:"issues"`
	PullRequests time.Duration `yaml:"pull_requests"`
	Actions      time.Duration `yaml:"actions"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
