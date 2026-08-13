package errorcategory_test

import (
	"testing"

	"github.com/stripe/stripe-cli/internal/lint/errorcategory"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errorcategory.Analyzer, "example.com/origins")
}
