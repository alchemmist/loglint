package main

import "testing"

func TestGetAnalyzers(t *testing.T) {
	t.Parallel()

	analyzers := AnalyzerPlugin.GetAnalyzers()
	if len(analyzers) != 1 {
		t.Fatalf("expected 1 analyzer, got %d", len(analyzers))
	}

	if analyzers[0].Name != "loglint" {
		t.Fatalf("unexpected analyzer name: %q", analyzers[0].Name)
	}
}
