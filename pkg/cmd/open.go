package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/version"
)

var nameURLmap = map[string]string{
	"api":                                "https://stripe.com/docs/api",
	"apiref":                             "https://stripe.com/docs/api",
	"cliref":                             "https://stripe.com/docs/cli",
	"dashboard":                          "https://dashboard.stripe.com%s",
	"dashboard/apikeys":                  "https://dashboard.stripe.com%s/apikeys",
	"dashboard/atlas":                    "https://dashboard.stripe.com%s/atlas",
	"dashboard/balance":                  "https://dashboard.stripe.com%s/balance/overview",
	"dashboard/billing":                  "https://dashboard.stripe.com%s/billing",
	"dashboard/connect":                  "https://dashboard.stripe.com%s/connect/overview",
	"dashboard/connect/accounts":         "https://dashboard.stripe.com%s/connect/accounts/overview",
	"dashboard/connect/collected-fees":   "https://dashboard.stripe.com%s/connect/application_fees",
	"dashboard/connect/transfers":        "https://dashboard.stripe.com%s/connect/transfers",
	"dashboard/coupons":                  "https://dashboard.stripe.com%s/coupons",
	"dashboard/customers":                "https://dashboard.stripe.com%s/customers",
	"dashboard/developers":               "https://dashboard.stripe.com%s/developers",
	"dashboard/disputes":                 "https://dashboard.stripe.com%s/disputes",
	"dashboard/events":                   "https://dashboard.stripe.com%s/events",
	"dashboard/invoices":                 "https://dashboard.stripe.com%s/invoices",
	"dashboard/logs":                     "https://dashboard.stripe.com%s/logs",
	"dashboard/orders":                   "https://dashboard.stripe.com%s/orders",
	"dashboard/orders/products":          "https://dashboard.stripe.com%s/orders/products",
	"dashboard/payments":                 "https://dashboard.stripe.com%s/payments",
	"dashboard/payouts":                  "https://dashboard.stripe.com%s/payouts",
	"dashboard/radar":                    "https://dashboard.stripe.com%s/radar",
	"dashboard/radar/list":               "https://dashboard.stripe.com%s/radar/list",
	"dashboard/radar/reviews":            "https://dashboard.stripe.com%s/radar/reviews",
	"dashboard/radar/rules":              "https://dashboard.stripe.com%s/radar/rules",
	"dashboard/settings":                 "https://dashboard.stripe.com%s/settings",
	"dashboard/subscriptions":            "https://dashboard.stripe.com%s/subscriptions",
	"dashboard/subscriptions/products":   "https://dashboard.stripe.com%s/subscriptions/products",
	"dashboard/tax-rates":                "https://dashboard.stripe.com%s/tax-rates",
	"dashboard/terminal":                 "https://dashboard.stripe.com%s/terminal",
	"dashboard/terminal/hardware_orders": "https://dashboard.stripe.com%s/terminal/hardware_orders",
	"dashboard/terminal/locations":       "https://dashboard.stripe.com%s/terminal/locations",
	"dashboard/topups":                   "https://dashboard.stripe.com%s/topups",
	"dashboard/transactions":             "https://dashboard.stripe.com%s/balance",
	"dashboard/webhooks":                 "https://dashboard.stripe.com%s/webhooks",
	"docs":                               "https://stripe.com/docs",
}

func openNames() []string {
	keys := make([]string, 0, len(nameURLmap))
	for k := range nameURLmap {
		keys = append(keys, k)
	}

	return keys
}

func getLongestShortcut(shortcuts []string) int {
	longest := 0
	for _, s := range shortcuts {
		if len(s) > longest {
			longest = len(s)
		}
	}

	return longest
}

func padName(name string, length int) string {
	difference := length - len(name)

	var b strings.Builder

	fmt.Fprint(&b, name)

	for i := 0; i < difference; i++ {
		fmt.Fprint(&b, " ")
	}

	return b.String()
}

type openCmd struct {
	cmd *cobra.Command
}

func newOpenCmd() *openCmd {
	oc := &openCmd{}
	oc.cmd = &cobra.Command{
		Use:       "open",
		ValidArgs: openNames(),
		Short:     i18n.T("open.short"),
		Long:      i18n.T("open.long"),
		Example:   i18n.T("open.example"),
		RunE:      oc.runOpenCmd,
	}

	oc.cmd.Flags().Bool("list", false, i18n.T("open.flags.list"))
	oc.cmd.Flags().Bool("live", false, i18n.T("open.flags.live"))

	return oc
}

func (oc *openCmd) runOpenCmd(cmd *cobra.Command, args []string) error {
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return err
	}

	livemode, err := cmd.Flags().GetBool("live")
	if err != nil {
		return err
	}

	if list || len(args) == 0 {
		fmt.Println(i18n.T("open.output.intro"))
		fmt.Println(i18n.T("open.output.shortcuts_header"))
		fmt.Println()

		shortcuts := openNames()
		sort.Strings(shortcuts)

		longest := getLongestShortcut(shortcuts)

		fmt.Printf("%s%s\n", padName(i18n.T("open.output.column_shortcut"), longest), "    "+i18n.T("open.output.column_url"))
		fmt.Printf("%s%s\n", padName("--------", longest), "    ---------")

		for _, shortcut := range shortcuts {
			maybeTestMode := ""
			if !livemode {
				maybeTestMode = "/test"
			}

			url := nameURLmap[shortcut]
			if strings.Contains(url, "%s") {
				url = fmt.Sprintf(url, maybeTestMode)
			}

			paddedName := padName(shortcut, longest)
			fmt.Printf("%s => %s\n", paddedName, url)
		}

		return nil
	}

	version.CheckLatestVersion()

	if url, ok := nameURLmap[args[0]]; ok {
		livemode, err := cmd.Flags().GetBool("live")
		if err != nil {
			return err
		}

		maybeTestMode := ""
		if !livemode {
			maybeTestMode = "/test"
		}

		if strings.Contains(url, "%s") {
			err = open.Browser(fmt.Sprintf(url, maybeTestMode))
		} else {
			err = open.Browser(url)
		}

		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%s", i18n.Tf("open.errors.unsupported_command", i18n.Args{"command": args[0]}))
	}

	return nil
}
