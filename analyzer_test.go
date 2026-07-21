package importalias_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	importalias "github.com/podhmo/go-importalias"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "dup")
}

// TestAnalyzer_SkipsGeneratedByDefault covers FR-5.7 / DEC-2.4: with the
// default skip_generated=true, the generated file's minority mapping is not
// counted, so the package is consistent and nothing is reported.
func TestAnalyzer_SkipsGeneratedByDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "gendefault")
}

// TestAnalyzer_IncludesGeneratedWhenDisabled covers the -skip_generated=false
// path: the generated file is analyzed and its minority mapping is reported.
func TestAnalyzer_IncludesGeneratedWhenDisabled(t *testing.T) {
	if err := importalias.Analyzer.Flags.Set("skip_generated", "false"); err != nil {
		t.Fatalf("set skip_generated=false: %v", err)
	}
	t.Cleanup(func() {
		if err := importalias.Analyzer.Flags.Set("skip_generated", "true"); err != nil {
			t.Fatalf("reset skip_generated=true: %v", err)
		}
	})
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "genreported")
}
