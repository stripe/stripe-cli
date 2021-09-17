package main

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/cmd"
)

func main() {
	cmd.Execute(context.Background())
}
