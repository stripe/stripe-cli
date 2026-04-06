package resource

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
)

func newDatabaseTestRoot(cfg *config.Config) *cobra.Command {
	return newDatabasesCmd(databaseTestConfig(cfg))
}

func executeDatabaseCommand(root *cobra.Command, input io.Reader, args ...string) (string, error) {
	combinedOutput := new(bytes.Buffer)
	root.SetOut(combinedOutput)
	root.SetErr(combinedOutput)
	if input != nil {
		root.SetIn(input)
	}
	root.SetArgs(args)

	_, err := root.ExecuteC()
	root.SetArgs([]string{})

	return combinedOutput.String(), err
}

var databaseTestNow = time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)

func databaseTestConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		cfg = &config.Config{}
	}

	cloned := *cfg
	if cloned.Profile.APIKey == "" {
		cloned.Profile.APIKey = "sk_test_1234"
	}

	return &cloned
}

func freezeDatabaseNow(t *testing.T) {
	previousNow := databaseNow
	databaseNow = func() time.Time {
		return databaseTestNow
	}
	t.Cleanup(func() {
		databaseNow = previousNow
	})
}

func databaseFixtureDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(filename), "testdata", "database-api")
}

func readDatabaseFixtureText(t *testing.T, name string) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(databaseFixtureDir(t), name))
	require.NoError(t, err)
	return string(body)
}

func databaseFixtureRequestKey(method, apiPath string) string {
	return method + " " + apiPath
}

func databaseFixtureFilename(method, apiPath string) string {
	normalizedPath := strings.TrimPrefix(apiPath, "/")
	normalizedPath = strings.ReplaceAll(normalizedPath, "/", "_")
	return strings.ToUpper(method) + "_" + normalizedPath + ".json"
}

func newDatabaseFixtureServer(t *testing.T, overrides map[string]string) *httptest.Server {
	t.Helper()

	fixtureDir := databaseFixtureDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected bearer authorization header, got %q", auth)
		}
		if got := r.Header.Get("Stripe-Version"); got != "unsafe-development" {
			t.Errorf("expected Stripe-Version header to be unsafe-development, got %q", got)
		}

		body, ok := overrides[databaseFixtureRequestKey(r.Method, r.URL.Path)]
		if !ok {
			fixturePath := filepath.Join(fixtureDir, databaseFixtureFilename(r.Method, r.URL.Path))
			fixtureBytes, err := os.ReadFile(fixturePath)
			require.NoErrorf(t, err, "missing fixture for %s %s", r.Method, r.URL.Path)
			body = string(fixtureBytes)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, body)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	return server
}

func setDatabaseAPIBase(t *testing.T, cmd *cobra.Command, apiBaseURL string) {
	t.Helper()

	if cmd.Flags().Lookup("api-base") != nil {
		require.NoError(t, cmd.Flags().Set("api-base", apiBaseURL))
	}

	for _, child := range cmd.Commands() {
		setDatabaseAPIBase(t, child, apiBaseURL)
	}
}

func newDatabaseTestRootWithAPIBase(t *testing.T, cfg *config.Config, apiBaseURL string) *cobra.Command {
	t.Helper()

	root := newDatabaseTestRoot(cfg)
	setDatabaseAPIBase(t, root, apiBaseURL)
	return root
}

func newDatabaseFixtureTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	return newDatabaseFixtureTestRootWithConfig(t, &config.Config{})
}

func newDatabaseFixtureTestRootWithConfig(t *testing.T, cfg *config.Config) *cobra.Command {
	t.Helper()
	freezeDatabaseNow(t)
	server := newDatabaseFixtureServer(t, nil)
	return newDatabaseTestRootWithAPIBase(t, cfg, server.URL)
}

func withAnsiColors(t *testing.T) {
	t.Helper()

	previousForceColors := ansi.ForceColors
	previousDisableColors := ansi.DisableColors
	ansi.ForceColors = true
	ansi.DisableColors = false
	t.Cleanup(func() {
		ansi.ForceColors = previousForceColors
		ansi.DisableColors = previousDisableColors
	})
}

