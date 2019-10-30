// This file is generated; DO NOT EDIT.

package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
)

func addAllResourcesCmds(rootCmd *cobra.Command) {
	// Namespace commands
	_ = resource.NewNamespaceCmd(rootCmd, ""){{ range $nsName, $nsData := .Namespaces }}{{ if $nsData.Resources }}{{ if ne $nsName "" }}
	ns{{ $nsName | ToCamel }}Cmd := resource.NewNamespaceCmd(rootCmd, "{{ $nsName }}"){{ end }}{{ end }}{{ end }}

	// Resource commands{{ range $nsName, $nsData := .Namespaces }}{{ range $resName, $resData := $nsData.Resources }}{{ if $resData.Operations }}
	r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd := resource.NewResourceCmd({{ if ne $nsName "" }}ns{{ $nsName | ToCamel }}Cmd.Cmd{{ else }}rootCmd{{ end }}, "{{ $resName }}"){{ end }}{{ end }}
	{{ end }}

	// Operation commands{{ range $nsName, $nsData := .Namespaces }}{{ range $resName, $resData := $nsData.Resources }}{{ range $opName, $opData := $resData.Operations }}
	resource.NewOperationCmd(r{{ (printf "%s_%s" $nsName $resName) | ToCamel }}Cmd.Cmd, "{{ $opName }}", "{{ $opData.Path }}", http.Method{{ $opData.HTTPVerb | ToCamel }}, map[string]string{ {{range $prop, $propType := $opData.PropFlags }}
		"{{ $prop }}": "{{ $propType }}",{{ end }}
	}, &Config){{ end }}{{ end }}{{ end }}
}
