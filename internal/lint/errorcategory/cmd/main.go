package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/stripe/stripe-cli/internal/lint/errorcategory"
)

func main() {
	singlechecker.Main(errorcategory.Analyzer)
}
