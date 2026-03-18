package main

import (
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestMainWiresAnalyzer(t *testing.T) {
	t.Parallel()

	original := runSinglechecker

	t.Cleanup(func() { runSinglechecker = original })

	var got *analysis.Analyzer

	runSinglechecker = func(a *analysis.Analyzer) {
		got = a
	}

	main()

	if got == nil {
		t.Fatal("expected analyzer to be passed to singlechecker")
	}

	if got.Name != "loglint" {
		t.Fatalf("unexpected analyzer name: %q", got.Name)
	}
}
