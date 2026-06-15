package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
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
		Short:   i18n.T("community.short"),
		Example: "stripe community",
		RunE:    cc.runCommunityCmd,
	}

	return cc
}

func (cc *communityCmd) runCommunityCmd(cmd *cobra.Command, args []string) error {
	if !canOpenBrowser() {
		fmt.Print(i18n.Tf("community.output.no_browser", i18n.Args{"url": communityURL}))
		return nil
	}

	fmt.Print(i18n.Tf("community.output.with_browser", i18n.Args{"url": communityURL}))

	input := os.Stdin
	fmt.Fscanln(input)

	err := openBrowser(communityURL)
	if err != nil {
		fmt.Print(i18n.Tf("community.output.browser_failed", i18n.Args{"url": communityURL}))
	}

	return nil
}
