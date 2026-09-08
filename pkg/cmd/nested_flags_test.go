package cmd

import (
	"regexp"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDottedAPIFieldToBracketFlag(t *testing.T) {
	assert.Equal(t, "result_options[compress_file]", dottedAPIFieldToBracketFlag("result_options.compress_file"))
	assert.Equal(t, "shipping[address][line1]", dottedAPIFieldToBracketFlag("shipping.address.line1"))
	assert.Equal(t, "plain", dottedAPIFieldToBracketFlag("plain"))
}

func TestUnknownLongFlagName(t *testing.T) {
	name, ok := unknownLongFlagName("unknown flag: --result_options.compress_file")
	assert.True(t, ok)
	assert.Equal(t, "result_options.compress_file", name)

	_, ok = unknownLongFlagName("unknown flag: --result_options[compress_file]")
	assert.False(t, ok)

	_, ok = unknownLongFlagName("unknown command: foo")
	assert.False(t, ok)
}

func TestReportingCreate_AcceptsBracketCompressFileAlias(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()
	err := cc.cmd.ParseFlags([]string{"--result_options[compress_file]=true"})
	require.NoError(t, err)
	assert.True(t, cc.compressFile)
}

func TestReportingCreate_DottedCompressFileSuggestsDedicatedFlag(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	root.SetFlagErrorFunc(flagErrorWithNestedAPIHint)
	root.AddCommand(newReportingCmd().cmd)

	_, err := executeCommand(root, "reporting", "query-runs", "create", "--result_options.compress_file=true")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag: --result_options.compress_file")
	assert.Contains(t, err.Error(), "--compress-file")
	assert.Contains(t, err.Error(), "--result_options[compress_file]=value")
}

func TestAnalyticsFlagUsagesDoNotLookLikeDottedFlags(t *testing.T) {
	re := regexp.MustCompile(`\(sets [A-Za-z0-9_]+\.[A-Za-z0-9_.]+\)`)
	cmds := []*cobra.Command{
		newReportingQueryRunsCreateCmd().cmd,
		newReportingQueryRunsRetrieveCmd().cmd,
		newDataMetricsRunCmd().cmd,
	}
	for _, cmd := range cmds {
		cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			if re.MatchString(flag.Usage) {
				t.Errorf("%s --%s usage looks like dotted flag syntax: %q", cmd.Use, flag.Name, flag.Usage)
			}
		})
	}
}
