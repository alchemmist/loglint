package main

import (
	"github.com/alchemmist/loglint/pkg/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var runSinglechecker = singlechecker.Main

func main() {
	runSinglechecker(analyzer.NewAnalyzer())
}