func compactDatabaseFixtureJSON(t *testing.T, name string) string {
	t.Helper()

	var compact bytes.Buffer
	err := json.Compact(&compact, []byte(readDatabaseFixtureText(t, name)))
	require.NoError(t, err)
	return compact.String()
}

func TestAddDatabasesCmd(t *testing.T) {
	root := &cobra.Command{
		Use:         "stripe",
		Annotations: make(map[string]string),
	}
	root.AddCommand(&cobra.Command{Use: "databases", Short: "generated"})

	err := AddDatabasesCmd(root, &config.Config{})
	require.NoError(t, err)

	count := 0
	for _, cmd := range root.Commands() {
		if cmd.Use == "databases" {
			count++
			require.True(t, cmd.Hidden)
			require.Equal(t, "Manage StripeDB (unstable preview APIs)", cmd.Short)
		}
	}
	require.Equal(t, 1, count)
}

func TestDatabaseHelp(t *testing.T) {
	root := newDatabaseTestRoot(&config.Config{})

	output, err := executeDatabaseCommand(root, nil, "--help")
	require.NoError(t, err)
	require.Contains(t, output, "Manage StripeDB")
	require.Contains(t, output, "unstable preview APIs")
	require.Contains(t, output, "--json")
	require.Contains(t, output, "create")
	require.Contains(t, output, "users")
}

func TestDatabaseCommands(t *testing.T) {
	t.Run("create prints created database details", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "create", "--api-version", "2026-01-28.clover")
		require.NoError(t, err)
		require.Equal(t, "Created StripeDB instance db_1XyZ2aBcDeFgHiJkLmN8pQr\n  API Version: 2026-01-28.clover\n  Mode: test\n\nConnection details:\n  Host:      db_1XyZ2aBcDeFgHiJkLmN8pQr.db.stripe.com\n  Username:  llama_user\n  Password:  pass_123\n  URL:       postgresql://llama_user:pass_123@db_1XyZ2aBcDeFgHiJkLmN8pQr.db.stripe.com:5432/data\n\n  Current status: backfilling. Check progress with: stripe databases retrieve db_1XyZ2aBcDeFgHiJkLmN8pQr\n", output)
	})

	t.Run("create pretty prints json output", func(t *testing.T) {
		freezeDatabaseNow(t)
		server := newDatabaseFixtureServer(t, map[string]string{
			databaseFixtureRequestKey(http.MethodPost, "/v2/data/databases"): compactDatabaseFixtureJSON(t, "POST_v2_data_databases.json"),
		})
		root := newDatabaseTestRootWithAPIBase(t, &config.Config{}, server.URL)
		output, err := executeDatabaseCommand(root, nil, "create", "--json")
		require.NoError(t, err)
		require.Equal(t, strings.TrimRight(readDatabaseFixtureText(t, "POST_v2_data_databases.json"), "\n")+"\n", output)
		require.Contains(t, output, "\n  \"id\":")
	})

	t.Run("retrieve supports envelope responses", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "retrieve", "db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.NoError(t, err)
		require.Contains(t, output, "ID")
		require.Regexp(t, `db_1XyZ2aBcDeFgHiJkLmN8pQr\s+db_1XyZ2aBcDeFgHiJkLmN8pQr\.db\.stripe\.com\s+backfilling\s+5d\s+test\s+2026-01-28\.clover`, output)
	})

	t.Run("list prints account databases", func(t *testing.T) {
		root := newDatabaseFixtureTestRootWithConfig(t, &config.Config{
			Profile: config.Profile{
				APIKey:    "sk_test_1234",
				AccountID: "acct_123",
			},
		})

		output, err := executeDatabaseCommand(root, nil, "list")
		require.NoError(t, err)
		require.Contains(t, output, `StripeDB instances for account acct_123`)
		require.Regexp(t, `db_1XyZ2aBcDeFgHiJkLmN8pQr\s+db_1XyZ2aBcDeFgHiJkLmN8pQr\.db\.stripe\.com\s+backfilling\s+5d\s+test\s+2026-01-28\.clover`, output)
		require.Regexp(t, `db_jS7fnaBcDeFgHiJkLmN8pQr\s+db_jS7fnaBcDeFgHiJkLmN8pQr\.db\.stripe\.com\s+ready\s+1y\s+test\s+2026-02-15\.clover`, output)
	})

	t.Run("list omits fake account placeholder when account is unavailable", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "list")
		require.NoError(t, err)
		require.Contains(t, output, "StripeDB instances\n\n")
		require.NotContains(t, output, "acct_xxx")
	})

	t.Run("list returns json", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "list", "--json")
		require.NoError(t, err)
		require.JSONEq(t, readDatabaseFixtureText(t, "GET_v2_data_databases.json"), output)
		require.Contains(t, output, "\n  \"data\": [")
	})

	t.Run("delete shows confirmation prompt when declined", func(t *testing.T) {
		root := newDatabaseTestRoot(&config.Config{})
		output, err := executeDatabaseCommand(root, bytes.NewBufferString("n\n"), "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.NoError(t, err)
		require.Contains(t, output, "Warning: this will permanently delete your StripeDB instance.")
		require.Contains(t, output, `Type remove StripeDB to continue.`)
	})

	t.Run("delete removes database after confirmation phrase", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, bytes.NewBufferString(databaseDeleteConfirmationPhrase+"\n"), "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.NoError(t, err)
		require.Contains(t, output, `Type remove StripeDB to continue.`)
		require.Contains(t, output, "Deleted StripeDB instance db_1XyZ2aBcDeFgHiJkLmN8pQr")
	})

	t.Run("delete requires yes with json output", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "--json")
		require.ErrorContains(t, err, "--yes is required with --json")
		require.Contains(t, output, "--yes is required with --json")
	})

	t.Run("delete returns json when confirmed", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "--yes", "--json")
		require.NoError(t, err)
		require.JSONEq(t, readDatabaseFixtureText(t, "DELETE_v2_data_databases_db_1XyZ2aBcDeFgHiJkLmN8pQr.json"), output)
	})
}

