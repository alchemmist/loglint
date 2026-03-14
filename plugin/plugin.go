// Package plugin provides the golangci-lint plugin interface for loglint.
//
// To use loglint as a golangci-lint plugin, build it as a Go plugin:
//
//	go build -buildmode=plugin -o loglint.so ./plugin/
//
// Then configure golangci-lint to use the plugin in .golangci.yml:
//
//	linters-settings:
//	  custom:
//	    loglint:
//	      path: ./loglint.so
//	      description: Linter for checking log messages
//	      original-url: github.com/alchemmist/loglint
package main

//nolint:depguard
import (
	"github.com/alchemmist/loglint/pkg/analyzer"
	"golang.org/x/tools/go/analysis"
)

// AnalyzerPlugin provides the analyzer to golangci-lint.
// This variable name is required by the golangci-lint plugin interface.
var AnalyzerPlugin analyzerPlugin //nolint:gochecknoglobals

type analyzerPlugin struct{}

// GetAnalyzers returns the list of analyzers provided by this plugin.
func (*analyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		analyzer.Analyzer,
	}
}
