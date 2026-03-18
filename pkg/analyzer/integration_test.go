package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/alchemmist/loglint/pkg/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestIntegrationSlog(t *testing.T) {
	analysistest.Run(t, testdataDir(t), analyzer.NewAnalyzer(), "example")
}

func TestIntegrationZap(t *testing.T) {
	analysistest.Run(t, testdataDir(t), analyzer.NewAnalyzer(), "zap_example")
}

func testdataDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}
	return path
}
