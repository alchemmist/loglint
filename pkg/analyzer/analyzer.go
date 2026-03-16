package analyzer

import (
	"errors"
	"flag"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	configFlag string
	fixFlag    bool
)

func NewAnalyzer() *analysis.Analyzer {
	analyzer := &analysis.Analyzer{
		Name: "loglint",
		Doc: "checks log messages for common issues: " +
			"uppercase start, non-English text, " +
			"special characters, and sensitive data",
		Requires:         []*analysis.Analyzer{inspect.Analyzer},
		Run:              run,
		URL:              "https://github.com/alchemmsit/loglint",
		Flags:            *flag.NewFlagSet("loglint", flag.ExitOnError),
		RunDespiteErrors: false,
		ResultType:       nil,
		FactTypes:        nil,
	}

	analyzer.Flags.StringVar(&configFlag, "config", "", "path to loglint configuration file")
	analyzer.Flags.BoolVar(&fixFlag, "fix", false, "apply suggested automatic fixes")

	return analyzer
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

var ErrNoInspector = errors.New("expected inspector.Inspector from inspect.Analyzer")

func run(pass *analysis.Pass) (any, error) {
	var cfg *Config

	if configFlag != "" {
		var err error

		cfg, err = loadConfigFile(configFlag)
		if err != nil {
			cfg = DefaultConfig()
		}
	} else {
		cfg = LoadConfig()
	}

	res, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok || res == nil {
		return nil, ErrNoInspector
	}

	insp := res

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		analyzeCall(pass, call, cfg)
	})

	return struct{}{}, nil
}

func analyzeCall(pass *analysis.Pass, call *ast.CallExpr, cfg *Config) {
	msgArg, msgIndex := extractLogMessage(call)
	if msgArg == nil {
		return
	}

	lit, ok := getStringLiteral(msgArg)
	if !ok {
		return
	}

	msg, err := strconv.Unquote(lit.Value)
	if err != nil {
		return
	}

	if len(msg) == 0 {
		return
	}

	pos := lit.Pos() + 1
	end := lit.End() - 1

	if cfg.Rules.LowercaseStart {
		checkLowercaseStart(pass, pos, end, msg)
	}

	if cfg.Rules.EnglishOnly {
		checkEnglishOnly(pass, pos, end, msg)
	}

	if cfg.Rules.NoSpecialChars {
		checkNoSpecialChars(pass, pos, end, msg)
	}

	if cfg.Rules.NoSensitiveData {
		checkSensitiveDataInArgs(pass, call, msgIndex, cfg.Patterns.SensitiveKeywords)
	}
}

func checkSensitiveDataInArgs(pass *analysis.Pass, call *ast.CallExpr, msgIndex int, keywords []string) {
	if msgIndex < 0 || msgIndex >= len(call.Args) {
		return
	}

	msgArg := call.Args[msgIndex]
	checkNoSensitiveData(pass, msgArg.Pos(), msgArg.End(), msgArg, keywords)

	for i := msgIndex + 1; i < len(call.Args); i++ {
		arg := call.Args[i]
		checkSensitiveIdent(pass, arg, keywords)
	}
}

func checkSensitiveIdent(pass *analysis.Pass, expr ast.Expr, keywords []string) {
	ast.Inspect(expr, func(n ast.Node) bool {
		identNode, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		lowerName := strings.ToLower(identNode.Name)

		for _, keyword := range keywords {
			if strings.Contains(lowerName, keyword) {
				pass.Reportf(expr.Pos(),
					"log argument may contain sensitive data: variable %q matches sensitive keyword %q",
					identNode.Name, keyword)

				return false
			}
		}

		return true
	})
}

func extractLogMessage(call *ast.CallExpr) (ast.Expr, int) {
	if len(call.Args) == 0 {
		return nil, -1
	}

	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return extractSelectorLogMessage(fn, call)
	case *ast.Ident:
		if stdLogFunctions[fn.Name] {
			return call.Args[0], 0
		}
	}

	return nil, -1
}

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

	if expr, idx, ok := handleIdent(sel, call, methodName); ok {
		return expr, idx
	}

	if expr, idx, ok := handleCallExpr(sel, call, methodName); ok {
		return expr, idx
	}

	if isZapSpecificMethod(methodName) {
		return getZapSugarMessageArg(methodName, call)
	}

	return nil, -1
}

func handleIdent(sel *ast.SelectorExpr, call *ast.CallExpr, methodName string) (ast.Expr, int, bool) {
	receiverNode, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, -1, false
	}

	switch receiverNode.Name {
	case "slog":
		if slogMethods[methodName] {
			expr, idx := getSlogMessageArg(methodName, call)
			return expr, idx, true
		}
	case "log":
		if stdLogFunctions[methodName] {
			return call.Args[0], 0, true
		}
	}

	if knownLogReceivers[strings.ToLower(receiverNode.Name)] {
		if slogMethods[methodName] {
			expr, idx := getSlogMessageArg(methodName, call)
			return expr, idx, true
		}

		if zapSugarMethods[methodName] {
			expr, idx := getZapSugarMessageArg(methodName, call)
			return expr, idx, true
		}

		if zapLoggerMethods[methodName] {
			return call.Args[0], 0, true
		}
	}

	return nil, -1, false
}

func handleCallExpr(sel *ast.SelectorExpr, call *ast.CallExpr, methodName string) (ast.Expr, int, bool) {
	if _, ok := sel.X.(*ast.CallExpr); ok && zapSugarMethods[methodName] {
		expr, idx := getZapSugarMessageArg(methodName, call)
		return expr, idx, true
	}

	return nil, -1, false
}

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

const (
	slogLogMsgIndex     = 2
	slogContextMsgIndex = 1
)

func getSlogMessageArg(method string, call *ast.CallExpr) (ast.Expr, int) {
	if method == "Log" {
		if len(call.Args) > slogLogMsgIndex {
			return call.Args[slogLogMsgIndex], slogLogMsgIndex
		}

		return nil, -1
	}

	if strings.HasSuffix(method, "Context") {
		if len(call.Args) > slogContextMsgIndex {
			return call.Args[slogContextMsgIndex], slogContextMsgIndex
		}

		return nil, -1
	}

	return call.Args[0], 0
}

func getZapSugarMessageArg(_ string, call *ast.CallExpr) (ast.Expr, int) {
	return call.Args[0], 0
}

func getStringLiteral(expr ast.Expr) (*ast.BasicLit, bool) {
	litNode, ok := expr.(*ast.BasicLit)
	if !ok || litNode.Kind != token.STRING {
		return nil, false
	}

	return litNode, true
}
