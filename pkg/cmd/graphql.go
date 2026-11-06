package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sort"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/graphqlexp"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// EXPERIMENT: throwaway command for probing graphql.stripe.com with a genqlient client.

// graphqlStripeVersion is hard-coded for now.
const graphqlStripeVersion = "unsafe-development"

type graphqlCmd struct {
	cmd  *cobra.Command
	cfg  *config.Config
	host string
	path string
	live bool
	dump bool
}

type authTransport struct {
	creds stripe.Credentials
	base  http.RoundTripper
	dump  io.Writer // when set, the request (as curl) and raw response are written here
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	t.creds.SetRequestHeaders(req)
	req.Header.Set("Stripe-Version", graphqlStripeVersion)

	var body []byte
	if t.dump != nil && req.Body != nil {
		var err error
		if body, err = io.ReadAll(req.Body); err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	resp, err := t.base.RoundTrip(req)

	if t.dump != nil {
		fmt.Fprintln(t.dump, curlFor(req, body))
		if err == nil {
			if raw, dumpErr := httputil.DumpResponse(resp, true); dumpErr == nil {
				fmt.Fprintf(t.dump, "\n--- response ---\n%s\n", raw)
			}
		}
	}

	return resp, err
}

// curlFor renders req as a copy-pasteable curl command. It includes the real
// Authorization header, so treat the output as sensitive.
func curlFor(req *http.Request, body []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "curl -i -X %s '%s'", req.Method, req.URL.String())

	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		for _, v := range req.Header[k] {
			fmt.Fprintf(&b, " \\\n  -H '%s: %s'", k, v)
		}
	}
	if len(body) > 0 {
		fmt.Fprintf(&b, " \\\n  --data-raw '%s'", strings.ReplaceAll(string(body), "'", `'\''`))
	}

	return b.String()
}

func newGraphqlCmd(cfg *config.Config) *graphqlCmd {
	gc := &graphqlCmd{cfg: cfg}

	gc.cmd = &cobra.Command{
		Use:    "graphql [name]",
		Args:   validators.MaximumNArgs(1),
		Short:  "Experimental: call the v2 GraphQL gateway",
		Long:   "Runs the inlineAPIVersion query, or inlineGreeting if a name is given, against the GraphQL gateway using your CLI credentials.",
		RunE:   gc.run,
		Hidden: true,
	}
	gc.cmd.Flags().StringVar(&gc.host, "host", "graphql.stripe.com", "GraphQL gateway host")
	gc.cmd.Flags().StringVar(&gc.path, "path", "/graphql", "GraphQL endpoint path (e.g. /graphql/multiregion)")
	gc.cmd.Flags().BoolVar(&gc.live, "live", false, "Use the live mode API key")
	gc.cmd.Flags().BoolVar(&gc.dump, "dump", false, "Print the request as a curl command (includes your token) and the raw response to stderr")

	return gc
}

func (gc *graphqlCmd) run(cmd *cobra.Command, args []string) error {
	// Keep QA credentials pointed at the QA gateway unless --host says otherwise.
	if !cmd.Flags().Changed("host") && rootAccessBaseURL == login.QAAccessBaseURL {
		gc.host = "qa-graphql.stripe.com"
	}

	creds, err := gc.cfg.Profile.ResolveCredentials(gc.live)
	if err != nil {
		return err
	}

	transport := &authTransport{creds: creds, base: http.DefaultTransport}
	if gc.dump {
		transport.dump = cmd.ErrOrStderr()
	}
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	client := graphql.NewClient("https://"+gc.host+gc.path, httpClient)

	if len(args) == 1 {
		resp, err := graphqlexp.Greeting(cmd.Context(), client, args[0])
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), resp.InlineGreeting.Greeting)

		return nil
	}

	resp, err := graphqlexp.APIVersion(cmd.Context(), client)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), resp.InlineAPIVersion.Version)

	return nil
}
