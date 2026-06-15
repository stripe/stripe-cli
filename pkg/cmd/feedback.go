package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const feedbackAsciiArt = `
     _        _
 ___| |_ _ __(_)_ __   ___
/ __| __| '__| | '_ \ / _ \
\__ \ |_| |  | | |_) |  __/
|___/\__|_|  |_| .__/ \___|
               |_|
`

type feedbackCmd struct {
	cmd *cobra.Command
}

func newFeedbackdCmd() *feedbackCmd {
	return &feedbackCmd{
		cmd: &cobra.Command{
			Use:   "feedback",
			Args:  validators.NoArgs,
			Short: i18n.T("feedback.short"),
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Print(feedbackAsciiArt)
				fmt.Println(i18n.T("feedback.output.body"))
			},
		},
	}
}
