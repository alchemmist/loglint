// Package analyzer implements a Go static analysis pass that checks log messages
// for compliance with logging best practices.
//
// The analyzer supports the following loggers:
//   - log/slog (Info, Warn, Error, Debug, and their context variants)
//   - go.uber.org/zap (Sugar methods: Infof, Warnf, Errorf, Debugf, Infow, etc.)
//
// Rules enforced:
//  1. Log messages must start with a lowercase letter.
//  2. Log messages must be in English only.
//  3. Log messages must not contain special characters or emojis.
//  4. Log messages must not contain potentially sensitive data.
package analyzer

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var configFlag string

var Analyzer = &analysis.Analyzer{
	Name:     "loglint",
	Doc:      "checks log messages for common issues: uppercase start, non-English text, special characters, and sensitive data",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func init() {
	Analyzer.Flags.StringVar(&configFlag, "config", "", "path to loglint configuration file")
}

var slogMethods = map[string]bool{
	"Info":         true,
	"Warn":         true,
	"Error":        true,
	"Debug":        true,
	"InfoContext":  true,
	"WarnContext":  true,
	"ErrorContext": true,
	"DebugContext": true,
	"Log":          true,
}

var zapSugarMethods = map[string]bool{
	"Info":    true,
	"Infof":   true,
	"Infow":   true,
	"Warn":    true,
	"Warnf":   true,
	"Warnw":   true,
	"Error":   true,
	"Errorf":  true,
	"Errorw":  true,
	"Debug":   true,
	"Debugf":  true,
	"Debugw":  true,
	"Fatal":   true,
	"Fatalf":  true,
	"Fatalw":  true,
	"Panic":   true,
	"Panicf":  true,
	"Panicw":  true,
	"DPanic":  true,
	"DPanicf": true,
	"DPanicw": true,
}

var zapLoggerMethods = map[string]bool{
	"Info":   true,
	"Warn":   true,
	"Error":  true,
	"Debug":  true,
	"Fatal":  true,
	"Panic":  true,
	"DPanic": true,
}

var stdLogFunctions = map[string]bool{
	"Print":   true,
	"Printf":  true,
	"Println": true,
	"Fatal":   true,
	"Fatalf":  true,
	"Fatalln": true,
	"Panic":   true,
	"Panicf":  true,
	"Panicln": true,
}

func run(pass *analysis.Pass) (any, error) {
	var cfg *Config
	if configFlag != "" {
		var err error
		cfg, err = LoadConfigFromPath(configFlag)
		if err != nil {
			cfg = DefaultConfig()
		}
	} else {
		cfg = LoadConfig()
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		analyzeCall(pass, call, cfg)
	})

	return nil, nil
}

// analyzeCall determines if a function call is a log call and inspects its message argument.
func analyzeCall(pass *analysis.Pass, call *ast.CallExpr, cfg *Config) {
	msgArg, msgIndex := extractLogMessage(call)
	if msgArg == nil {
		return
	}

	// For rules 1-3, we need the string literal value.
	if lit, ok := getStringLiteral(msgArg); ok {
		msg, err := strconv.Unquote(lit.Value)
		if err != nil {
			return
		}
		if len(msg) == 0 {
			return
		}

		pos := lit.Pos() + 1 // skip opening quote
		end := lit.End() - 1 // skip closing quote

		if cfg.Rules.LowercaseStart {
			checkLowercaseStart(pass, pos, end, msg)
		}
		if cfg.Rules.EnglishOnly {
			checkEnglishOnly(pass, pos, end, msg)
		}
		if cfg.Rules.NoSpecialChars {
			checkNoSpecialChars(pass, pos, end, msg)
		}
	}

	// Rule 4: Check for sensitive data (works on expressions, not just literals).
	if cfg.Rules.NoSensitiveData {
		checkSensitiveDataInArgs(pass, call, msgIndex, cfg.Patterns.SensitiveKeywords)
	}
}

// checkSensitiveDataInArgs checks the log call arguments for sensitive data patterns.
func checkSensitiveDataInArgs(pass *analysis.Pass, call *ast.CallExpr, msgIndex int, keywords []string) {
	if msgIndex < 0 || msgIndex >= len(call.Args) {
		return
	}

	// Check the message argument itself (could be concatenation).
	msgArg := call.Args[msgIndex]
	checkNoSensitiveData(pass, msgArg.Pos(), msgArg.End(), msgArg, keywords)

	// Also check remaining arguments for sensitive variable references.
	for i := msgIndex + 1; i < len(call.Args); i++ {
		arg := call.Args[i]
		checkSensitiveIdent(pass, arg, keywords)
	}
}

