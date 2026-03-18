package analyzer

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	// We use a custom test approach since analysistest requires
	// specific directory structure with Go modules.
	// Integration tests are in the cmd/loglint package.
	t.Run("analyzer is configured", func(t *testing.T) {
		t.Parallel()

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
	t.Parallel()

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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := isEmoji(testCase.r)
			if got != testCase.want {
				t.Errorf("isEmoji(%U %q) = %v, want %v", testCase.r, string(testCase.r), got, testCase.want)
			}
		})
	}
}

func TestIsSpecialChar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{"letter", 'a', false},
		{"digit", '1', false},
		{"space", ' ', false},
		{"dot", '.', true},
		{"comma", ',', true},
		{"colon", ':', true},
		{"exclamation", '!', true},
		{"cyrillic", '\u0430', false}, // а (Russian) — letters are caught by english_only rule
		{"emoji rocket", '\U0001f680', true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := isSpecialChar(testCase.r)
			if got != testCase.want {
				t.Errorf("isSpecialChar(%U %q) = %v, want %v", testCase.r, string(testCase.r), got, testCase.want)
			}
		})
	}
}

func TestCleanSpecialChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with emoji", "server started \U0001f680", "server started"},
		{"clean", "hello world", "hello world"},
		{"emoji in middle", "error! \U0001f4a5 boom", "error boom"},
		{"only emoji", "\U0001f680\U0001f525", ""},
		{"punctuation", "error: bad!", "error bad"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := cleanSpecialChars(testCase.input)
			if got != testCase.want {
				t.Errorf("cleanSpecialChars(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	keywords := DefaultSensitiveKeywords()
	expected := []string{"password", "token", "api_key", "secret", "credentials"}

	for _, keyword := range expected {
		found := false

		for _, candidate := range keywords {
			if candidate == keyword {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected keyword %q not found in defaults", keyword)
		}
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	t.Parallel()

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
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
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
	t.Parallel()

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
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
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
	t.Parallel()

	cfg, err := loadConfigFile("/nonexistent/.loglint.yml")
	if err == nil {
		t.Error("expected error for missing config file")
	}

	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
}

func TestCheckExprForSensitiveEmptyKeywords(t *testing.T) {
	t.Parallel()

	diagnostics := 0
	pass := &analysis.Pass{ //nolint:exhaustruct
		Report: func(_ analysis.Diagnostic) {
			diagnostics++
		},
	}

	checkExprForSensitive(pass, &ast.Ident{Name: "token"}, nil, true)                                  //nolint:exhaustruct
	checkExprForSensitive(pass, &ast.BasicLit{Kind: token.STRING, Value: `"token"`}, []string{}, true) //nolint:exhaustruct

	if diagnostics != 0 {
		t.Errorf("expected no diagnostics with empty keywords, got %d", diagnostics)
	}
}

func TestCheckLowercaseStart(t *testing.T) {
	t.Parallel()

	diags := collectDiagnostics(func(pass *analysis.Pass) {
		checkLowercaseStart(pass, token.Pos(1), token.Pos(5), "Hello")
	})

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}

	if !strings.Contains(diags[0].Message, "lowercase") {
		t.Fatalf("unexpected diagnostic: %q", diags[0].Message)
	}
}

func TestCheckEnglishOnly(t *testing.T) {
	t.Parallel()

	diags := collectDiagnostics(func(pass *analysis.Pass) {
		checkEnglishOnly(pass, token.Pos(1), token.Pos(10), "запуск сервера")
	})

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

func TestCheckNoSpecialChars(t *testing.T) {
	t.Parallel()

	diags := collectDiagnostics(func(pass *analysis.Pass) {
		checkNoSpecialChars(pass, token.Pos(1), token.Pos(10), "server started!")
	})

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

func TestCheckExprForSensitive(t *testing.T) {
	t.Parallel()

	keywords := []string{"token"}
	diags := collectDiagnostics(func(pass *analysis.Pass) {
		checkExprForSensitive(pass, &ast.Ident{Name: "tokenValue"}, keywords, false) //nolint:exhaustruct
	})

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

func TestCheckExprForSensitiveLiteral(t *testing.T) {
	t.Parallel()

	keywords := []string{"token"}
	diags := collectDiagnostics(func(pass *analysis.Pass) {
		checkExprForSensitive(pass, &ast.BasicLit{Kind: token.STRING, Value: `"token"`}, keywords, true) //nolint:exhaustruct
	})

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

func collectDiagnostics(run func(*analysis.Pass)) []analysis.Diagnostic {
	diagnostics := make([]analysis.Diagnostic, 0, 4)
	pass := &analysis.Pass{ //nolint:exhaustruct
		Report: func(d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	run(pass)

	return diagnostics
}

func TestExtractLogMessageSlogSelectors(t *testing.T) {
	t.Parallel()

	ctx := &ast.Ident{Name: "ctx"}     //nolint:exhaustruct
	level := &ast.Ident{Name: "level"} //nolint:exhaustruct
	msg := stringLit("msg")

	tests := []struct {
		name      string
		call      *ast.CallExpr
		wantIndex int
		wantNil   bool
	}{
		{
			name: "slog info",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "slog"}, //nolint:exhaustruct
					Sel: &ast.Ident{Name: "Info"}, //nolint:exhaustruct
				},
				Args: []ast.Expr{msg},
			},
			wantIndex: 0,
			wantNil:   false,
		},
		{
			name: "slog log",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "slog"}, //nolint:exhaustruct
					Sel: &ast.Ident{Name: "Log"},  //nolint:exhaustruct
				},
				Args: []ast.Expr{ctx, level, msg},
			},
			wantIndex: 2,
			wantNil:   false,
		},
		{
			name: "slog info context",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "slog"},        //nolint:exhaustruct
					Sel: &ast.Ident{Name: "InfoContext"}, //nolint:exhaustruct
				},
				Args: []ast.Expr{ctx, msg},
			},
			wantIndex: 1,
			wantNil:   false,
		},
	}

	assertExtractLogMessage(t, tests, msg)
}

