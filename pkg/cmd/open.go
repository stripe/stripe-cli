package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/open"
)

var nameURLmap = map[string]string{
	"api":                                "https://stripe.com/docs/api",
	"apiref":                             "https://stripe.com/docs/api",
	"dashboard":                          "https://dashboard.stripe.com/test",
	"dashboard/apikeys":                  "https://dashboard.stripe.com/test/apikeys",
	"dashboard/atlas":                    "https://dashboard.stripe.com/test/atlas",
	"dashboard/balance":                  "https://dashboard.stripe.com/test/balance/overview",
	"dashboard/billing":                  "https://dashboard.stripe.com/test/billing",
	"dashboard/connect":                  "https://dashboard.stripe.com/test/connect/overview",
	"dashboard/connect/accounts":         "https://dashboard.stripe.com/test/connect/accounts/overview",
	"dashboard/connect/collected-fees":   "https://dashboard.stripe.com/test/connect/application_fees",
	"dashboard/connect/transfers":        "https://dashboard.stripe.com/test/connect/transfers",
	"dashboard/coupons":                  "https://dashboard.stripe.com/test/coupons",
	"dashboard/customers":                "https://dashboard.stripe.com/test/customers",
	"dashboard/developers":               "https://dashboard.stripe.com/test/developers",
	"dashboard/disputes":                 "https://dashboard.stripe.com/test/disputes",
	"dashboard/events":                   "https://dashboard.stripe.com/test/events",
	"dashboard/invoices":                 "https://dashboard.stripe.com/test/invoices",
	"dashboard/logs":                     "https://dashboard.stripe.com/test/logs",
	"dashboard/orders":                   "https://dashboard.stripe.com/test/orders",
	"dashboard/orders/products":          "https://dashboard.stripe.com/test/orders/products",
	"dashboard/payments":                 "https://dashboard.stripe.com/test/payments",
	"dashboard/payouts":                  "https://dashboard.stripe.com/test/payouts",
	"dashboard/radar":                    "https://dashboard.stripe.com/test/radar",
	"dashboard/radar/list":               "https://dashboard.stripe.com/test/radar/list",
	"dashboard/radar/reviews":            "https://dashboard.stripe.com/test/radar/reviews",
	"dashboard/radar/rules":              "https://dashboard.stripe.com/test/radar/rules",
	"dashboard/settings":                 "https://dashboard.stripe.com/test/settings",
	"dashboard/subscriptions":            "https://dashboard.stripe.com/test/subscriptions",
	"dashboard/subscriptions/products":   "https://dashboard.stripe.com/test/subscriptions/products",
	"dashboard/tax-rates":                "https://dashboard.stripe.com/test/tax-rates",
	"dashboard/terminal":                 "https://dashboard.stripe.com/test/terminal",
	"dashboard/terminal/hardware_orders": "https://dashboard.stripe.com/test/terminal/hardware_orders",
	"dashboard/terminal/locations":       "https://dashboard.stripe.com/test/terminal/locations",
	"dashboard/topups":                   "https://dashboard.stripe.com/test/topups",
	"dashboard/transactions":             "https://dashboard.stripe.com/test/balance",
	"dashboard/webhooks":                 "https://dashboard.stripe.com/test/webhooks",
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
		Short:     "Quickly open Stripe pages",
		RunE:      oc.runOpenCmd,
	}

	oc.cmd.Flags().Bool("list", false, "List all supported short cuts")

	return oc
}

func (oc *openCmd) runOpenCmd(cmd *cobra.Command, args []string) error {
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return err
	}

	if list {
		shortcuts := openNames()
		sort.Strings(shortcuts)
		longest := getLongestShortcut(shortcuts)

		for _, shortcut := range shortcuts {
			paddedName := padName(shortcut, longest)
			fmt.Println(fmt.Sprintf("%s => %s", paddedName, nameURLmap[shortcut]))
		}

		return nil
	}

	if url, ok := nameURLmap[args[0]]; ok {
		err := open.Browser(url)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unsupported open command, given: %s", args[0])
	}

	return nil
}
