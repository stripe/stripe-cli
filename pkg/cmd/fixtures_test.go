package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-cli/pkg/config"
)

func TestFixturesCmdMultipleFiles(t *testing.T) {
	requestsReceived := make(map[string]int)
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		requestsReceived[req.URL.String()]++
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(`{"id": "test_id"}`))
	}))
	defer ts.Close()

	tempDir, err := os.MkdirTemp("", "stripe-cli-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fixture1Path := filepath.Join(tempDir, "fixture1.json")
	fixture1Content := fmt.Sprintf(`{
		"_meta": { "template_version": 0 },
		"fixtures": [
			{
				"name": "cust_1",
				"path": "/v1/customers",
				"method": "post"
			}
		]
	}`)
	os.WriteFile(fixture1Path, []byte(fixture1Content), 0644)

	fixture2Path := filepath.Join(tempDir, "fixture2.json")
	fixture2Content := fmt.Sprintf(`{
		"_meta": { "template_version": 0 },
		"fixtures": [
			{
				"name": "cust_2",
				"path": "/v1/customers",
				"method": "post"
			}
		]
	}`)
	os.WriteFile(fixture2Path, []byte(fixture2Content), 0644)

	cfg := &config.Config{
		Profile: config.Profile{
			APIKey: "sk_test_this_is_a_dummy_key_for_testing_purposes_only",
		},
	}
	fc := newFixturesCmd(cfg)
	fc.apiBaseURL = ts.URL

	ctx := context.Background()
	fc.Cmd.SetContext(ctx)

	err = fc.runFixturesCmd(fc.Cmd, []string{fixture1Path, fixture2Path})
	require.NoError(t, err)

	// Verify that /v1/customers was called twice (once for each fixture)
	require.Equal(t, 2, requestsReceived["/v1/customers"])
}

func TestFixturesCmdMinimumArgs(t *testing.T) {
	cfg := &config.Config{}
	fc := newFixturesCmd(cfg)

	// Test with 0 args - should fail because we changed to MinimumNArgs(1)
	err := fc.Cmd.Args(fc.Cmd, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires at least 1 positional argument")

	// Test with 1 arg - should pass validation
	err = fc.Cmd.Args(fc.Cmd, []string{"fixture.json"})
	require.NoError(t, err)

	// Test with 2 args - should pass validation
	err = fc.Cmd.Args(fc.Cmd, []string{"f1.json", "f2.json"})
	require.NoError(t, err)
}
