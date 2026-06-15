package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

func newDeleteCmd(isPreview bool) *requests.Base {
	reqs := &requests.Base{
		Method:           http.MethodDelete,
		Profile:          &Config.Profile,
		IsPreviewCommand: isPreview,
	}

	preview := ""
	verb := "DELETE"
	if isPreview {
		preview = "preview "
		verb = "preview DELETE"
	}

	previewNote := ""
	if isPreview {
		previewNote = i18n.T("delete.preview_note")
	}

	reqs.Cmd = &cobra.Command{
		Use:     "delete <path>",
		Args:    validators.ExactArgs(1),
		Short:   i18n.Tf("delete.short", i18n.Args{"verb": verb}),
		Long:    i18n.Tf("delete.long", i18n.Args{"verb": verb, "preview_note": previewNote}),
		Example: i18n.Tf("delete.example", i18n.Args{"preview": preview}),
		RunE:    reqs.RunRequestsCmd,
	}

	reqs.InitFlags()

	return reqs
}
