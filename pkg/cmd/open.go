package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/validators"
)

var nameURLmap = map[string]string{
	"dashboard":                          "https://dashboard.stripe.com/",
	"dashboard/payments":                 "https://dashboard.stripe.com/payments",
	"dashboard/disputes":                 "https://dashboard.stripe.com/disputes",
	"dashboard/balance":                  "https://dashboard.stripe.com/balance/overview",
	"dashboard/payouts":                  "https://dashboard.stripe.com/payouts",
	"dashboard/topups":                   "https://dashboard.stripe.com/topups",
	"dashboard/transactions":             "https://dashboard.stripe.com/balance",
	"dashboard/customers":                "https://dashboard.stripe.com/customers",
	"dashboard/atlas":                    "https://dashboard.stripe.com/atlas",
	"dashboard/radar":                    "https://dashboard.stripe.com/radar",
	"dashboard/radar/reviews":            "https://dashboard.stripe.com/radar/reviews",
	"dashboard/radar/list":               "https://dashboard.stripe.com/radar/list",
	"dashboard/radar/rules":              "https://dashboard.stripe.com/radar/rules",
	"dashboard/billing":                  "https://dashboard.stripe.com/billing",
	"dashboard/invoices":                 "https://dashboard.stripe.com/invoices",
	"dashboard/subscriptions":            "https://dashboard.stripe.com/subscriptions",
	"dashboard/subscriptions/products":   "https://dashboard.stripe.com/subscriptions/products",
	"dashboard/tax-rates":                "https://dashboard.stripe.com/tax-rates",
	"dashboard/coupons":                  "https://dashboard.stripe.com/coupons",
	"dashboard/connect":                  "https://dashboard.stripe.com/connect/overview",
	"dashboard/connect/accounts":         "https://dashboard.stripe.com/connect/accounts/overview",
	"dashboard/connect/transfers":        "https://dashboard.stripe.com/connect/transfers",
	"dashboard/connect/collected-fees":   "https://dashboard.stripe.com/connect/application_fees",
	"dashboard/orders":                   "https://dashboard.stripe.com/orders",
	"dashboard/orders/products":          "https://dashboard.stripe.com/orders/products",
	"dashboard/terminal":                 "https://dashboard.stripe.com/terminal",
	"dashboard/terminal/locations":       "https://dashboard.stripe.com/terminal/locations",
	"dashboard/terminal/hardware_orders": "https://dashboard.stripe.com/terminal/hardware_orders",
	"dashboard/developers":               "https://dashboard.stripe.com/developers",
	"dashboard/apikeys":                  "https://dashboard.stripe.com/apikeys",
	"dashboard/webhooks":                 "https://dashboard.stripe.com/webhooks",
	"dashboard/events":                   "https://dashboard.stripe.com/events",
	"dashboard/logs":                     "https://dashboard.stripe.com/logs",
	"dashboard/settings":                 "https://dashboard.stripe.com/settings",
	"api":                                "https://stripe.com/docs/api",
	"apiref":                             "https://stripe.com/docs/api",
	"docs":                               "https://stripe.com/docs",
}

func openNames() []string {
	keys := make([]string, 0, len(nameURLmap))
	for k := range nameURLmap {
		keys = append(keys, k)
	}

	return keys
}

type openCmd struct {
	cmd *cobra.Command
}

func newOpenCmd() *openCmd {
	oc := &openCmd{}
	oc.cmd = &cobra.Command{
		Use:       "open",
		Args:      validators.ExactArgs(1),
		ValidArgs: openNames(),
		Short:     "Quickly open Stripe pages",
		RunE:      oc.runOpenCmd,
	}

	return oc
}

func (oc *openCmd) runOpenCmd(cmd *cobra.Command, args []string) error {
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
