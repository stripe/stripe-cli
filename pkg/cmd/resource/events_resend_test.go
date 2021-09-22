package resource

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestRunEventsResendCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/events/evt_123/retry", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, 1, len(vals))
		require.Equal(t, vals["for_stripecli"][0], "true")
	}))
	defer ts.Close()

	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	profile := config.Profile{
		APIKey: "sk_test_1234",
	}
	erc := NewEventsResendCmd(parentCmd, &config.Config{Profile: profile})
	erc.opCmd.APIBaseURL = ts.URL

	parentCmd.SetArgs([]string{"resend", "evt_123"})
	err := parentCmd.ExecuteContext(context.Background())

	require.NoError(t, err)
}

func TestRunEventsResendCmd_WithWebhookEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/events/evt_123/retry", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, 1, len(vals))
		require.Equal(t, vals["webhook_endpoint"][0], "we_123")
	}))
	defer ts.Close()

	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	profile := config.Profile{
		APIKey: "sk_test_1234",
	}
	erc := NewEventsResendCmd(parentCmd, &config.Config{Profile: profile})
	erc.opCmd.APIBaseURL = ts.URL

	erc.opCmd.Cmd.Flags().Set("webhook-endpoint", "we_123")

	parentCmd.SetArgs([]string{"resend", "evt_123"})
	err := parentCmd.ExecuteContext(context.Background())

	require.NoError(t, err)
}
