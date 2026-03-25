package resource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
)

func TestNewOperationCmd(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodGet, map[string]string{}, map[string][]string{}, &config.Config{}, false, "")

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
	}, map[string][]string{}, &config.Config{}, false, "")

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
	}, map[string][]string{}, &config.Config{
		Profile: profile,
	}, false, "")
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
	}, map[string][]string{}, &config.Config{
		Profile: profile,
	}, false, "")
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
	}, map[string][]string{}, &config.Config{}, false, "")

	err := oc.runOperationCmd(oc.Cmd, []string{"bar_123", "param1=value1", "param2=value2"})

	require.Error(t, err, "your API key has not been configured. Use `stripe login` to set your API key")
}

func TestRunOperationCmd_DryRun(t *testing.T) {
	serverCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	profile := config.Profile{APIKey: "sk_test_1234567890abcdef"}
	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodPost, map[string]string{
		"param1": "string",
	}, map[string][]string{}, &config.Config{Profile: profile}, false, "")
	oc.APIBaseURL = ts.URL

	var buf bytes.Buffer
	oc.Cmd.SetOut(&buf)
	oc.Cmd.Flags().Set("param1", "value1")
	oc.Cmd.Flags().Set("dry-run", "true")

	err := oc.runOperationCmd(oc.Cmd, []string{"bar_123"})

	require.NoError(t, err)
	require.False(t, serverCalled, "HTTP server should not be called during dry-run")

	var result requests.DryRunOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	// "sk_test_1234567890abcdef" (24 chars) redacts to "sk_test_************cdef"
	require.Equal(t, requests.DryRunOutput{DryRun: requests.DryRunDetails{
		Method: "POST",
		URL:    ts.URL + "/v1/bars/bar_123",
		Params: map[string]interface{}{"param1": "value1"},
		Headers: map[string]string{
			"Authorization": "Bearer sk_test_************cdef",
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		AuthAvailable:        true,
		RequiresConfirmation: false,
	}}, result)
}

func TestRunOperationCmd_DryRun_NoAPIKey(t *testing.T) {
	viper.Reset()

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodPost, map[string]string{}, map[string][]string{}, &config.Config{}, false, "")

	var buf bytes.Buffer
	oc.Cmd.SetOut(&buf)
	oc.Cmd.Flags().Set("dry-run", "true")

	err := oc.runOperationCmd(oc.Cmd, []string{"bar_123"})

	require.NoError(t, err, "dry-run should succeed even without an API key")

	var result requests.DryRunOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	require.Equal(t, requests.DryRunOutput{DryRun: requests.DryRunDetails{
		Method:               "POST",
		URL:                  "https://api.stripe.com/v1/bars/bar_123",
		Params:               map[string]interface{}{},
		Headers:              map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, result)
}

// assertDryRunParityV1 checks that structured dry-run params are semantically
// consistent with the URL-encoded body received by a test server.
// One-directional: every param in dry-run must appear in the server request.
func assertDryRunParityV1(t *testing.T, serverBody []byte, dryRunParams map[string]interface{}) {
	t.Helper()
	serverVals, err := url.ParseQuery(string(serverBody))
	require.NoError(t, err)
	flattenAndAssert(t, serverVals, dryRunParams, "")
}

func flattenAndAssert(t *testing.T, serverVals url.Values, params map[string]interface{}, prefix string) {
	t.Helper()
	for k, v := range params {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "[" + k + "]"
		}
		switch val := v.(type) {
		case string:
			require.Equal(t, val, serverVals.Get(fullKey), "param %q mismatch", fullKey)
		case map[string]interface{}:
			flattenAndAssert(t, serverVals, val, fullKey)
		case []interface{}:
			for _, item := range val {
				require.Contains(t, serverVals[fullKey+"[]"], fmt.Sprint(item))
			}
		default:
			require.Equal(t, fmt.Sprint(val), serverVals.Get(fullKey), "param %q mismatch", fullKey)
		}
	}
}

