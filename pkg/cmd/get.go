package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

func newGetCmd(isPreview bool) *requests.Base {
	reqs := &requests.Base{
		Method:           http.MethodGet,
		Profile:          &Config.Profile,
		IsPreviewCommand: isPreview,
	}

	preview := ""
	verb := "GET"
	if isPreview {
		preview = "preview "
		verb = "preview GET"
	}

	previewNote := ""
	if isPreview {
		previewNote = i18n.T("get.preview_note")
	}

	reqs.Cmd = &cobra.Command{
		Use:     "get <id or path>",
		Args:    validators.ExactArgs(1),
		Short:   i18n.Tf("get.short", i18n.Args{"verb": verb}),
		Long:    i18n.Tf("get.long", i18n.Args{"verb_upper": verb, "preview_note": previewNote}),
		Example: i18n.Tf("get.example", i18n.Args{"preview": preview}),
		RunE:    reqs.RunRequestsCmd,
	}

	reqs.InitFlags()

	return reqs
}
