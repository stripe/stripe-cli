package login

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestLogin(t *testing.T) {
	if os.Getenv("OPEN_URL") == "1" {
		os.Exit(0)
		return
	}

	execCommand = func(string, ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestLogin")
		cmd.Env = []string{"OPEN_URL=1"}
		return cmd
	}
	defer func() { execCommand = exec.Command }()

	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")

	p := config.Profile{
		DeviceName:  "st-testing",
		ProfileName: "tests",
	}

	c := &config.Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}

	var pollURL string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stripecli/auth":
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			expectedLinks := Links{
				BrowserURL:       "https://dashboard.stripe.com/stripecli/confirm_auth?t=cliauth_secret",
				PollURL:          pollURL,
				VerificationCode: "dinosaur-pineapple-polkadot",
			}
			json.NewEncoder(w).Encode(expectedLinks)
		case "/stripecli/auth/cliauth_123":
			require.Equal(t, "cliauth_secret", r.URL.Query().Get("secret"))

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			data := []byte(`{"redeemed":  true, "account_id": "acct_123", "testmode_key_secret": "sk_test_1234", "account_display_name": "test_disp_name"}`)
			fmt.Println(string(data))
			w.Write(data)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	pollURL = fmt.Sprintf("%s%s", ts.URL, "/stripecli/auth/cliauth_123?secret=cliauth_secret")

	input := strings.NewReader("\n")
	err := Login(ts.URL, c, input)
	require.NoError(t, err)
}

func TestGetLinks(t *testing.T) {
	expectedLinks := Links{
		BrowserURL:       "https://stripe.com/browser",
		PollURL:          "https://stripe.com/poll",
		VerificationCode: "dinosaur-pineapple-polkadot",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		require.NoError(t, r.ParseForm())
		require.Equal(t, "test", r.PostFormValue("device_name"))

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedLinks)
	}))
	defer ts.Close()

	links, err := getLinks(ts.URL, "test")
	require.NoError(t, err)
	require.Equal(t, expectedLinks, *links)
}

func TestGetLinksHTTPStatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	links, err := getLinks(ts.URL, "test")
	require.EqualError(t, err, "unexpected http status code: 500 ")
	require.Empty(t, links)
}

func TestGetLinksRequestError(t *testing.T) {
	// Immediately close the HTTP server so that the request fails.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	links, err := getLinks(ts.URL, "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect: connection refused")
	require.Empty(t, links)
}

func TestGetLinksParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		badLinks := make(map[string]int)
		badLinks["browser_url"] = 10
		json.NewEncoder(w).Encode(badLinks)
	}))
	defer ts.Close()

	links, err := getLinks(ts.URL, "test")
	require.EqualError(t, err, "json: cannot unmarshal number into Go struct field Links.browser_url of type string")
	require.Empty(t, links)
}
