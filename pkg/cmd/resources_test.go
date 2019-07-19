package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/spec"
)

func BenchmarkAddAllResourceCmds(b *testing.B) {
	cmd := &cobra.Command{
		Annotations: make(map[string]string),
	}
	stripeAPI, _ := spec.LoadSpec("")

	for n := 0; n < b.N; n++ {
		addAllResourceCmds(cmd, stripeAPI)
	}
}
