package resource

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NormalizeBooleanRequestArgs rewrites `--flag true|false` to `--flag=true|false`
// for API boolean flags, so the value is never taken as a positional argument
// (such as the object ID). It never sets flag values.
func NormalizeBooleanRequestArgs(root *cobra.Command, args []string) []string {
	cmd, _, err := root.Find(args)
	if err != nil || cmd.Parent() == nil || cmd.Parent().Annotations[cmd.Name()] != "operation" {
		return args
	}

	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.SetNormalizeFunc(cmd.Flags().GetNormalizeFunc())
	flags.AddFlagSet(cmd.Flags())
	flags.AddFlagSet(cmd.InheritedFlags())

	normalized := append([]string(nil), args...)
	for i := 0; i+1 < len(normalized); i++ {
		name, longFlag := strings.CutPrefix(normalized[i], "--")
		flag := flags.Lookup(name)
		if !longFlag || flag == nil || flag.Value.Type() != "bool" || len(flag.Annotations["request"]) == 0 ||
			(normalized[i+1] != "true" && normalized[i+1] != "false") {
			continue
		}
		if endsWithFlag(flags, normalized[:i+1], flag) {
			normalized[i] += "=" + normalized[i+1]
			normalized = append(normalized[:i+1], normalized[i+2:]...)
		}
	}
	return normalized
}

// endsWithFlag distinguishes a flag from another flag's value or a positional
// argument, including tokens after --, using pflag's own parsing rules.
// ParseAll with a callback leaves flag values untouched.
func endsWithFlag(flags *pflag.FlagSet, args []string, want *pflag.Flag) bool {
	var last *pflag.Flag
	positionalCount := -1
	err := flags.ParseAll(args, func(flag *pflag.Flag, _ string) error {
		last, positionalCount = flag, flags.NArg()
		return nil
	})
	return err == nil && last == want && positionalCount == flags.NArg()
}
