package importalias_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	importalias "github.com/podhmo/go-importalias"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "dup")
}