// checkSensitiveIdent checks if an expression references a sensitive variable.
func checkSensitiveIdent(pass *analysis.Pass, expr ast.Expr, keywords []string) {
	ast.Inspect(expr, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		name := strings.ToLower(ident.Name)
		for _, keyword := range keywords {
			if strings.Contains(name, keyword) {
				pass.Reportf(expr.Pos(),
					"log argument may contain sensitive data: variable %q matches sensitive keyword %q",
					ident.Name, keyword)
				return false
			}
		}
		return true
	})
}

// extractLogMessage identifies log calls and returns the message argument expression and its index.
// Returns (nil, -1) if the call is not a recognized log call.
func extractLogMessage(call *ast.CallExpr) (ast.Expr, int) {
	if len(call.Args) == 0 {
		return nil, -1
	}

	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return extractSelectorLogMessage(fn, call)
	case *ast.Ident:
		// Direct function calls like log.Print (after dot-import)
		if stdLogFunctions[fn.Name] {
			return call.Args[0], 0
		}
	}

	return nil, -1
}

// knownLogReceivers contains common variable names used for loggers.
// This reduces false positives from matching any method named Info/Error/etc.
var knownLogReceivers = map[string]bool{
	"log":    true,
	"logger": true,
	"l":      true,
	"sugar":  true,
	"s":      true,
	"zap":    true,
}

func extractSelectorLogMessage(sel *ast.SelectorExpr, call *ast.CallExpr) (ast.Expr, int) {
	methodName := sel.Sel.Name

	switch x := sel.X.(type) {
	case *ast.Ident:
		// Package-level calls: slog.Info("msg"), log.Print("msg")
		switch x.Name {
		case "slog":
			if slogMethods[methodName] {
				return getSlogMessageArg(methodName, call)
			}
		case "log":
			if stdLogFunctions[methodName] {
				return call.Args[0], 0
			}
		}

		// Method calls on known logger variable names: logger.Info("msg"), sugar.Infow("msg")
		if knownLogReceivers[strings.ToLower(x.Name)] {
			if slogMethods[methodName] {
				return getSlogMessageArg(methodName, call)
			}
			if zapSugarMethods[methodName] {
				return getZapSugarMessageArg(methodName, call)
			}
			if zapLoggerMethods[methodName] {
				return call.Args[0], 0
			}
		}

	case *ast.CallExpr:
		// Chained calls: logger.Sugar().Infow("msg")
		// sel.X is the Sugar() call expression, methodName is Infow/etc.
		if zapSugarMethods[methodName] {
			return getZapSugarMessageArg(methodName, call)
		}
	}

	// Only match zap-specific suffixed methods (Infof, Infow, Warnf, etc.)
	// to reduce false positives. These method names are very unlikely outside of loggers.
	if isZapSpecificMethod(methodName) {
		return getZapSugarMessageArg(methodName, call)
	}

	return nil, -1
}

// isZapSpecificMethod returns true for methods that are specific to zap's SugaredLogger
// and very unlikely to appear in non-logger code (e.g., Infof, Infow, Warnf, Debugw).
func isZapSpecificMethod(name string) bool {
	switch name {
	case "Infof", "Infow",
		"Warnf", "Warnw",
		"Errorf", "Errorw",
		"Debugf", "Debugw",
		"Fatalf", "Fatalw",
		"Panicf", "Panicw",
		"DPanicf", "DPanicw":
		return true
	}
	return false
}

// getSlogMessageArg returns the message argument for slog calls.
//
// Signatures:
//   - slog.Info(msg, args...)          → msg is Args[0]
//   - slog.InfoContext(ctx, msg, args...) → msg is Args[1]
//   - slog.Log(ctx, level, msg, args...) → msg is Args[2]
func getSlogMessageArg(method string, call *ast.CallExpr) (ast.Expr, int) {
	if method == "Log" {
		// slog.Log(ctx context.Context, level Level, msg string, args ...any)
		if len(call.Args) >= 3 {
			return call.Args[2], 2
		}
		return nil, -1
	}
	if strings.HasSuffix(method, "Context") {
		// slog.InfoContext(ctx context.Context, msg string, args ...any)
		if len(call.Args) >= 2 {
			return call.Args[1], 1
		}
		return nil, -1
	}
	// slog.Info(msg string, args ...any)
	return call.Args[0], 0
}

// getZapSugarMessageArg returns the message argument for zap sugar logger calls.
func getZapSugarMessageArg(method string, call *ast.CallExpr) (ast.Expr, int) {
	// All sugar methods take message as the first argument.
	return call.Args[0], 0
}

// getStringLiteral extracts a basic string literal from an expression.
func getStringLiteral(expr ast.Expr) (*ast.BasicLit, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			return e, true
		}
	}
	return nil, false
}
