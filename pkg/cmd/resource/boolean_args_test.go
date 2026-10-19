package resource

import (
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestNormalizeBooleanRequestArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string // nil means the invocation must remain unchanged
	}{
		{"true", []string{"id", "--disabled", "true"}, []string{"id", "--disabled=true"}},
		{"false", []string{"id", "--disabled", "false"}, []string{"id", "--disabled=false"}},
		{"before ID", []string{"--disabled", "false", "id"}, []string{"--disabled=false", "id"}},
		{"bare", []string{"id", "--disabled"}, nil},
		{"equals", []string{"id", "--disabled=false"}, nil},
		{"missing ID", []string{"--disabled", "false"}, []string{"--disabled=false"}},
		{"boolean positional before flag", []string{"true", "--disabled"}, nil},
		{"positional equals false", []string{"false", "--disabled", "false"}, []string{"false", "--disabled=false"}},
		{"nested", []string{"id", "--settings.enabled", "false"}, []string{"id", "--settings.enabled=false"}},
		{"nested brackets", []string{"id", "--settings[enabled]", "false"}, []string{"id", "--settings[enabled]=false"}},
		{"multiple", []string{"id", "--disabled", "true", "--settings.enabled", "false"}, []string{"id", "--disabled=true", "--settings.enabled=false"}},
		{"repeated", []string{"id", "--disabled", "false", "--disabled"}, []string{"id", "--disabled=false", "--disabled"}},
		{"equals followed by positional", []string{"--disabled=true", "false"}, nil},
		{"unknown flag", []string{"id", "--unknown", "--disabled", "false"}, nil},
		{"invalid value", []string{"id", "--disabled", "invalid"}, nil},
		{"too many arguments", []string{"id", "extra", "--disabled", "false"}, []string{"id", "extra", "--disabled=false"}},
		{"string value resembles flag", []string{"id", "--description", "--disabled", "false"}, nil},
		{"short value resembles flag", []string{"id", "-d", "--disabled", "false"}, nil},
		{"short cluster value resembles flag", []string{"id", "-sd", "--disabled", "false"}, nil},
		{"attached short value", []string{"id", "-ddescription=--disabled", "--disabled", "false"}, []string{"id", "-ddescription=--disabled", "--disabled=false"}},
		{"short equals value", []string{"id", "-d=description", "--disabled", "false"}, []string{"id", "-d=description", "--disabled=false"}},
		{"raw arrays", []string{"id", "-d", "items[]=a", "-d", "items[]=b", "--disabled", "false"}, []string{"id", "-d", "items[]=a", "-d", "items[]=b", "--disabled=false"}},
		{"array flags", []string{"id", "--items", "one", "--items", "two", "--disabled", "false"}, []string{"id", "--items", "one", "--items", "two", "--disabled=false"}},
		{"CLI boolean", []string{"id", "--confirm", "false"}, nil},
		{"dash boundary", []string{"id", "--disabled", "--", "false"}, nil},
		{"after dash", []string{"--", "id", "--disabled", "false"}, nil},
		{"value before dash", []string{"--disabled", "false", "--", "id"}, []string{"--disabled=false", "--", "id"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &cobra.Command{Use: "stripe", Annotations: make(map[string]string)}
			resource := NewResourceCmd(root, "widgets")
			oc := NewOperationCmd(resource.Cmd, &OperationSpec{
				Name: "update", Path: "/v1/widgets/{id}", Method: http.MethodPost,
				Params: map[string]*ParamSpec{
					"disabled":         {Type: "boolean"},
					"settings.enabled": {Type: "boolean"},
					"description":      {Type: "string"},
					"items":            {Type: "array"},
				},
			}, &config.Config{})
			args := append([]string{"widgets", "update"}, tt.args...)
			want := args
			if tt.want != nil {
				want = append([]string{"widgets", "update"}, tt.want...)
			}
			original := append([]string(nil), args...)
			require.Equal(t, want, NormalizeBooleanRequestArgs(root, args))
			require.Equal(t, original, args, "normalization must not mutate the caller's arguments")
			oc.Cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				require.False(t, flag.Changed, "normalization must not set --%s", flag.Name)
			})
			require.Empty(t, *oc.arrayFlags["items"])
			require.False(t, *oc.boolFlags["disabled"])
		})
	}
}

func TestNormalizeBooleanRequestArgsInheritedFlags(t *testing.T) {
	root := &cobra.Command{Use: "stripe", Annotations: make(map[string]string)}
	var project string
	root.PersistentFlags().StringVarP(&project, "project-name", "p", "", "Project")
	resource := NewResourceCmd(root, "widgets")
	NewOperationCmd(resource.Cmd, &OperationSpec{
		Name: "update", Path: "/v1/widgets/{id}", Method: http.MethodPost,
		Params: map[string]*ParamSpec{"disabled": {Type: "boolean"}},
	}, &config.Config{})
	for _, flag := range []string{"--project-name", "-p"} {
		args := []string{flag, "--disabled", "widgets", "update", "id", "--disabled", "false"}
		want := []string{flag, "--disabled", "widgets", "update", "id", "--disabled=false"}
		require.Equal(t, want, NormalizeBooleanRequestArgs(root, args))
		require.Empty(t, project)
	}
}

func TestNormalizeBooleanRequestArgsMissingID(t *testing.T) {
	root := &cobra.Command{Use: "stripe", Annotations: make(map[string]string), SilenceErrors: true, SilenceUsage: true}
	resource := NewResourceCmd(root, "widgets")
	NewOperationCmd(resource.Cmd, &OperationSpec{
		Name: "update", Path: "/v1/widgets/{id}", Method: http.MethodPost,
		Params: map[string]*ParamSpec{"disabled": {Type: "boolean"}},
	}, &config.Config{})
	// Without normalization, "false" would become the ID and disabled would be true.
	root.SetArgs(NormalizeBooleanRequestArgs(root, []string{"widgets", "update", "--disabled", "false", "--dry-run"}))
	require.ErrorContains(t, root.Execute(), "requires exactly 1 positional argument")
}
