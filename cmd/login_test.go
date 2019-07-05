package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-cli/login"
	"github.com/stripe/stripe-cli/profile"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

)

func TestLogin(t *testing.T) {
	configFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := profile.Profile{
		Color:       "auto",
		ConfigFile:  configFile,
		LogLevel:  "info",
		ProfileName: "tests",
		DeviceName: "st-testing",
	}



	var pollUrl string
	var browserUrl string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "auth") {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			expectedLinks := login.Links{
				BrowserURL:       browserUrl,
				PollURL:          pollUrl,
				VerificationCode: "dinosaur-pineapple-polkadot",
			}
			json.NewEncoder(w).Encode(expectedLinks)
		}
		if strings.Contains(r.URL.Path,"browser") {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<HTML></HTML>"))

		}
		if strings.Contains(r.URL.Path,"poll") {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			data := []byte(`{"redeemed":  true, "account_id": "acct_123", "testmode_key_secret": "sk_test_1234"}`)
			fmt.Println(string(data))
			w.Write(data)
		}
	}))
	defer ts.Close()

	authUrl := fmt.Sprintf( "%s%s",  ts.URL, "/auth")
	pollUrl = fmt.Sprintf( "%s%s",  ts.URL, "/poll")
	browserUrl = fmt.Sprintf( "%s%s",  ts.URL, "/browser")

	err := login.Login(authUrl, p)
	assert.NoError(t, err)
}



