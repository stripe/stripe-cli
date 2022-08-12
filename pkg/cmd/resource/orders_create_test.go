package resource

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestRunOrdersCreateCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/orders", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, 4, len(vals))
	}))
	defer ts.Close()

	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	profile := config.Profile{
		APIKey: "sk_test_1234",
	}
	erc := NewOrdersCreateCmd(parentCmd, &config.Config{Profile: profile})
	erc.opCmd.APIBaseURL = ts.URL

	parentCmd.SetArgs([]string{"create",
		"--currency", "usd",
		"--line-items[][product]", "dummyProduct",
		"--line-items[][quantity]", "1",
		"--automatic-tax[enabled]", "true",
	})
	err := parentCmd.ExecuteContext(context.Background())

	require.NoError(t, err)
}
