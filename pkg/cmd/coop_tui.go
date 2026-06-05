package cmd

import (
	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/tui"
)

func runCoopTUI(store *coop.Store, sessionID string) error {
	return tui.Run(store, sessionID)
}
