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
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
)

var databaseTemplateFuncsOnce sync.Once

func ensureDatabaseTemplateFuncs() {
	databaseTemplateFuncsOnce.Do(func() {
		cobra.AddTemplateFunc("WrappedInheritedFlagUsages", func(cmd *cobra.Command) string {
			return cmd.InheritedFlags().FlagUsagesWrapped(80)
		})
		cobra.AddTemplateFunc("WrappedLocalFlagUsages", func(cmd *cobra.Command) string {
			return cmd.LocalFlags().FlagUsagesWrapped(80)
		})
		cobra.AddTemplateFunc("WrappedRequestParamsFlagUsages", func(cmd *cobra.Command) string {
			return cmd.LocalFlags().FlagUsagesWrapped(80)
		})
		cobra.AddTemplateFunc("WrappedNonRequestParamsFlagUsages", func(cmd *cobra.Command) string {
			return cmd.LocalFlags().FlagUsagesWrapped(80)
		})
		cobra.AddTemplateFunc("AIAgentHelp", func(*cobra.Command) string { return "" })
	})
}

func newDatabaseTestRoot(cfg *config.Config) *cobra.Command {
	ensureDatabaseTemplateFuncs()
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

	createHelp, err := executeDatabaseCommand(root, nil, "create", "--help")
	require.NoError(t, err)
	require.Contains(t, createHelp, "--api-version")
	require.Contains(t, createHelp, "--dry-run")
	require.Contains(t, createHelp, "--live")
	require.Contains(t, createHelp, "--stripe-account")
	require.NotContains(t, createHelp, "--stripe-version")

	deleteHelp, err := executeDatabaseCommand(root, nil, "delete", "--help")
	require.NoError(t, err)
	require.Contains(t, deleteHelp, "--yes")
	require.NotContains(t, deleteHelp, "--confirm")
	require.NotContains(t, deleteHelp, "--stripe-version")
}

