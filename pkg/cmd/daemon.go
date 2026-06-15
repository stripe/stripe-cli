package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/rpcservice"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type daemonCmd struct {
	cmd  *cobra.Command
	port int
	cfg  *config.Config
}

func newDaemonCmd(cfg *config.Config) *daemonCmd {
	dc := &daemonCmd{
		cfg: cfg,
	}

	dc.cmd = &cobra.Command{
		Use:    "daemon",
		Args:   validators.NoArgs,
		Short:  i18n.T("daemon.short"),
		Long:   i18n.T("daemon.long"),
		Run:    dc.runDaemonCmd,
		Hidden: true,
	}
	dc.cmd.Flags().IntVar(&dc.port, "port", 0, i18n.T("daemon.flags.port"))

	return dc
}

func (dc *daemonCmd) runDaemonCmd(cmd *cobra.Command, args []string) {
	telemetryClient := stripe.GetTelemetryClient(cmd.Context())
	srv := rpcservice.New(&rpcservice.Config{
		Port:    dc.port,
		Log:     log.StandardLogger(),
		UserCfg: dc.cfg,
	}, telemetryClient)

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.daemonCmd.runDaemonCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	go srv.Run(ctx)

	<-ctx.Done()
}
