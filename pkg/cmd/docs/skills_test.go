package cmd_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cliconfig "github.com/stripe/stripe-cli/pkg/config"

	"github.com/stripe/stripe-cli-docs-plugin/cmd"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
)

func TestSkillsListCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.well-known/skills/index.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"skills":[{"name":"stripe-best-practices","description":"Guides Stripe integration decisions.","files":["SKILL.md"]},{"name":"upgrade-stripe","description":"Guide for upgrading Stripe API versions and SDKs","files":["SKILL.md"]}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(cmd.WithClient(client)).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"skills", "list"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "Install agent skills")
	assert.Contains(t, plainOutput, "stripe-best-practices")
	assert.Contains(t, plainOutput, "Guides Stripe integration decisions.")
	assert.Contains(t, plainOutput, "upgrade-stripe")
	assert.Contains(t, plainOutput, "Guide for upgrading Stripe API versions and SDKs")
	assert.Contains(t, plainOutput, "•")

	// Preamble should appear before the skill list.
	preambleIdx := strings.Index(plainOutput, "Install agent skills")
	firstSkillIdx := strings.Index(plainOutput, "stripe-best-practices")
	assert.Less(t, preambleIdx, firstSkillIdx)
}

func TestSkillsListCommand_ColorOff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"skills":[{"name":"stripe-best-practices","description":"Guides Stripe integration decisions.","files":["SKILL.md"]}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(&cliconfig.Config{Color: "off"}),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"skills", "list"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	output := out.String()
	assert.NotContains(t, output, "\x1b[", "expected no ANSI escape codes with color off")
	assert.Contains(t, output, "stripe-best-practices")
	assert.Contains(t, output, "Guides Stripe integration decisions.")
}

func TestSkillsListCommand_DescriptionWrapsAt120(t *testing.T) {
	longDesc := strings.Repeat("word ", 30) // 150 chars, well over 120
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"skills":[{"name":"skill-a","description":%q,"files":[]}]}`, longDesc)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(&cliconfig.Config{Color: "off"}),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"skills", "list"})

	require.NoError(t, root.ExecuteContext(context.Background()))

	for _, line := range strings.Split(out.String(), "\n") {
		assert.LessOrEqual(t, len(stripANSI(line)), 120+4, // +4 for bullet + indent
			"line exceeds 120 chars: %q", line)
	}
}

func TestSkillsListCommand_FetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	root := cmd.New().WithOptions(cmd.WithClient(client)).Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"skills", "list"})

	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skills list:")
}
