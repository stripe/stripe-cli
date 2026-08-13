package errorcategory_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	categorylint "github.com/stripe/stripe-cli/internal/lint/errorcategory"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, categorylint.Analyzer, "a", "generated", "testfiles", "github.com/stripe/stripe-cli/pkg/errorcategory")
}
