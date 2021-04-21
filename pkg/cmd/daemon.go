package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"context"

	"github.com/stripe/stripe-cli/pkg/rpcserver"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type daemonCmd struct {
	cmd  *cobra.Command
	port int
}

func newDaemonCmd() *daemonCmd {
	dc := &daemonCmd{}

	dc.cmd = &cobra.Command{
		Use:   "daemon",
		Args:  validators.NoArgs,
		Short: "Run as a daemon on your localhost",
		Long: `Start a local gRPC server, enabling you to invoke Stripe CLI commands programmatically from a gRPC
client.

Currently, stripe daemon only supports a subset of CLI commands. Documentation is not yet available.`,
		RunE:   dc.runDaemonCmd,
		Hidden: true,
	}
	dc.cmd.Flags().IntVar(&dc.port, "port", 0, "The TCP port the daemon will listen to (default: an available port)")

	return dc
}

func (dc *daemonCmd) runDaemonCmd(cmd *cobra.Command, args []string) error {
	srv := rpcserver.New(&rpcserver.Config{
		Port: dc.port,
		Log:  log.StandardLogger(),
	})

	err := srv.Run(context.Background())
	if err != nil {
		return err
	}

	return nil
}
