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
	"github.com/stripe/stripe-cli/pkg/spec"
)

func TestNewOperationCmd(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodGet, map[string]string{}, map[string][]spec.StripeEnumValue{}, &config.Config{}, false)

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

func TestNewOperationCmd_NumberType(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	oc := NewOperationCmd(parentCmd, "create", "/v1/test", http.MethodPost, map[string]string{
		"percentage":   "number",
		"percent_off":  "number",
		"string_param": "string",
		"int_param":    "integer",
		"bool_param":   "boolean",
	}, map[string][]spec.StripeEnumValue{}, &config.Config{}, false)

	// Check that number type parameters create string flags
	_, err := oc.Cmd.Flags().GetString("percentage")
	require.NoError(t, err, "percentage flag should exist as string flag")

	_, err = oc.Cmd.Flags().GetString("percent-off")
	require.NoError(t, err, "percent-off flag should exist as string flag")

	// Verify other types still work correctly
	_, err = oc.Cmd.Flags().GetString("string-param")
	require.NoError(t, err)

	_, err = oc.Cmd.Flags().GetInt("int-param")
	require.NoError(t, err)

	_, err = oc.Cmd.Flags().GetBool("bool-param")
	require.NoError(t, err)
}

func TestRunOperationCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/bars/bar_123", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, 5, len(vals))
		require.Equal(t, vals["param1"][0], "value1")
		require.Equal(t, vals["param2"][0], "value2")
		require.Equal(t, vals["param_with_underscores"][0], "some_value")
		require.Equal(t, vals["param[with][dots]"][0], "some_other_value")
		require.Equal(t, vals["param_array[]"], []string{"data1", "data2"})
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
		"param.with.dots":        "string",
		"param_array":            "array",
	}, map[string][]spec.StripeEnumValue{}, &config.Config{
		Profile: profile,
	}, false)
	oc.APIBaseURL = ts.URL

	oc.Cmd.Flags().Set("param1", "value1")
	oc.Cmd.Flags().Set("param2", "value2")
	oc.Cmd.Flags().Set("param-with-underscores", "some_value")
	oc.Cmd.Flags().Set("param.with.dots", "some_other_value")
	oc.Cmd.Flags().Set("param-array", "data1")
	oc.Cmd.Flags().Set("param-array", "data2")

	parentCmd.SetArgs([]string{"foo", "bar_123"})
	err := parentCmd.ExecuteContext(context.Background())

	require.NoError(t, err)
}

func TestRunOperationCmd_ExtraParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := io.ReadAll(r.Body)
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
	}, map[string][]spec.StripeEnumValue{}, &config.Config{
		Profile: profile,
	}, false)
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
	}, map[string][]spec.StripeEnumValue{}, &config.Config{}, false)

	err := oc.runOperationCmd(oc.Cmd, []string{"bar_123", "param1=value1", "param2=value2"})

	require.Error(t, err, "your API key has not been configured. Use `stripe login` to set your API key")
}

func TestConstructParamFromDot(t *testing.T) {
	param := constructParamFromDot("shipping.address.line1")
	require.Equal(t, "shipping[address][line1]", param)
}
