package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the linter configuration.
type Config struct {
	Rules    RulesConfig    `json:"rules" yaml:"rules"`
	Patterns PatternsConfig `json:"patterns" yaml:"patterns"`
}

// RulesConfig allows enabling/disabling individual rules.
type RulesConfig struct {
	LowercaseStart  bool `json:"lowercase_start" yaml:"lowercase_start"`
	EnglishOnly     bool `json:"english_only" yaml:"english_only"`
	NoSpecialChars  bool `json:"no_special_chars" yaml:"no_special_chars"`
	NoSensitiveData bool `json:"no_sensitive_data" yaml:"no_sensitive_data"`
}

// PatternsConfig allows specifying custom patterns for sensitive data detection.
type PatternsConfig struct {
	SensitiveKeywords []string `json:"sensitive_keywords" yaml:"sensitive_keywords"`
}

// DefaultConfig returns the default configuration with all rules enabled.
func DefaultConfig() *Config {
	return &Config{
		Rules: RulesConfig{
			LowercaseStart:  true,
			EnglishOnly:     true,
			NoSpecialChars:  true,
			NoSensitiveData: true,
		},
		Patterns: PatternsConfig{
			SensitiveKeywords: DefaultSensitiveKeywords(),
		},
	}
}

// DefaultSensitiveKeywords returns the default list of sensitive data keywords.
func DefaultSensitiveKeywords() []string {
	return []string{
		"password", "passwd", "pwd",
		"secret", "token",
		"api_key", "apikey", "api-key",
		"private_key", "privatekey", "private-key",
		"access_token", "accesstoken", "access-token",
		"refresh_token", "refreshtoken", "refresh-token",
		"credentials", "credential",
		"auth_token", "authtoken", "auth-token",
		"session_id", "sessionid", "session-id",
		"credit_card", "creditcard", "credit-card",
		"ssn", "social_security",
	}
}

// LoadConfig attempts to load configuration from known file locations.
// It searches for .loglint.yml, .loglint.yaml, or .loglint.json
// in the current directory and parent directories.
func LoadConfig() *Config {
	configNames := []string{".loglint.yml", ".loglint.yaml", ".loglint.json"}

	dir, err := os.Getwd()
	if err != nil {
		return DefaultConfig()
	}

	for {
		for _, name := range configNames {
			path := filepath.Join(dir, name)
			if cfg, err := loadConfigFile(path); err == nil {
				return cfg
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return DefaultConfig()
}

// LoadConfigFromPath loads configuration from a specific file path.
func LoadConfigFromPath(path string) (*Config, error) {
	return loadConfigFile(path)
}

func loadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// If custom sensitive keywords are not provided, use defaults.
	if len(cfg.Patterns.SensitiveKeywords) == 0 {
		cfg.Patterns.SensitiveKeywords = DefaultSensitiveKeywords()
	}

	return cfg, nil
}
