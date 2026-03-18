package analyzer_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alchemmist/loglint/pkg/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

//nolint:paralleltest
func TestIntegrationConfigJSON(t *testing.T) {
	t.Chdir(filepath.Join(testdataDir(t), "src", "config_json"))
	analysistest.Run(t, testdataDir(t), analyzer.NewAnalyzer(), "config_json")
}

//nolint:paralleltest
func TestIntegrationConfigYAML(t *testing.T) {
	t.Chdir(filepath.Join(testdataDir(t), "src", "config_yaml"))
	analysistest.Run(t, testdataDir(t), analyzer.NewAnalyzer(), "config_yaml")
}

func testdataDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}

	path, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", "testdata"))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}

	return path
}

// Intentionally no t.Parallel: tests use t.Chdir which changes process CWD.
