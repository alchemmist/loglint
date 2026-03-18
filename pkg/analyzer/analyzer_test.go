package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzer(t *testing.T) {
	// We use a custom test approach since analysistest requires
	// specific directory structure with Go modules.
	// Integration tests are in the cmd/loglint package.
	t.Run("analyzer is configured", func(t *testing.T) {
		analyzer := NewAnalyzer()

		if analyzer.Name != "loglint" {
			t.Errorf("expected analyzer name 'loglint', got %q", analyzer.Name)
		}
		if analyzer.Run == nil {
			t.Error("analyzer Run function is nil")
		}
		if len(analyzer.Requires) == 0 {
			t.Error("analyzer should require inspect pass")
		}
	})
}

// --- Unit tests for individual rules (pure functions) ---

func TestIsEmoji(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{"rocket", '\U0001f680', true},
		{"crying face", '\U0001f622', true},
		{"collision", '\U0001f4a5', true},
		{"star", '\u2B50', false}, // not in our emoji ranges
		{"letter a", 'a', false},
		{"exclamation", '!', false},
		{"digit", '1', false},
		{"sun", '\u2600', true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmoji(tt.r)
			if got != tt.want {
				t.Errorf("isEmoji(%U %q) = %v, want %v", tt.r, string(tt.r), got, tt.want)
			}
		})
	}
}

func TestIsSpecialChar(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{"letter", 'a', false},
		{"digit", '1', false},
		{"space", ' ', false},
		{"dot", '.', false},
		{"comma", ',', false},
		{"colon", ':', false},
		{"exclamation", '!', false},
		{"cyrillic", '\u0430', false}, // а (Russian) — letters are caught by english_only rule
		{"emoji rocket", '\U0001f680', true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSpecialChar(tt.r)
			if got != tt.want {
				t.Errorf("isSpecialChar(%U %q) = %v, want %v", tt.r, string(tt.r), got, tt.want)
			}
		})
	}
}

func TestCleanSpecialChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with emoji", "server started \U0001f680", "server started"},
		{"clean", "hello world", "hello world"},
		{"emoji in middle", "error! \U0001f4a5 boom", "error! boom"},
		{"only emoji", "\U0001f680\U0001f525", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanSpecialChars(tt.input)
			if got != tt.want {
				t.Errorf("cleanSpecialChars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Rules.LowercaseStart {
		t.Error("LowercaseStart should be true by default")
	}
	if !cfg.Rules.EnglishOnly {
		t.Error("EnglishOnly should be true by default")
	}
	if !cfg.Rules.NoSpecialChars {
		t.Error("NoSpecialChars should be true by default")
	}
	if !cfg.Rules.NoSensitiveData {
		t.Error("NoSensitiveData should be true by default")
	}
	if len(cfg.Patterns.SensitiveKeywords) == 0 {
		t.Error("SensitiveKeywords should not be empty by default")
	}
}

func TestDefaultSensitiveKeywords(t *testing.T) {
	keywords := DefaultSensitiveKeywords()
	expected := []string{"password", "token", "api_key", "secret", "credentials"}
	for _, kw := range expected {
		found := false
		for _, k := range keywords {
			if k == kw {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected keyword %q not found in defaults", kw)
		}
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".loglint.yml")

	content := []byte(`rules:
  lowercase_start: true
  english_only: false
  no_special_chars: true
  no_sensitive_data: true
patterns:
  sensitive_keywords:
    - password
    - custom_secret
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := loadConfigFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !cfg.Rules.LowercaseStart {
		t.Error("expected LowercaseStart to be true")
	}
	if cfg.Rules.EnglishOnly {
		t.Error("expected EnglishOnly to be false")
	}
	if len(cfg.Patterns.SensitiveKeywords) != 2 {
		t.Errorf("expected 2 sensitive keywords, got %d", len(cfg.Patterns.SensitiveKeywords))
	}
}

func TestLoadConfigJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".loglint.json")

	content := []byte(`{
  "rules": {
    "lowercase_start": false,
    "english_only": true,
    "no_special_chars": false,
    "no_sensitive_data": true
  },
  "patterns": {
    "sensitive_keywords": ["password", "my_custom_key"]
  }
}`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := loadConfigFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Rules.LowercaseStart {
		t.Error("expected LowercaseStart to be false")
	}
	if !cfg.Rules.EnglishOnly {
		t.Error("expected EnglishOnly to be true")
	}
	if cfg.Rules.NoSpecialChars {
		t.Error("expected NoSpecialChars to be false")
	}
	if len(cfg.Patterns.SensitiveKeywords) != 2 {
		t.Errorf("expected 2 sensitive keywords, got %d", len(cfg.Patterns.SensitiveKeywords))
	}
}

func TestLoadConfigMissing(t *testing.T) {
	cfg, err := loadConfigFile("/nonexistent/.loglint.yml")
	if err == nil {
		t.Error("expected error for missing config file")
	}
	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
}

func TestSlogMethods(t *testing.T) {
	expected := []string{"Info", "Warn", "Error", "Debug", "InfoContext", "WarnContext", "ErrorContext", "DebugContext"}
	for _, m := range expected {
		if !slogMethods[m] {
			t.Errorf("expected slog method %q to be recognized", m)
		}
	}
}

func TestZapSugarMethods(t *testing.T) {
	expected := []string{"Info", "Infof", "Infow", "Warn", "Warnf", "Error", "Errorf", "Debug", "Debugf", "Fatal", "Fatalf"}
	for _, m := range expected {
		if !zapSugarMethods[m] {
			t.Errorf("expected zap sugar method %q to be recognized", m)
		}
	}
}
