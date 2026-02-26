package config

import (
	"os"
	"path/filepath"
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

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `github:
  organization: ecmwf
  repositories:
    - name: eccodes
      branches: [master, develop]
    - name: atlas
      branches: [main]
fetch_intervals:
  issues: 30m
  pull_requests: 10m
  actions: 5m
server:
  host: "0.0.0.0"
  port: 8000
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.GitHub.Organization != "ecmwf" {
		t.Errorf("Organization = %q, want %q", cfg.GitHub.Organization, "ecmwf")
	}
	if len(cfg.GitHub.Repositories) != 2 {
		t.Fatalf("len(Repositories) = %d, want 2", len(cfg.GitHub.Repositories))
	}
	if cfg.GitHub.Repositories[0].Name != "eccodes" {
		t.Errorf("Repositories[0].Name = %q, want %q", cfg.GitHub.Repositories[0].Name, "eccodes")
	}
	if len(cfg.GitHub.Repositories[0].Branches) != 2 {
		t.Errorf("Repositories[0].Branches = %v, want [master develop]", cfg.GitHub.Repositories[0].Branches)
	}
	if cfg.GitHub.Repositories[1].Name != "atlas" {
		t.Errorf("Repositories[1].Name = %q, want %q", cfg.GitHub.Repositories[1].Name, "atlas")
	}
	if cfg.FetchIntervals.Issues != 30*time.Minute {
		t.Errorf("Issues interval = %v, want %v", cfg.FetchIntervals.Issues, 30*time.Minute)
	}
	if cfg.FetchIntervals.PullRequests != 10*time.Minute {
		t.Errorf("PullRequests interval = %v, want %v", cfg.FetchIntervals.PullRequests, 10*time.Minute)
	}
	if cfg.FetchIntervals.Actions != 5*time.Minute {
		t.Errorf("Actions interval = %v, want %v", cfg.FetchIntervals.Actions, 5*time.Minute)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8000 {
		t.Errorf("Port = %d, want %d", cfg.Server.Port, 8000)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("error should mention reading config file: %v", err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Invalid YAML: tab indentation mixed with bad syntax
	content := "github:\n\t- [invalid yaml{{{"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error should mention parsing config: %v", err)
	}
}

func TestLoad_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Valid YAML but empty organization triggers validation error
	content := `github:
  organization: ""
  repositories:
    - name: eccodes
      branches: [master]
fetch_intervals:
  issues: 30m
  pull_requests: 10m
  actions: 5m
server:
  host: "0.0.0.0"
  port: 8000
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty organization")
	}
	if !strings.Contains(err.Error(), "invalid config") {
		t.Errorf("error should mention invalid config: %v", err)
	}
	if !strings.Contains(err.Error(), "organization") {
		t.Errorf("error should mention organization: %v", err)
	}
}