func TestDatabaseUserCommands(t *testing.T) {
	t.Run("create prints connection details", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "users", "create", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "--username", "llama_user")
		require.NoError(t, err)
		require.Equal(t, "Created StripeDB user dbuser_1JqM7xBcDeFgHiJkLmN8pQrS\n  Username: llama_user\n  Mode: test\n\nConnection details:\n  Password:  new_pass_123\n  URL:       postgresql://llama_user:new_pass_123@db_1XyZ2aBcDeFgHiJkLmN8pQr.db.stripe.com:5432/data\n", output)
	})

	t.Run("create returns json", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "users", "create", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "--username", "llama_user", "--json")
		require.NoError(t, err)
		require.JSONEq(t, readDatabaseFixtureText(t, "POST_v2_data_databases_db_1XyZ2aBcDeFgHiJkLmN8pQr_users.json"), output)
	})

	t.Run("retrieve supports envelope responses", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "users", "retrieve", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "dbuser_1JqM7xBcDeFgHiJkLmN8pQrS")
		require.NoError(t, err)
		require.Contains(t, output, "StripeDB users for db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.Regexp(t, `dbuser_1JqM7xBcDeFgHiJkLmN8pQrS\s+llama_user\s+2h\s+test`, output)
	})

	t.Run("list prints database users", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "users", "list", "db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.NoError(t, err)
		require.Contains(t, output, "StripeDB users for db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.Regexp(t, `dbuser_1JqM7xBcDeFgHiJkLmN8pQrS\s+llama_user\s+2h\s+test`, output)
		require.Regexp(t, `dbuser_4NtP9yBcDeFgHiJkLmN8pQrV\s+rotated_user\s+2mo\s+test`, output)
	})

	t.Run("delete with yes prints success", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "users", "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "dbuser_1JqM7xBcDeFgHiJkLmN8pQrS", "--yes")
		require.NoError(t, err)
		require.Equal(t, "Deleted StripeDB user dbuser_1JqM7xBcDeFgHiJkLmN8pQrS\n  StripeDB: db_1XyZ2aBcDeFgHiJkLmN8pQr\n", output)
	})

	t.Run("delete after confirmation phrase prints success", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, bytes.NewBufferString(databaseUserDeleteConfirmationText+"\n"), "users", "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "dbuser_1JqM7xBcDeFgHiJkLmN8pQrS")
		require.NoError(t, err)
		require.Contains(t, output, `Type remove user to continue.`)
		require.Contains(t, output, "> \nDeleted StripeDB user dbuser_1JqM7xBcDeFgHiJkLmN8pQrS\n")
	})
}