func TestRunOperationCmd_DryRunParity_V1(t *testing.T) {
	var capturedPath string
	var capturedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	viper.Reset()
	profile := config.Profile{APIKey: "sk_test_1234"}
	propFlags := map[string]string{
		"param1":    "string",
		"int-param": "integer",
		"arr-param": "array",
	}

	newOC := func(dryRun bool) (*OperationCmd, *cobra.Command) {
		parentCmd := &cobra.Command{Annotations: make(map[string]string)}
		oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodPost,
			propFlags, map[string][]string{}, &config.Config{Profile: profile}, false, "")
		oc.APIBaseURL = ts.URL
		oc.Cmd.Flags().Set("param1", "value1")
		oc.Cmd.Flags().Set("int-param", "42")
		oc.Cmd.Flags().Set("arr-param", "x")
		oc.Cmd.Flags().Set("arr-param", "y")
		oc.Cmd.Flags().Set("data", "metadata[env]=staging")
		oc.Cmd.Flags().Set("data", "metadata[version]=2")
		if dryRun {
			oc.Cmd.Flags().Set("dry-run", "true")
		}
		parentCmd.SetArgs([]string{"foo", "bar_123"})
		return oc, parentCmd
	}

	// --- LIVE RUN ---
	_, liveCmd := newOC(false)
	require.NoError(t, liveCmd.ExecuteContext(t.Context()))

	// --- DRY-RUN ---
	dryOC, dryCmd := newOC(true)
	var buf bytes.Buffer
	dryOC.Cmd.SetOut(&buf)
	require.NoError(t, dryCmd.ExecuteContext(t.Context()))

	var dryOut requests.DryRunOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &dryOut))

	require.Equal(t, "/v1/bars/bar_123", capturedPath)
	require.Contains(t, dryOut.DryRun.URL, "/v1/bars/bar_123")
	assertDryRunParityV1(t, capturedBody, dryOut.DryRun.Params)
}

func TestRunOperationCmd_DryRunParity_V2(t *testing.T) {
	var capturedBody []byte
	var capturedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	viper.Reset()
	profile := config.Profile{APIKey: "sk_test_1234"}
	jsonData := `{"event_name": "foo", "value": 100}`

	newOC := func(dryRun bool) (*OperationCmd, *cobra.Command) {
		parentCmd := &cobra.Command{Annotations: make(map[string]string)}
		oc := NewOperationCmd(parentCmd, "create", "/v2/billing/meter_events",
			http.MethodPost, map[string]string{}, map[string][]string{},
			&config.Config{Profile: profile}, false, "")
		oc.APIBaseURL = ts.URL
		oc.Cmd.Flags().Set("data", jsonData)
		if dryRun {
			oc.Cmd.Flags().Set("dry-run", "true")
		}
		parentCmd.SetArgs([]string{"create"})
		return oc, parentCmd
	}

	// --- LIVE RUN ---
	_, liveCmd := newOC(false)
	require.NoError(t, liveCmd.ExecuteContext(t.Context()))

	// --- DRY-RUN ---
	dryOC, dryCmd := newOC(true)
	var buf bytes.Buffer
	dryOC.Cmd.SetOut(&buf)
	require.NoError(t, dryCmd.ExecuteContext(t.Context()))

	var dryOut requests.DryRunOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &dryOut))

	require.Equal(t, "", capturedQuery)
	var liveParams map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &liveParams))

	require.Equal(t, liveParams, dryOut.DryRun.Params)
	require.NotContains(t, dryOut.DryRun.URL, "?")
}

func TestConstructParamFromDot(t *testing.T) {
	param := constructParamFromDot("shipping.address.line1")
	require.Equal(t, "shipping[address][line1]", param)
}

func TestNewOperationCmd_WithServerURL(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	serverURL := "https://files.stripe.com/"
	oc := NewOperationCmd(parentCmd, "pdf", "/v1/quotes/{quote}/pdf", http.MethodGet, map[string]string{}, map[string][]string{}, &config.Config{}, false, serverURL)

	require.Equal(t, "pdf", oc.Name)
	require.Equal(t, "/v1/quotes/{quote}/pdf", oc.Path)
	require.Equal(t, serverURL, oc.APIBaseURL)

	// Verify the flag default value is also set
	flag := oc.Cmd.Flags().Lookup("api-base")
	require.NotNil(t, flag)
	require.Equal(t, serverURL, flag.DefValue)
}
