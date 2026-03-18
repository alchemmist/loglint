package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
)

func checkLowercaseStart(pass *analysis.Pass, litPos, litEnd token.Pos, msg string) {
	if len(msg) == 0 {
		return
	}

	firstRune, _ := utf8.DecodeRuneInString(msg)
	if unicode.IsUpper(firstRune) {
		fixed := strings.ToLower(string(firstRune)) + msg[utf8.RuneLen(firstRune):]
		pass.Report(analysis.Diagnostic{
			Pos:      litPos,
			End:      litEnd,
			Message:  fmt.Sprintf("log message should start with a lowercase letter, got %q", msg),
			Category: "",
			URL:      "",
			Related:  nil,
			SuggestedFixes: []analysis.SuggestedFix{
				{
					Message: "change first letter to lowercase",
					TextEdits: []analysis.TextEdit{
						{
							Pos:     litPos,
							End:     litEnd,
							NewText: []byte(fixed),
						},
					},
				},
			},
		})
	}
}

func checkEnglishOnly(pass *analysis.Pass, pos, end token.Pos, msg string) {
	for _, runeChar := range msg {
		if runeChar <= unicode.MaxASCII {
			continue
		}

		if isEmoji(runeChar) {
			continue
		}

		if unicode.IsLetter(runeChar) {
			pass.Report(analysis.Diagnostic{
				Pos: pos,
				End: end,
				Message: fmt.Sprintf(
					"log message should be in English only, found non-English character %q in %q",
					string(runeChar),
					msg,
				),
				Category:       "",
				URL:            "",
				Related:        nil,
				SuggestedFixes: nil,
			})

			return
		}
	}
}

func checkNoSpecialChars(pass *analysis.Pass, litPos, litEnd token.Pos, msg string) {
	for _, runeChar := range msg {
		if isEmoji(runeChar) || isSpecialChar(runeChar) {
			pass.Report(analysis.Diagnostic{
				Pos: litPos,
				End: litEnd,
				Message: fmt.Sprintf(
					"log message should not contain special characters or emojis, found %q in %q",
					string(runeChar),
					msg,
				),
				Category: "",
				URL:      "",
				Related:  nil,
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "remove special characters and emojis",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     litPos,
								End:     litEnd,
								NewText: []byte(cleanSpecialChars(msg)),
							},
						},
					},
				},
			})

			return
		}
	}
}

type runeRange struct {
	lo, hi rune
}

var emojiRanges = []runeRange{
	{0x1F600, 0x1F64F},
	{0x1F300, 0x1F5FF},
	{0x1F680, 0x1F6FF},
	{0x1F1E0, 0x1F1FF},
	{0x2600, 0x26FF},
	{0x2700, 0x27BF},
	{0xFE00, 0xFE0F},
	{0x1F900, 0x1F9FF},
	{0x1FA00, 0x1FA6F},
	{0x1FA70, 0x1FAFF},
	{0x200D, 0x200D},
	{0x231A, 0x231B},
}

func isEmoji(r rune) bool {
	for _, rng := range emojiRanges {
		if r >= rng.lo && r <= rng.hi {
			return true
		}
	}

	return false
}

func isSpecialChar(runeChar rune) bool {
	if unicode.IsLetter(runeChar) || unicode.IsDigit(runeChar) || unicode.IsSpace(runeChar) {
		return false
	}

	return true
}

func cleanSpecialChars(s string) string {
	var builder strings.Builder

	for _, runeChar := range s {
		if !isEmoji(runeChar) && !isSpecialChar(runeChar) {
			builder.WriteRune(runeChar)
		}
	}

	result := builder.String()
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return strings.TrimSpace(result)
}

func checkExprForSensitive(pass *analysis.Pass, expr ast.Expr, keywords []string, allowStringLiteral bool) {
	if expr == nil || len(keywords) == 0 {
		return
	}

	visitor := sensitiveVisitor{
		pass:               pass,
		keywords:           keywords,
		allowStringLiteral: allowStringLiteral,
	}
	ast.Inspect(expr, visitor.visit)
}

