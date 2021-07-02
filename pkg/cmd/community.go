package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/open"
)

const communityURL = "https://stripe.com/go/developer-chat"

var openBrowser = open.Browser
var canOpenBrowser = open.CanOpenBrowser

type communityCmd struct {
	cmd *cobra.Command
}

func newCommunityCmd() *communityCmd {
	cc := &communityCmd{}

	cc.cmd = &cobra.Command{
		Use:     "community",
		Aliases: []string{"discord", "chat"},
		Short:   "Chat with Stripe engineers and other developers",
		Example: "stripe community",
		RunE:    cc.runCommunityCmd,
	}

	return cc
}

func (cc *communityCmd) runCommunityCmd(cmd *cobra.Command, args []string) error {
	if !canOpenBrowser() {
		fmt.Printf("Chat with other developers and Stripe engineers in the official Stripe Discord server: %s\n", communityURL)
		return nil
	}

	fmt.Printf("Chat with other developers and Stripe engineers in the official Stripe Discord server.\n\nPress Enter to open the browser or visit %s", communityURL)

	input := os.Stdin
	fmt.Fscanln(input)

	err := openBrowser(communityURL)
	if err != nil {
		fmt.Printf("Failed to open browser, please go to %s manually.", communityURL)
	}

	return nil
}
