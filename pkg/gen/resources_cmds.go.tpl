// This file is generated; DO NOT EDIT.

package resources

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/config"
)

// AddAllResourcesCmds registers all Stripe API resource commands on rootCmd.
func AddAllResourcesCmds(rootCmd *cobra.Command, cfg *config.Config) {
	v1root := rootCmd
	v2root := resource.NewNamespaceCmd(rootCmd, "v2").Cmd
	previewRoot := resource.NewNamespaceCmd(rootCmd, "preview").Cmd
	previewV2Root := resource.NewNamespaceCmd(previewRoot, "v2").Cmd

	addV1ResourcesCmds(v1root, cfg)
	addV2ResourcesCmds(v2root, cfg)

	{{- if index .ApiNamespaces "v1-preview" }}
	addV1PreviewResourcesCmds(previewRoot, cfg)
	{{- end }}
	addV2PreviewResourcesCmds(previewV2Root, cfg)
}

{{ range $apiNamespace, $vData := .ApiNamespaces }}
func add{{ $apiNamespace | ToCamel }}ResourcesCmds(rootCmd *cobra.Command, cfg *config.Config) {
	// Namespace commands
	_ = resource.NewNamespaceCmd(rootCmd, ""){{ range $nsName, $nsData := $vData.Namespaces }}{{ if $nsData.Resources }}{{ if ne $nsName "" }}
	ns{{ $nsName | ToCamel }}Cmd := resource.NewNamespaceCmd(rootCmd, "{{ $nsName }}"){{ end }}{{ end }}{{ end }}

	// Resource commands{{ range $nsName, $nsData := $vData.Namespaces }}{{ range $resName, $resData := $nsData.Resources }}{{ if eq $resData.SubResources nil }}
	r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd := resource.NewResourceCmd({{ if ne $nsName "" }}ns{{ $nsName | ToCamel }}Cmd.Cmd{{ else }}rootCmd{{ end }}, "{{ $resName }}"){{ else }}
	r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd := resource.NewResourceCmd({{ if ne $nsName "" }}ns{{ $nsName | ToCamel }}Cmd.Cmd{{ else }}rootCmd{{ end }}, "{{ $resName }}"){{ range $subResName, $subResData := $resData.SubResources }}{{ if $subResData.Operations }}
	r{{ (printf "%s_%s_%s" $nsName $resName $subResName) | ToCamel }}Cmd := resource.NewResourceCmd(r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd.Cmd, "{{ $subResName }}"){{ end }}{{ end }}{{ end }}{{ end }}{{ end }}

	// Operation commands{{ range $nsName, $nsData := $vData.Namespaces }}{{ range $resName, $resData := $nsData.Resources }}{{ range $opName, $opData := $resData.Operations }}
	resource.NewOperationCmd(r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd.Cmd, &{{ $opData.VarName }}, cfg){{ end }}{{ range $subResName, $subResData := $resData.SubResources }}{{range $opName, $opData := $subResData.Operations }}
	resource.NewOperationCmd(r{{ (printf "%s_%s_%s" $nsName $resName $subResName) | ToCamel }}Cmd.Cmd, &{{ $opData.VarName }}, cfg){{ end }}{{ end }}{{ end }}{{ end }}
}
{{ end }}