func TestDatabaseCommands(t *testing.T) {
	t.Run("create prints created database details", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "create", "--api-version", "2026-01-28.clover")
		require.NoError(t, err)
		require.Contains(t, output, "Creating StripeDB instance...")
		require.Contains(t, output, "Created StripeDB instance db_1Xy...mN8pQr (StripeDB Instance)")
		require.Contains(t, output, "API Version: 2026-01-28.clover")
		require.Contains(t, output, "Mode: test")
		require.Contains(t, output, "Host:      db_1XyZ2aBcDeFgHiJkLmN8pQr.db.stripe.com")
		require.Contains(t, output, "Username:  llama_user")
		require.Contains(t, output, "Password:  pass_123")
		require.Contains(t, output, "Save this password now — it will not be shown again.")
		require.Contains(t, output, "Dashboard:")
		require.Contains(t, output, "https://dashboard.stripe.com/test/data-management/databases/db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.Contains(t, output, "Current status: Backfilling. Check progress with:")
		require.Contains(t, output, "https://stripe.com/privacy")
		require.Contains(t, output, "https://stripe.com/stripe-database-preview-terms")
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
		require.Contains(t, output, "StripeDB instance db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.Contains(t, output, "View in Dashboard:")
		require.Contains(t, output, "Backfilling")
		require.Contains(t, output, "ago") // Created field uses databaseRelativeTimeAgo
		require.Contains(t, output, "test")
		require.Contains(t, output, "2026-01-28.clover")
		require.Contains(t, output, "db_1XyZ2aBcDeFgHiJkLmN8pQr.db.stripe.com")
		require.Contains(t, output, "https://dashboard.stripe.com/test/data-management/databases/db_1XyZ2aBcDeFgHiJkLmN8pQr")
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
		require.Contains(t, output, "ID")
		require.Regexp(t, `db_1XyZ2aBcDeFgHiJkLmN8pQr\s+○ Backfilling\s+5d\s+test\s+2026-01-28\.clover`, output)
		require.Regexp(t, `db_jS7fnaBcDeFgHiJkLmN8pQr\s+● Active\s+1y\s+test\s+2026-02-15\.clover`, output)
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

	t.Run("list dry-run uses unsafe preview header", func(t *testing.T) {
		root := newDatabaseTestRoot(&config.Config{})

		output, err := executeDatabaseCommand(root, nil, "list", "--dry-run")
		require.NoError(t, err)
		require.Contains(t, output, `"Stripe-Version": "unsafe-development"`)
		require.Contains(t, output, `"url": "https://api.stripe.com/v2/data/databases"`)
	})

	t.Run("delete shows confirmation prompt when declined", func(t *testing.T) {
		root := newDatabaseTestRoot(&config.Config{})
		output, err := executeDatabaseCommand(root, bytes.NewBufferString("n\n"), "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.NoError(t, err)
		require.Contains(t, output, "Warning: this will permanently delete StripeDB Instance (db_1Xy...mN8pQr).")
		require.Contains(t, output, `Type delete database to continue.`)
	})

	t.Run("delete removes database after confirmation phrase", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, bytes.NewBufferString(databaseDeleteConfirmationPhrase+"\n"), "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.NoError(t, err)
		require.Contains(t, output, `Type delete database to continue.`)
		require.Contains(t, output, "Deleted StripeDB Instance")
		require.Contains(t, output, "(db_1Xy...mN8pQr)")
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
		require.Contains(t, output, "Created StripeDB user dbuser_1JqM7xBcDeFgHiJkLmN8pQrS")
		require.Contains(t, output, "Username: llama_user")
		require.Contains(t, output, "Mode: test")
		require.Contains(t, output, "Password:  new_pass_123")
		require.Contains(t, output, "Save this password now — it will not be shown again.")
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
		require.Contains(t, output, "StripeDB user dbuser_1JqM7xBcDeFgHiJkLmN8pQrS")
		require.Contains(t, output, "Username:")
		require.Contains(t, output, "llama_user")
	})

	t.Run("list prints database users", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "users", "list", "db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.NoError(t, err)
		require.Contains(t, output, "StripeDB users for db_1XyZ2aBcDeFgHiJkLmN8pQr")
		require.Regexp(t, `llama_user\s+dbuser_1JqM7xBcDeFgHiJkLmN8pQrS\s+2h\s+test`, output)
		require.Regexp(t, `rotated_user\s+dbuser_4NtP9yBcDeFgHiJkLmN8pQrV\s+2mo\s+test`, output)
	})

	t.Run("delete with yes prints success", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, nil, "users", "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "dbuser_1JqM7xBcDeFgHiJkLmN8pQrS", "--yes")
		require.NoError(t, err)
		require.Contains(t, output, "Deleted dbuser_1JqM7xBcDeFgHiJkLmN8pQrS")
	})

	t.Run("delete after confirmation phrase prints success", func(t *testing.T) {
		root := newDatabaseFixtureTestRoot(t)

		output, err := executeDatabaseCommand(root, bytes.NewBufferString(databaseUserDeleteConfirmationText+"\n"), "users", "delete", "db_1XyZ2aBcDeFgHiJkLmN8pQr", "dbuser_1JqM7xBcDeFgHiJkLmN8pQrS")
		require.NoError(t, err)
		require.Contains(t, output, `Type remove user to continue.`)
		require.Contains(t, output, "Deleted dbuser_1JqM7xBcDeFgHiJkLmN8pQrS")
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
		// Columns: ID, Status (with glyph), Created, Mode, API Version (Name column disabled until API ships display_name)
		separatorFields := strings.Fields(lines[1])
		require.Len(t, separatorFields, 5)
		// Verify separators use Unicode box-drawing characters
		require.Contains(t, separatorFields[0], "─")
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
		// Status column should be green-colored (ready = Active)
		require.Regexp(t, `\x1b\[[0-9;]*m● Active\x1b\[[0-9;]*m`, lines[2])
		// ID column should be muted
		require.Regexp(t, `\x1b\[[0-9;]*mdb_1\x1b\[[0-9;]*m`, lines[2])
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
