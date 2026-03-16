package analyzer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules    RulesConfig    `json:"rules"    yaml:"rules"`
	Patterns PatternsConfig `json:"patterns" yaml:"patterns"`
}

type RulesConfig struct {
	LowercaseStart  bool `json:"lowercaseStart"  yaml:"lowercase_start"`
	EnglishOnly     bool `json:"englishOnly"     yaml:"english_only"`
	NoSpecialChars  bool `json:"noSpecialChars"  yaml:"no_special_chars"`
	NoSensitiveData bool `json:"noSensitiveData" yaml:"no_sensitive_data"`
}

type PatternsConfig struct {
	SensitiveKeywords []string `json:"sensitiveKeywords" yaml:"sensitive_keywords"`
}

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

func loadConfigFile(path string) (*Config, error) {
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, wrapReadError(readErr)
	}

	cfg := DefaultConfig()
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
			return nil, wrapUnmarshalError(unmarshalErr)
		}
	case ".yml", ".yaml":
		if unmarshalErr := yaml.Unmarshal(data, cfg); unmarshalErr != nil {
			return nil, wrapUnmarshalError(unmarshalErr)
		}
	}

	if len(cfg.Patterns.SensitiveKeywords) == 0 {
		cfg.Patterns.SensitiveKeywords = DefaultSensitiveKeywords()
	}

	return cfg, nil
}

var (
	ErrConfigRead      = errors.New("failed to read config file")
	ErrConfigUnmarshal = errors.New("failed to unmarshal config file")
)

func wrapReadError(err error) error {
	return errors.Join(ErrConfigRead, err)
}

func wrapUnmarshalError(err error) error {
	return errors.Join(ErrConfigUnmarshal, err)
}
