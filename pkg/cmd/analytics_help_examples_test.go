package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var helpExamplePlaceholder = regexp.MustCompile(`<[^>]+>`)

type helpExample struct {
	Name string
	Raw  string
	Argv []string
}

func extractHelpExamples(example string) []helpExample {
	blocks := strings.Split(example, "\n\n")
	var out []helpExample
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		name := ""
		var cmdLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				if name == "" {
					name = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
				}
				continue
			}
			cmdLines = append(cmdLines, line)
		}
		joined := strings.TrimSpace(joinBackslashContinuations(cmdLines))
		if joined == "" {
			continue
		}
		argv, err := tokenizeHelpExample(joined)
		if err != nil {
			continue
		}
		out = append(out, helpExample{Name: name, Raw: joined, Argv: argv})
	}
	return out
}

func joinBackslashContinuations(lines []string) string {
	var b strings.Builder
	for i, line := range lines {
		trimmedRight := strings.TrimRightFunc(line, unicode.IsSpace)
		cont := strings.HasSuffix(trimmedRight, "\\")
		if cont {
			trimmedRight = strings.TrimSpace(strings.TrimSuffix(trimmedRight, "\\"))
		}
		b.WriteString(strings.TrimSpace(trimmedRight))
		if i < len(lines)-1 || cont {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func tokenizeHelpExample(s string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case unicode.IsSpace(rune(c)) && !inQuote:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	if inQuote {
		return nil, errUnclosedQuote
	}
	return tokens, nil
}

var errUnclosedQuote = errString("unclosed quote in help example")

type errString string

func (e errString) Error() string { return string(e) }

func helpExampleIsPlaceholder(ex helpExample) bool {
	return helpExamplePlaceholder.MatchString(ex.Raw)
}

func analyticsHelpCommands() []*cobra.Command {
	return []*cobra.Command{
		newDataCmd().cmd,
		newDataMetricsCmd().cmd,
		newDataMetricsRunCmd().cmd,
		newReportingCmd().cmd,
		newReportingQueryRunsCmd().cmd,
		newReportingQueryRunsCreateCmd().cmd,
		newReportingQueryRunsRetrieveCmd().cmd,
	}
}

func TestAnalyticsHelpExamples_NoRottingCopyPaste(t *testing.T) {
	prevKey := Config.Profile.APIKey
	Config.Profile.APIKey = "sk_test_1234567890abcdef"
	t.Cleanup(func() { Config.Profile.APIKey = prevKey })

	var captured []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "metric_query") {
			_, _ = w.Write([]byte(`{"object":"v2.data.analytics.metric_query_result","data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"qryrun_test_ok","object":"v2.data.reporting.query_run","status":"running"}`))
	}))
	t.Cleanup(ts.Close)

	var ran int
	for _, cmd := range analyticsHelpCommands() {
		examples := extractHelpExamples(cmd.Example)
		for _, ex := range examples {
			t.Run(cmd.Name()+"/"+ex.Name, func(t *testing.T) {
				require.NotEmpty(t, ex.Argv, "example should parse to argv: %s", ex.Raw)
				require.Equal(t, "stripe", ex.Argv[0], "example should start with stripe: %s", ex.Raw)

				if helpExampleIsPlaceholder(ex) {
					assert.Regexp(t, helpExamplePlaceholder, ex.Raw)
					return
				}

				require.NotContains(t, ex.Raw, "|", "piped examples cannot be copy-pasted as a stripe command")
				require.NotContains(t, ex.Argv, "--sql-file", "sql-file examples depend on a local path; keep them out of copy-paste Examples")

				captured = nil
				args := append(append([]string{}, ex.Argv[1:]...), "--api-base", ts.URL)

				root := &cobra.Command{Use: "stripe", SilenceUsage: true, SilenceErrors: true}
				root.AddCommand(newDataCmd().cmd)
				root.AddCommand(newReportingCmd().cmd)

				_, err := executeCommand(root, args...)
				require.NoError(t, err, "copy-pasteable help example failed: %s", ex.Raw)
				ran++

				if strings.Contains(ex.Raw, "--group-by") {
					var body map[string]interface{}
					require.NoError(t, json.Unmarshal(captured, &body))
					groupBy, ok := body["group_by"].([]interface{})
					require.True(t, ok, "group_by should serialize as a JSON array of strings")
					require.NotEmpty(t, groupBy)
					_, isString := groupBy[0].(string)
					require.True(t, isString, "group_by entries should be strings, got %T", groupBy[0])
					assert.Equal(t, "price", groupBy[0])
				}
			})
		}
	}
	require.Greater(t, ran, 0, "expected at least one copy-pasteable help example to execute")
}

func TestExtractHelpExamples_GroupByPrice(t *testing.T) {
	examples := extractHelpExamples(newDataMetricsRunCmd().cmd.Example)
	var found bool
	for _, ex := range examples {
		if strings.Contains(ex.Raw, "--group-by") {
			found = true
			assert.Contains(t, ex.Argv, "--group-by")
			assert.Contains(t, ex.Argv, "price")
			assert.NotContains(t, ex.Argv, "product")
			assert.False(t, helpExampleIsPlaceholder(ex))
		}
	}
	require.True(t, found, "expected a group-by example")
}

func TestExtractHelpExamples_NoCopyPasteableMetricIDs(t *testing.T) {
	reRealLookingID := regexp.MustCompile(`metric_[0-9A-Za-z]+`)
	for _, cmd := range analyticsHelpCommands() {
		assert.NotRegexp(t, reRealLookingID, cmd.Example, "%s example should not include a copy-pasteable metric id", cmd.Use)
		cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			assert.NotRegexp(t, reRealLookingID, flag.Usage, "%s --%s should not include a copy-pasteable metric id", cmd.Use, flag.Name)
		})
	}
}
