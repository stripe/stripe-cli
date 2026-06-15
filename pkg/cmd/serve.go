package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/handlers"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type serveCmd struct {
	cmd *cobra.Command
}

func newServeCmd() *serveCmd {
	var port string

	sc := &serveCmd{}

	sc.cmd = &cobra.Command{
		Use:     "serve",
		Aliases: []string{"srv"},
		Short:   i18n.T("serve.short"),
		Args:    validators.MaximumNArgs(1),
		Example: i18n.T("serve.example"),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}

			absoluteDir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}

			fmt.Print(i18n.Tf("serve.output.starting_server", i18n.Args{"dir": absoluteDir}))

			fmt.Println(i18n.Tf("serve.output.server_address", i18n.Args{"port": port}))
			http.Handle("/", http.FileServer(http.Dir(absoluteDir)))
			return http.ListenAndServe(fmt.Sprintf("localhost:%s", port), handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))
		},
	}

	sc.cmd.Flags().StringVar(&port, "port", "4242", i18n.T("serve.flags.port"))

	return sc
}
