package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/profile"
)

func TestLogin(t *testing.T) {
	configFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := profile.Profile{
		Color:       "auto",
		ConfigFile:  configFile,
		LogLevel:    "info",
		ProfileName: "tests",
		DeviceName:  "st-testing",
	}

	var pollURL string
	var browserURL string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "auth") {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			expectedLinks := login.Links{
				BrowserURL:       browserURL,
				PollURL:          pollURL,
				VerificationCode: "dinosaur-pineapple-polkadot",
			}
			json.NewEncoder(w).Encode(expectedLinks)
		}
		if strings.Contains(r.URL.Path, "browser") {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<HTML></HTML>"))

		}
		if strings.Contains(r.URL.Path, "poll") {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			data := []byte(`{"redeemed":  true, "account_id": "acct_123", "testmode_key_secret": "sk_test_1234"}`)
			fmt.Println(string(data))
			w.Write(data)
		}
	}))
	defer ts.Close()

	authURL := fmt.Sprintf("%s%s", ts.URL, "/auth")
	pollURL = fmt.Sprintf("%s%s", ts.URL, "/poll")
	browserURL = fmt.Sprintf("%s%s", ts.URL, "/browser")

	err := login.Login(authURL, p)
	assert.NoError(t, err)
}