func TestExtractLogMessageReceivers(t *testing.T) {
	t.Parallel()

	msg := stringLit("msg")

	tests := []struct {
		name      string
		call      *ast.CallExpr
		wantIndex int
		wantNil   bool
	}{
		{
			name: "std log ident",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun:  &ast.Ident{Name: "Printf"}, //nolint:exhaustruct
				Args: []ast.Expr{msg},
			},
			wantIndex: 0,
			wantNil:   false,
		},
		{
			name: "known receiver slog",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "logger"}, //nolint:exhaustruct
					Sel: &ast.Ident{Name: "Info"},   //nolint:exhaustruct
				},
				Args: []ast.Expr{msg},
			},
			wantIndex: 0,
			wantNil:   false,
		},
		{
			name: "known receiver zap sugar",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "sugar"}, //nolint:exhaustruct
					Sel: &ast.Ident{Name: "Infow"}, //nolint:exhaustruct
				},
				Args: []ast.Expr{msg},
			},
			wantIndex: 0,
			wantNil:   false,
		},
		{
			name: "known receiver zap logger",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "zap"},  //nolint:exhaustruct
					Sel: &ast.Ident{Name: "Info"}, //nolint:exhaustruct
				},
				Args: []ast.Expr{msg},
			},
			wantIndex: 0,
			wantNil:   false,
		},
	}

	assertExtractLogMessage(t, tests, msg)
}

func TestExtractLogMessageFallbacks(t *testing.T) {
	t.Parallel()

	msg := stringLit("msg")

	tests := []struct {
		name      string
		call      *ast.CallExpr
		wantIndex int
		wantNil   bool
	}{
		{
			name: "call expr receiver",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.CallExpr{},           //nolint:exhaustruct
					Sel: &ast.Ident{Name: "Infof"}, //nolint:exhaustruct
				},
				Args: []ast.Expr{msg},
			},
			wantIndex: 0,
			wantNil:   false,
		},
		{
			name: "zap specific fallback",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun: &ast.SelectorExpr{
					X:   &ast.IndexExpr{},          //nolint:exhaustruct
					Sel: &ast.Ident{Name: "Infof"}, //nolint:exhaustruct
				},
				Args: []ast.Expr{msg},
			},
			wantIndex: 0,
			wantNil:   false,
		},
		{
			name: "no args",
			call: &ast.CallExpr{ //nolint:exhaustruct
				Fun:  &ast.Ident{Name: "Printf"}, //nolint:exhaustruct
				Args: nil,
			},
			wantIndex: -1,
			wantNil:   true,
		},
	}

	assertExtractLogMessage(t, tests, msg)
}

func TestGetLiteralMessage(t *testing.T) {
	t.Parallel()

	msg, pos, end, ok := getLiteralMessage(stringLit("hello"))
	if !ok {
		t.Fatal("expected literal message")
	}

	if msg != "hello" {
		t.Fatalf("unexpected message: %q", msg)
	}

	if pos == token.NoPos || end == token.NoPos {
		t.Fatalf("expected positions, got %v %v", pos, end)
	}
}

func assertExtractLogMessage(t *testing.T, tests []struct {
	name      string
	call      *ast.CallExpr
	wantIndex int
	wantNil   bool
}, msg ast.Expr,
) {
	t.Helper()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotExpr, gotIdx := extractLogMessage(testCase.call)
			if testCase.wantNil {
				if gotExpr != nil || gotIdx != -1 {
					t.Fatalf("expected nil/-1, got %v/%d", gotExpr, gotIdx)
				}

				return
			}

			if gotExpr != msg {
				t.Fatalf("unexpected message expr: %#v", gotExpr)
			}

			if gotIdx != testCase.wantIndex {
				t.Fatalf("unexpected index: %d", gotIdx)
			}
		})
	}
}

func stringLit(value string) *ast.BasicLit {
	return &ast.BasicLit{ //nolint:exhaustruct
		Kind:  token.STRING,
		Value: `"` + value + `"`,
	}
}

func TestSlogMethods(t *testing.T) {
	t.Parallel()

	expected := []string{"Info", "Warn", "Error", "Debug", "InfoContext", "WarnContext", "ErrorContext", "DebugContext"}
	for _, method := range expected {
		if !slogMethods[method] {
			t.Errorf("expected slog method %q to be recognized", method)
		}
	}
}

func TestZapSugarMethods(t *testing.T) {
	t.Parallel()

	expected := []string{
		"Info", "Infof", "Infow",
		"Warn", "Warnf",
		"Error", "Errorf",
		"Debug", "Debugf",
		"Fatal", "Fatalf",
	}
	for _, method := range expected {
		if !zapSugarMethods[method] {
			t.Errorf("expected zap sugar method %q to be recognized", method)
		}
	}
}
