package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		GitHub: GitHubConfig{
			Organization: "ecmwf",
			Repositories: []RepositoryConfig{
				{Name: "eccodes", Branches: []string{"master", "develop"}},
			},
		},
		FetchIntervals: FetchIntervalsConfig{
			Issues:       30 * time.Minute,
			PullRequests: 10 * time.Minute,
			Actions:      5 * time.Minute,
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8000,
		},
	}
}

func TestValidConfigPasses(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should pass: %v", err)
	}
}

func TestValidateEmptyOrg(t *testing.T) {
	cfg := validConfig()
	cfg.GitHub.Organization = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty organization")
	}
	if !strings.Contains(err.Error(), "organization") {
		t.Errorf("error should mention organization: %v", err)
	}
}

func TestValidateNoRepos(t *testing.T) {
	cfg := validConfig()
	cfg.GitHub.Repositories = nil
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for no repositories")
	}
}

func TestValidateRepoBranches(t *testing.T) {
	cfg := validConfig()
	cfg.GitHub.Repositories[0].Branches = nil
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for no branches")
	}
}

func TestValidateZeroDuration(t *testing.T) {
	cfg := validConfig()
	cfg.FetchIntervals.Issues = 0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for zero issues interval")
	}
}

func TestValidateNegativePort(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = -1
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative port")
	}
}

func TestValidatePortTooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 70000
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for port > 65535")
	}
}
