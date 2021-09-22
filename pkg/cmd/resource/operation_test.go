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

func TestNewOperationCmd(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodGet, map[string]string{}, &config.Config{})

	require.Equal(t, "foo", oc.Name)
	require.Equal(t, "/v1/bars/{id}", oc.Path)
	require.Equal(t, "GET", oc.HTTPVerb)
	require.Equal(t, []string{"{id}"}, oc.URLParams)
	require.True(t, parentCmd.HasSubCommands())
	val, ok := parentCmd.Annotations["foo"]
	require.True(t, ok)
	require.Equal(t, "operation", val)
	require.Contains(t, oc.Cmd.UsageTemplate(), "<id>")
}

func TestRunOperationCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/bars/bar_123", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, 3, len(vals))
		require.Equal(t, vals["param1"][0], "value1")
		require.Equal(t, vals["param2"][0], "value2")
		require.Equal(t, vals["param_with_underscores"][0], "some_value")
	}))
	defer ts.Close()

	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	profile := config.Profile{
		APIKey: "sk_test_1234",
	}
	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodPost, map[string]string{
		"param1":                 "string",
		"param2":                 "string",
		"param_with_underscores": "string",
	}, &config.Config{
		Profile: profile,
	})
	oc.APIBaseURL = ts.URL

	oc.Cmd.Flags().Set("param1", "value1")
	oc.Cmd.Flags().Set("param2", "value2")
	oc.Cmd.Flags().Set("param-with-underscores", "some_value")

	parentCmd.SetArgs([]string{"foo", "bar_123"})
	err := parentCmd.ExecuteContext(context.Background())

	require.NoError(t, err)
}

func TestRunOperationCmd_ExtraParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/bars/bar_123", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, vals["param1"][0], "value1")
		require.Equal(t, vals["shipping[address][line1]"][0], "123 Main St")
		require.Equal(t, vals["shipping[name]"][0], "name")
	}))
	defer ts.Close()

	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	profile := config.Profile{
		APIKey: "sk_test_1234",
	}
	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodPost, map[string]string{
		"param1": "string",
	}, &config.Config{
		Profile: profile,
	})
	oc.APIBaseURL = ts.URL

	oc.Cmd.Flags().Set("param1", "value1")
	oc.Cmd.Flags().Set("data", "shipping[address][line1]=123 Main St")
	oc.Cmd.Flags().Set("data", "shipping[name]=name")

	parentCmd.SetArgs([]string{"foo", "bar_123"})
	err := parentCmd.ExecuteContext(context.Background())

	require.NoError(t, err)
}

func TestRunOperationCmd_NoAPIKey(t *testing.T) {
	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodPost, map[string]string{
		"param1": "string",
		"param2": "string",
	}, &config.Config{})

	err := oc.runOperationCmd(oc.Cmd, []string{"bar_123", "param1=value1", "param2=value2"})

	require.Error(t, err, "your API key has not been configured. Use `stripe login` to set your API key")
}
