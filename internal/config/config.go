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
	Organization string             `yaml:"organization"`
	Repositories []RepositoryConfig `yaml:"repositories"`
}

type RepositoryConfig struct {
	Name     string   `yaml:"name"`
	Branches []string `yaml:"branches"`
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

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required config fields have sensible values.
func (c *Config) Validate() error {
	var errs []string

	if c.GitHub.Organization == "" {
		errs = append(errs, "github.organization is required")
	}
	if len(c.GitHub.Repositories) == 0 {
		errs = append(errs, "at least one repository is required")
	}
	for i, repo := range c.GitHub.Repositories {
		if repo.Name == "" {
			errs = append(errs, fmt.Sprintf("repository[%d].name is required", i))
		}
		if len(repo.Branches) == 0 {
			errs = append(errs, fmt.Sprintf("repository[%d] (%s) needs at least one branch", i, repo.Name))
		}
	}

	if c.FetchIntervals.Issues <= 0 {
		errs = append(errs, "fetch_intervals.issues must be > 0")
	}
	if c.FetchIntervals.PullRequests <= 0 {
		errs = append(errs, "fetch_intervals.pull_requests must be > 0")
	}
	if c.FetchIntervals.Actions <= 0 {
		errs = append(errs, "fetch_intervals.actions must be > 0")
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port must be 1-65535, got %d", c.Server.Port))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", joinErrors(errs))
	}
	return nil
}

func joinErrors(errs []string) string {
	result := errs[0]
	for _, e := range errs[1:] {
		result += "; " + e
	}
	return result
}
