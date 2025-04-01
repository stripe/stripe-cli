// This file is generated; DO NOT EDIT.

package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/spec"
)

func addAllResourcesCmds(rootCmd *cobra.Command) {
	{{ range $specVersion, $vData := .Versions }}{{ if eq $specVersion "v1" }}v1root := rootCmd{{ else if eq $specVersion "v2-preview" }}
	previewRoot := resource.NewNamespaceCmd(rootCmd, "preview").Cmd
	previewV2Root := resource.NewNamespaceCmd(previewRoot, "v2").Cmd{{ else }}
	{{ $specVersion }}root := resource.NewNamespaceCmd(rootCmd, "{{ $specVersion }}").Cmd{{ end }}
	add{{ $specVersion | ToCamel }}ResourcesCmds({{ if eq $specVersion "v2-preview" }}previewV2Root{{ else }}{{ $specVersion }}root{{ end }}){{ end }}
}

{{ range $specVersion, $vData := .Versions }}
func add{{ $specVersion | ToCamel }}ResourcesCmds(rootCmd *cobra.Command) {
	// Namespace commands
	_ = resource.NewNamespaceCmd(rootCmd, ""){{ range $nsName, $nsData := $vData.Namespaces }}{{ if $nsData.Resources }}{{ if ne $nsName "" }}
	ns{{ $nsName | ToCamel }}Cmd := resource.NewNamespaceCmd(rootCmd, "{{ $nsName }}"){{ end }}{{ end }}{{ end }}

	// Resource commands{{ range $nsName, $nsData := $vData.Namespaces }}{{ range $resName, $resData := $nsData.Resources }}{{ if eq $resData.SubResources nil }}
	r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd := resource.NewResourceCmd({{ if ne $nsName "" }}ns{{ $nsName | ToCamel }}Cmd.Cmd{{ else }}rootCmd{{ end }}, "{{ $resName }}"){{ else }}
	r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd := resource.NewResourceCmd({{ if ne $nsName "" }}ns{{ $nsName | ToCamel }}Cmd.Cmd{{ else }}rootCmd{{ end }}, "{{ $resName }}"){{ range $subResName, $subResData := $resData.SubResources }}{{ if $subResData.Operations }}
	r{{ (printf "%s_%s_%s" $nsName $resName $subResName) | ToCamel }}Cmd := resource.NewResourceCmd(r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd.Cmd, "{{ $subResName }}"){{ end }}{{ end }}{{ end }}{{ end }}{{ end }}

	// Operation commands{{ range $nsName, $nsData := $vData.Namespaces }}{{ range $resName, $resData := $nsData.Resources }}{{ range $opName, $opData := $resData.Operations }}
	resource.NewOperationCmd(r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd.Cmd, "{{ $opName }}", "{{ $opData.Path }}", http.Method{{ $opData.HTTPVerb | ToCamel }}, map[string]string{ {{range $prop, $propType := $opData.PropFlags }}
		"{{ $prop }}": "{{ $propType }}",{{ end }}
	}, map[string][]spec.StripeEnumValue{ {{range $prop, $enumValues := $opData.EnumFlags }}
		"{{ $prop }}": {{ printf "%#v" $enumValues }},{{ end }}
	}, &Config, {{ if eq $specVersion "v2-preview" }}true{{ else }}false{{ end }}){{ end }}{{ range $subResName, $subResData := $resData.SubResources }}{{range $opName, $opData := $subResData.Operations }}
	resource.NewOperationCmd(r{{ (printf "%s_%s_%s" $nsName $resName $subResName) | ToCamel }}Cmd.Cmd, "{{ $opName }}", "{{ $opData.Path }}", http.Method{{ $opData.HTTPVerb | ToCamel }}, map[string]string{ {{range $prop, $propType := $opData.PropFlags }}
		"{{ $prop }}": "{{ $propType }}",{{ end }}
	}, map[string][]spec.StripeEnumValue{ {{range $prop, $enumValues := $opData.EnumFlags }}
		"{{ $prop }}": {{ printf "%#v" $enumValues }},{{ end }}
	}, &Config, {{ if eq $specVersion "v2-preview" }}true{{ else }}false{{ end }}){{ end }}{{ end }}{{ end }}{{ end }}
}
{{ end }}
