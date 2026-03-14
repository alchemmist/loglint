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

// checkLowercaseStart checks that the log message starts with a lowercase letter.
// Rule 1: Log messages must start with a lowercase letter.
func checkLowercaseStart(pass *analysis.Pass, litPos token.Pos, litEnd token.Pos, msg string) {
	if len(msg) == 0 {
		return
	}
	r, _ := utf8.DecodeRuneInString(msg)
	if unicode.IsUpper(r) {
		fixed := strings.ToLower(string(r)) + msg[utf8.RuneLen(r):]
		pass.Report(analysis.Diagnostic{
			Pos:     litPos,
			End:     litEnd,
			Message: fmt.Sprintf("log message should start with a lowercase letter, got %q", msg),
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

// checkEnglishOnly checks that the log message contains only English (ASCII) characters.
// Rule 2: Log messages must be in English only.
// Note: emojis and symbols are skipped here — they are handled by checkNoSpecialChars.
func checkEnglishOnly(pass *analysis.Pass, pos token.Pos, end token.Pos, msg string) {
	for _, r := range msg {
		if r <= unicode.MaxASCII {
			continue
		}
		// Skip emojis — they are handled by the no_special_chars rule.
		if isEmoji(r) {
			continue
		}
		// Non-ASCII letter that is not basic Latin = non-English text.
		if unicode.IsLetter(r) {
			pass.Report(analysis.Diagnostic{
				Pos:     pos,
				End:     end,
				Message: fmt.Sprintf("log message should be in English only, found non-English character %q in %q", string(r), msg),
			})
			return
		}
	}
}

// checkNoSpecialChars checks that the log message does not contain special characters or emojis.
// Rule 3: Log messages must not contain special characters or emojis.
// Also detects repeated punctuation like "!!!", "...", "???" which indicate noisy log messages.
func checkNoSpecialChars(pass *analysis.Pass, litPos token.Pos, litEnd token.Pos, msg string) {
	// Check for emojis and non-ASCII special characters.
	for _, r := range msg {
		if isEmoji(r) || isSpecialChar(r) {
			pass.Report(analysis.Diagnostic{
				Pos:     litPos,
				End:     litEnd,
				Message: fmt.Sprintf("log message should not contain special characters or emojis, found %q in %q", string(r), msg),
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

	// Check for repeated punctuation (e.g., "!!!", "...", "???").
	if rep, found := findRepeatedPunctuation(msg); found {
		pass.Report(analysis.Diagnostic{
			Pos:     litPos,
			End:     litEnd,
			Message: fmt.Sprintf("log message should not contain repeated punctuation %q in %q", rep, msg),
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

// findRepeatedPunctuation finds sequences of 2+ identical punctuation characters.
func findRepeatedPunctuation(s string) (string, bool) {
	runes := []rune(s)
	for i := 0; i < len(runes)-1; i++ {
		if !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) && !unicode.IsSpace(runes[i]) {
			count := 1
			for j := i + 1; j < len(runes) && runes[j] == runes[i]; j++ {
				count++
			}
			if count >= 2 {
				return string(runes[i : i+count]), true
			}
		}
	}
	return "", false
}

// cleanRepeatedPunctuation replaces sequences of repeated punctuation with a single instance.
func cleanRepeatedPunctuation(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		b.WriteRune(runes[i])
		if !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) && !unicode.IsSpace(runes[i]) {
			// Skip duplicate punctuation characters.
			for i+1 < len(runes) && runes[i+1] == runes[i] {
				i++
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// isEmoji checks if a rune is an emoji.
func isEmoji(r rune) bool {
	// Common emoji ranges
	return (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
		(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
		(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
		(r >= 0x1F1E0 && r <= 0x1F1FF) || // Flags
		(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
		(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols
		(r >= 0x1FA00 && r <= 0x1FA6F) || // Chess Symbols
		(r >= 0x1FA70 && r <= 0x1FAFF) || // Symbols and Pictographs Extended-A
		(r >= 0x200D && r <= 0x200D) || // Zero width joiner
		(r >= 0x231A && r <= 0x231B) // Watch and Hourglass
}

// isSpecialChar checks if a character is a "special" punctuation beyond what's normally allowed in logs.
func isSpecialChar(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
		return false
	}
	// Allowed basic punctuation in log messages
	allowed := ".,;:'-/?!=()[]{}@#$%^&*_+\\|<>`~\""
	if strings.ContainsRune(allowed, r) {
		return false
	}
	// If it's ASCII printable, it's fine
	if r >= 0x20 && r <= 0x7E {
		return false
	}
	// Everything else is special
	return r > unicode.MaxASCII
}

// cleanSpecialChars removes special characters and emojis from a string.
func cleanSpecialChars(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !isEmoji(r) && !isSpecialChar(r) {
			b.WriteRune(r)
		}
	}
	// Clean up multiple consecutive spaces
	result := b.String()
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	return strings.TrimSpace(result)
}

// checkNoSensitiveData checks that the log message does not contain sensitive data patterns.
// Rule 4: Log messages must not contain potentially sensitive data.
// This checks for string concatenation with variables whose names suggest sensitive data.
func checkNoSensitiveData(pass *analysis.Pass, pos token.Pos, end token.Pos, expr ast.Expr, keywords []string) {
	// Check for string concatenation patterns like "user password: " + password
	if binExpr, ok := expr.(*ast.BinaryExpr); ok && binExpr.Op == token.ADD {
		checkSensitiveConcatenation(pass, pos, end, binExpr, keywords)
		return
	}

	// Check for fmt.Sprintf-like patterns inside the log call arguments
	if call, ok := expr.(*ast.CallExpr); ok {
		checkSensitiveFmtCall(pass, pos, end, call, keywords)
	}
}

func checkSensitiveConcatenation(pass *analysis.Pass, pos token.Pos, end token.Pos, expr *ast.BinaryExpr, keywords []string) {
	// Check if any identifier in the binary expression has a sensitive name
	ast.Inspect(expr, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			name := strings.ToLower(ident.Name)
			for _, keyword := range keywords {
				if strings.Contains(name, keyword) {
					pass.Report(analysis.Diagnostic{
						Pos:     pos,
						End:     end,
						Message: fmt.Sprintf("log message may contain sensitive data: variable %q matches sensitive keyword %q", ident.Name, keyword),
					})
					return false
				}
			}
		}
		return true
	})
}

func checkSensitiveFmtCall(pass *analysis.Pass, pos token.Pos, end token.Pos, call *ast.CallExpr, keywords []string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "fmt" {
		return
	}

	funcName := sel.Sel.Name

	// Determine which arguments to check based on the fmt function.
	var argsToCheck []ast.Expr
	switch funcName {
	case "Sprintf", "Fprintf":
		// First argument is the format string, check the rest.
		if len(call.Args) > 1 {
			argsToCheck = call.Args[1:]
		}
	case "Sprint", "Sprintln":
		// All arguments are values — check all of them.
		argsToCheck = call.Args
	default:
		return
	}

	for _, arg := range argsToCheck {
		ast.Inspect(arg, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				name := strings.ToLower(id.Name)
				for _, keyword := range keywords {
					if strings.Contains(name, keyword) {
						pass.Report(analysis.Diagnostic{
							Pos:     pos,
							End:     end,
							Message: fmt.Sprintf("log message may contain sensitive data: variable %q matches sensitive keyword %q", id.Name, keyword),
						})
						return false
					}
				}
			}
			return true
		})
	}
}
