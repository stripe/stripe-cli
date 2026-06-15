package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

func newPostCmd(isPreview bool) *requests.Base {
	reqs := &requests.Base{
		Method:           http.MethodPost,
		Profile:          &Config.Profile,
		IsPreviewCommand: isPreview,
	}

	preview := ""
	verb := "POST"
	if isPreview {
		preview = "preview "
		verb = "preview POST"
	}

	previewNote := ""
	if isPreview {
		previewNote = i18n.T("post.preview_note")
	}

	reqs.Cmd = &cobra.Command{
		Use:     "post <path>",
		Args:    validators.ExactArgs(1),
		Short:   i18n.Tf("post.short", i18n.Args{"verb": verb}),
		Long:    i18n.Tf("post.long", i18n.Args{"verb": verb, "preview_note": previewNote}),
		Example: i18n.Tf("post.example", i18n.Args{"preview": preview}),
		RunE:    reqs.RunRequestsCmd,
	}

	reqs.InitFlags()

	return reqs
}
