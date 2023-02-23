package login

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/pkg/open"
)

type stubInputReader struct {
}

func (r stubInputReader) scanln(ch chan int) {
	ch <- 1
	close(ch)
}

func TestLogin(t *testing.T) {
	if os.Getenv("OPEN_URL") == "1" {
		os.Exit(0)
		return
	}

	didOpenBrowser := false
	openBrowser = func(string) error {
		didOpenBrowser = true
		return nil
	}

	defer func() { openBrowser = open.Browser }()

	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	viper.SetConfigFile(profilesFile)

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

	links, err := GetLinks(context.Background(), ts.URL, p.DeviceName)
	require.NoError(t, err)
	configurer := keys.NewRAKConfigurer(c, afero.NewOsFs())
	rt := keys.NewRAKTransfer(configurer)
	auth := NewAuthenticator(rt)
	auth.asyncInputReader = stubInputReader{}

	err = auth.Login(context.Background(), links)
	require.NoError(t, err)
	assert.Equal(t, true, didOpenBrowser)

	viper.Reset()
}

type noInputReader struct {
}

func (r noInputReader) scanln(ch chan int) {
}

func TestLoginNoInput(t *testing.T) {
	if os.Getenv("OPEN_URL") == "1" {
		os.Exit(0)
		return
	}

	didOpenBrowser := false
	openBrowser = func(string) error {
		didOpenBrowser = true
		return nil
	}

	defer func() { openBrowser = open.Browser }()

	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	viper.SetConfigFile(profilesFile)

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

	links, err := GetLinks(context.Background(), ts.URL, p.DeviceName)
	require.NoError(t, err)
	configurer := keys.NewRAKConfigurer(c, afero.NewOsFs())
	rt := keys.NewRAKTransfer(configurer)
	auth := NewAuthenticator(rt)
	auth.asyncInputReader = noInputReader{}

	err = auth.Login(context.Background(), links)
	require.NoError(t, err)
	assert.Equal(t, false, didOpenBrowser)

	viper.Reset()
}
