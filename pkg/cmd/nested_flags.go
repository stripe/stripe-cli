package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// flagErrorWithNestedAPIHint wraps unknown-flag errors so a dotted API field
// path (the usual guess from API docs, e.g. --result_options.compress_file)
// points at the dedicated CLI flag and/or the accepted bracket form.
func flagErrorWithNestedAPIHint(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	flagName, ok := unknownLongFlagName(err.Error())
	if !ok {
		return err
	}

	hint := nestedAPIFieldHint(cmd, flagName)
	if hint == "" {
		return err
	}

	return errorcategory.UserInputErrorf("%s\n%s", err.Error(), hint)
}

func unknownLongFlagName(message string) (string, bool) {
	const prefix = "unknown flag: --"
	if !strings.HasPrefix(message, prefix) {
		return "", false
	}
	name := strings.TrimSpace(strings.TrimPrefix(message, prefix))
	if name == "" || strings.Contains(name, "[") {
		return "", false
	}
	return name, true
}

func nestedAPIFieldHint(cmd *cobra.Command, flagName string) string {
	if !strings.Contains(flagName, ".") {
		return ""
	}

	bracket := dottedAPIFieldToBracketFlag(flagName)
	lastSegment := flagName[strings.LastIndex(flagName, ".")+1:]
	lastKebab := strings.ReplaceAll(lastSegment, "_", "-")
	hyphenDot := strings.ReplaceAll(flagName, "_", "-")

	var dedicated []string
	if hasLocalFlag(cmd, lastKebab) {
		dedicated = append(dedicated, "--"+lastKebab)
	}
	if hyphenDot != lastKebab && hyphenDot != flagName && hasLocalFlag(cmd, hyphenDot) {
		dedicated = append(dedicated, "--"+hyphenDot)
	}

	if len(dedicated) == 1 {
		return "Nested API fields are not dotted flags. Use " + dedicated[0] +
			", or the raw form --" + bracket + "=value."
	}
	if len(dedicated) > 1 {
		return "Nested API fields are not dotted flags. Use " + strings.Join(dedicated, " or ") +
			", or the raw form --" + bracket + "=value."
	}

	return "Nested API fields use bracket notation, e.g. --" + bracket +
		"=value — not dotted --" + flagName + "."
}

func dottedAPIFieldToBracketFlag(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return name
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += "[" + part + "]"
	}
	return out
}

func hasLocalFlag(cmd *cobra.Command, name string) bool {
	if cmd == nil || name == "" {
		return false
	}
	flag := cmd.Flags().Lookup(name)
	return flag != nil && !flag.Hidden
}