type sensitiveVisitor struct {
	pass               *analysis.Pass
	keywords           []string
	allowStringLiteral bool
}

func (v *sensitiveVisitor) visit(node ast.Node) bool {
	switch nodeValue := node.(type) {
	case *ast.BasicLit:
		return v.handleBasicLit(nodeValue)
	case *ast.Ident:
		return v.handleIdent(nodeValue)
	case *ast.CallExpr:
		return v.handleCallExpr(nodeValue)
	case *ast.BinaryExpr:
		v.handleBinaryExpr(nodeValue)
		return false
	default:
		return true
	}
}

func (v *sensitiveVisitor) handleBasicLit(lit *ast.BasicLit) bool {
	if !v.allowStringLiteral || lit.Kind != token.STRING {
		return true
	}

	unquoted, err := strconv.Unquote(lit.Value)
	if err != nil {
		return true
	}

	if v.reportLiteral(lit, unquoted) {
		return false
	}

	return true
}

func (v *sensitiveVisitor) handleIdent(ident *ast.Ident) bool {
	return !v.reportIdent(ident)
}

func (v *sensitiveVisitor) handleCallExpr(call *ast.CallExpr) bool {
	funcName, ok := isFmtCall(call)
	if !ok {
		return true
	}

	for _, arg := range fmtArgsToCheck(funcName, call) {
		checkExprForSensitive(v.pass, arg, v.keywords, v.allowStringLiteral)
	}

	return false
}

func (v *sensitiveVisitor) handleBinaryExpr(expr *ast.BinaryExpr) {
	checkExprForSensitive(v.pass, expr.X, v.keywords, v.allowStringLiteral)
	checkExprForSensitive(v.pass, expr.Y, v.keywords, v.allowStringLiteral)
}

func (v *sensitiveVisitor) reportIdent(ident *ast.Ident) bool {
	lower := strings.ToLower(ident.Name)
	for _, keyword := range v.keywords {
		if strings.Contains(lower, keyword) {
			v.pass.Report(analysis.Diagnostic{
				Pos: ident.Pos(),
				End: ident.End(),
				Message: fmt.Sprintf(
					"log message may contain sensitive data: variable %q matches sensitive keyword %q",
					ident.Name,
					keyword,
				),
				Category:       "",
				URL:            "",
				Related:        nil,
				SuggestedFixes: nil,
			})

			return true
		}
	}

	return false
}

func (v *sensitiveVisitor) reportLiteral(lit *ast.BasicLit, unquoted string) bool {
	lower := strings.ToLower(unquoted)
	for _, keyword := range v.keywords {
		if strings.Contains(lower, keyword) {
			v.pass.Report(analysis.Diagnostic{
				Pos: lit.Pos(),
				End: lit.End(),
				Message: fmt.Sprintf(
					"log message may contain sensitive data: literal %q matches sensitive keyword %q",
					unquoted,
					keyword,
				),
				Category:       "",
				URL:            "",
				Related:        nil,
				SuggestedFixes: nil,
			})

			return true
		}
	}

	return false
}

func fmtArgsToCheck(funcName string, call *ast.CallExpr) []ast.Expr {
	switch funcName {
	case "Sprintf", "Fprintf":
		if len(call.Args) > 1 {
			return call.Args[1:]
		}
	case "Sprint", "Sprintln":
		return call.Args
	}

	return nil
}

func isFmtCall(call *ast.CallExpr) (string, bool) {
	sel, isSelector := call.Fun.(*ast.SelectorExpr)
	if !isSelector {
		return "", false
	}

	identExpr, isIdent := sel.X.(*ast.Ident)
	if !isIdent || identExpr.Name != "fmt" {
		return "", false
	}

	return sel.Sel.Name, true
}
