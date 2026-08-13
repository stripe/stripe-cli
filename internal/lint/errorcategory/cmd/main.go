package main

import (
	"github.com/stripe/stripe-cli/internal/lint/errorcategory"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(errorcategory.Analyzer)
}
