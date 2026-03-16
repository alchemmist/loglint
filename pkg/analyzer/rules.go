package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
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

	if rep, found := findRepeatedPunctuation(msg); found {
		pass.Report(analysis.Diagnostic{
			Pos:      litPos,
			End:      litEnd,
			Message:  fmt.Sprintf("log message should not contain repeated punctuation %q in %q", rep, msg),
			Category: "",
			URL:      "",
			Related:  nil,
			SuggestedFixes: []analysis.SuggestedFix{
				{
					Message: "clean up repeated punctuation",
					TextEdits: []analysis.TextEdit{
						{
							Pos:     litPos,
							End:     litEnd,
							NewText: []byte(cleanRepeatedPunctuation(msg)),
						},
					},
				},
			},
		})
	}
}

const minRepeatedPunctuation = 2

func findRepeatedPunctuation(s string) (string, bool) {
	runes := []rune(s)

	for idx := range len(runes) - 1 {
		currentRune := runes[idx]
		if !unicode.IsLetter(currentRune) && !unicode.IsDigit(currentRune) && !unicode.IsSpace(currentRune) {
			count := 1

			for nextIdx := idx + 1; nextIdx < len(runes) && runes[nextIdx] == currentRune; nextIdx++ {
				count++
			}

			if count >= minRepeatedPunctuation {
				return string(runes[idx : idx+count]), true
			}
		}
	}

	return "", false
}

func cleanRepeatedPunctuation(s string) string {
	runes := []rune(s)

	var builder strings.Builder

	for idx := 0; idx < len(runes); idx++ {
		builder.WriteRune(runes[idx])

		if !unicode.IsLetter(runes[idx]) && !unicode.IsDigit(runes[idx]) && !unicode.IsSpace(runes[idx]) {
			for idx+1 < len(runes) && runes[idx+1] == runes[idx] {
				idx++
			}
		}
	}

	return strings.TrimSpace(builder.String())
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

	allowed := ".,;:'-/?!=()[]{}@#$%^&*_+\\|<>`~\""
	if strings.ContainsRune(allowed, runeChar) {
		return false
	}

	if runeChar >= 0x20 && runeChar <= 0x7E {
		return false
	}

	return runeChar > unicode.MaxASCII
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

func checkNoSensitiveData(pass *analysis.Pass, pos, end token.Pos, expr ast.Expr, keywords []string) {
	if binExpr, ok := expr.(*ast.BinaryExpr); ok && binExpr.Op == token.ADD {
		checkSensitiveConcatenation(pass, pos, end, binExpr, keywords)
		return
	}

	if call, ok := expr.(*ast.CallExpr); ok {
		checkSensitiveFmtCall(pass, pos, end, call, keywords)
	}
}

func checkSensitiveConcatenation(pass *analysis.Pass, pos, end token.Pos, expr *ast.BinaryExpr, keywords []string) {
	ast.Inspect(expr, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			nameLower := strings.ToLower(ident.Name)
			for _, keyword := range keywords {
				if strings.Contains(nameLower, keyword) {
					pass.Report(analysis.Diagnostic{
						Pos: pos,
						End: end,
						Message: fmt.Sprintf("log message may contain sensitive data: variable %q matches sensitive keyword"+
							" %q", ident.Name, keyword),
						Category:       "",
						URL:            "",
						Related:        nil,
						SuggestedFixes: nil,
					})

					return false
				}
			}
		}

		return true
	})
}

func checkSensitiveFmtCall(pass *analysis.Pass, pos, end token.Pos, call *ast.CallExpr, keywords []string) {
	_, funcName, ok := isFmtCall(call)
	if !ok {
		return
	}

	args := fmtArgsToCheck(funcName, call)
	if len(args) == 0 {
		return
	}

	checkArgsForSensitive(pass, pos, end, args, keywords)
}

func checkArgsForSensitive(pass *analysis.Pass, pos, end token.Pos, args []ast.Expr, keywords []string) {
	for _, arg := range args {
		ast.Inspect(arg, func(node ast.Node) bool {
			identNode, ok := node.(*ast.Ident)
			if !ok {
				return true
			}

			nameLower := strings.ToLower(identNode.Name)

			for _, keyword := range keywords {
				if strings.Contains(nameLower, keyword) {
					pass.Report(analysis.Diagnostic{
						Pos: pos,
						End: end,
						Message: fmt.Sprintf("log message may contain sensitive data: variable %q matches sensitive keyword"+
							" %q", identNode.Name, keyword),
						Category:       "",
						URL:            "",
						Related:        nil,
						SuggestedFixes: nil,
					})

					return false
				}
			}

			return true
		})
	}
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

func isFmtCall(call *ast.CallExpr) (*ast.Ident, string, bool) {
	sel, isSelector := call.Fun.(*ast.SelectorExpr)
	if !isSelector {
		return nil, "", false
	}

	identExpr, isIdent := sel.X.(*ast.Ident)
	if !isIdent || identExpr.Name != "fmt" {
		return nil, "", false
	}

	return identExpr, sel.Sel.Name, true
}
