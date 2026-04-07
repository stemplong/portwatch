// Package config handles loading and validating portwatch configuration
// from a YAML file or environment variables.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the full portwatch daemon configuration.
type Config struct {
	// ScanInterval is how often to scan open ports.
	ScanInterval time.Duration `yaml:"scan_interval"`

	// Interfaces lists which network interfaces to monitor (empty = all).
	Interfaces []string `yaml:"interfaces"`

	// AllowedPorts defines ports that are expected to be open.
	// Changes outside this list trigger alerts.
	AllowedPorts []int `yaml:"allowed_ports"`

	// Alert contains notification settings.
	Alert AlertConfig `yaml:"alert"`

	// StateFile is the path where port state is persisted between runs.
	StateFile string `yaml:"state_file"`

	// LogLevel controls verbosity: debug, info, warn, error.
	LogLevel string `yaml:"log_level"`
}

// AlertConfig holds webhook and email notification settings.
type AlertConfig struct {
	Webhook WebhookConfig `yaml:"webhook"`
	Email   EmailConfig   `yaml:"email"`
}

// WebhookConfig configures HTTP webhook alerts.
type WebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	// Secret is used to sign webhook payloads (HMAC-SHA256).
	Secret  string `yaml:"secret"`
	Timeout time.Duration `yaml:"timeout"`
}

// EmailConfig configures SMTP email alerts.
type EmailConfig struct {
	Enabled    bool     `yaml:"enabled"`
	SMTPHost   string   `yaml:"smtp_host"`
	SMTPPort   int      `yaml:"smtp_port"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
	From       string   `yaml:"from"`
	To         []string `yaml:"to"`
	UseTLS     bool     `yaml:"use_tls"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ScanInterval: 30 * time.Second,
		StateFile:    "/var/lib/portwatch/state.json",
		LogLevel:     "info",
		Alert: AlertConfig{
			Webhook: WebhookConfig{
				Timeout: 10 * time.Second,
			},
			Email: EmailConfig{
				SMTPPort: 587,
				UseTLS:   true,
			},
		},
	}
}

// Load reads and parses a YAML config file at the given path.
// Missing optional fields are filled with defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks that required fields are set and values are sensible.
func (c *Config) Validate() error {
	if c.ScanInterval < time.Second {
		return fmt.Errorf("scan_interval must be at least 1s, got %s", c.ScanInterval)
	}

	if c.Alert.Webhook.Enabled && c.Alert.Webhook.URL == "" {
		return fmt.Errorf("alert.webhook.url is required when webhook alerts are enabled")
	}

	if c.Alert.Email.Enabled {
		if c.Alert.Email.SMTPHost == "" {
			return fmt.Errorf("alert.email.smtp_host is required when email alerts are enabled")
		}
		if c.Alert.Email.From == "" {
			return fmt.Errorf("alert.email.from is required when email alerts are enabled")
		}
		if len(c.Alert.Email.To) == 0 {
			return fmt.Errorf("alert.email.to must have at least one recipient when email alerts are enabled")
		}
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("log_level must be one of debug/info/warn/error, got %q", c.LogLevel)
	}

	return nil
}