func TestDatabaseOutputHelpers(t *testing.T) {
	t.Run("table uses dynamic separators", func(t *testing.T) {
		freezeDatabaseNow(t)

		var out bytes.Buffer
		printDatabaseTable(&out, []databaseObject{
			{
				ID:         "db_1XyZ2aBcDeFgHiJkLmN8pQr",
				Livemode:   false,
				Created:    "2026-03-28T12:00:00Z",
				Status:     "backfilling",
				APIVersion: "2026-01-28.clover",
				Connection: databaseConnection{
					Host: "db_1XyZ2aBcDeFgHiJkLmN8pQr.db.stripe.com",
				},
			},
		})

		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		require.Len(t, lines, 3)
		require.Equal(t, []string{
			strings.Repeat("-", len("db_1XyZ2aBcDeFgHiJkLmN8pQr")),
			strings.Repeat("-", len("db_1XyZ2aBcDeFgHiJkLmN8pQr.db.stripe.com")),
			strings.Repeat("-", len("backfilling")),
			strings.Repeat("-", len("Created")),
			strings.Repeat("-", len("test")),
			strings.Repeat("-", len("2026-01-28.clover")),
		}, strings.Fields(lines[1]))
	})

	t.Run("table shows empty state", func(t *testing.T) {
		var out bytes.Buffer

		printDatabaseTable(&out, nil)
		require.Equal(t, "No StripeDB instances found.\n", out.String())
	})

	t.Run("table alternates column styling when color enabled", func(t *testing.T) {
		freezeDatabaseNow(t)
		withAnsiColors(t)

		var out bytes.Buffer
		printDatabaseTable(&out, []databaseObject{
			{
				ID:         "db_1",
				Livemode:   false,
				Created:    "2026-03-28T12:00:00Z",
				Status:     "ready",
				APIVersion: "2026-01-28.clover",
				Connection: databaseConnection{
					Host: "db_1.db.stripe.com",
				},
			},
		})

		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		require.Len(t, lines, 3)
		require.Regexp(t, `^db_1\s+\x1b\[[0-9;]*mdb_1\.db\.stripe\.com\x1b\[[0-9;]*m\s+ready\s+\x1b\[[0-9;]*m5d\x1b\[[0-9;]*m\s+test\s+\x1b\[[0-9;]*m2026-01-28\.clover\x1b\[[0-9;]*m$`, lines[2])
		require.NotRegexp(t, `\x1b\[[0-9;]*mready\x1b\[[0-9;]*m`, lines[2])
	})

	t.Run("relative time formats months and minutes differently", func(t *testing.T) {
		freezeDatabaseNow(t)

		require.Equal(t, "5m", databaseRelativeTime("2026-04-02T11:55:00Z"))
		require.Equal(t, "2mo", databaseRelativeTime("2026-02-02T12:00:00Z"))
		require.Equal(t, "1y", databaseRelativeTime("2025-04-02T12:00:00Z"))
	})

	t.Run("list heading omits account when unavailable", func(t *testing.T) {
		var out bytes.Buffer

		printDatabaseListHeading(&out, nil)
		require.Equal(t, "StripeDB instances\n\n", out.String())
	})
}
