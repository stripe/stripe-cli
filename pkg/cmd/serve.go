package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/spf13/cobra"

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
		Short:   "Serve static files locally",
		Args:    validators.MaximumNArgs(1),
		Example: "stripe serve /path/to/directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}

			fmt.Println("Starting static file server at address", fmt.Sprintf("http://localhost:%s", port))
			http.Handle("/", http.FileServer(http.Dir(dir)))
			err := http.ListenAndServe(fmt.Sprintf(":%s", port), handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))

			return err
		},
	}

	sc.cmd.Flags().StringVar(&port, "port", "4242", "Provide a custom port to serve content from.")

	return sc
}
