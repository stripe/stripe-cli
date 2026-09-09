package resource

import (
	"strings"

	"github.com/spf13/pflag"
)

// normalizeBracketNestedFlag maps --parent[child]=value onto the generated
// flag name (underscores become hyphens, nesting becomes dots).
//
// OpenAPI nested fields are registered as flags like --result-options.compress-file.
// Users translating from the API reference often try the bracket form
// --result_options[compress_file]=true instead of the dotted-hyphen flag.
func normalizeBracketNestedFlag(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "[") {
		name = bracketFlagToGeneratedName(name)
	}
	return pflag.NormalizedName(name)
}

func bracketFlagToGeneratedName(name string) string {
	dotted := strings.ReplaceAll(strings.ReplaceAll(name, "[", "."), "]", "")
	return strings.ReplaceAll(dotted, "_", "-")
}
