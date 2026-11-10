package fixtures

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/afero"
	"github.com/tidwall/gjson"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const testFixture = `
{
	"_meta": {
		"template_version": 0
	},
	"fixtures": [
		{
			"name": "cust_bender",
			"path": "/v1/customers",
			"method": "post",
			"idempotency_key": "create_cust_bender",
			"params": {
				"name": "Bender Bending Rodriguez",
				"email": "bender@planex.com",
				"phone": "${.env:PHONE_NO_CLASH|+1234567890}",
				"address": {
					"line1": "1 Planet Express St",
					"city": "New New York"
				}
			}
		},
		{
			"name": "char_bender",
			"path": "/v1/charges",
			"method": "post",
			"params": {
				"customer": "${cust_bender:id}",
				"source": "tok_visa",
				"amount": "100",
				"currency": "${cust_bender:currency|usd}",
				"capture": false
			}
		},
		{
			"name": "capt_bender",
			"path": "/v1/charges/${char_bender:id}/capture",
			"method": "post"
		}
	]
}`

const failureTestFixture = `
{
	"_meta": {
		"template_version": 0
	},
	"fixtures": [
		{
			"name": "charge_expected_failure",
			"expected_error_type": "card_error",
			"path": "/v1/charges",
			"method": "post",
			"params": {
				"source": "tok_chargeDeclined",
				"amount": 100,
				"currency": "usd",
				"description": "(created by Stripe CLI)"
			}
	 	}
	]
}`

const apiKey = "sk_test_1234"
const file = "test_fixture.json"
const customersPath = "/v1/customers"
const chargePath = "/v1/charges"
const capturePath = "/v1/charges/char_12345/capture"

func TestAccountCreationFixtureError(t *testing.T) {
	fxt, err := NewFixtureFromFile(afero.NewMemMapFs(), stripe.NewAPIKeyCredentials(apiKey), "", "", "triggers/account.updated.json", nil, nil, nil, nil, false)
	require.NoError(t, err)
	data := fxt.FixtureData.Requests[0]
	requestErr := requests.RequestError{
		StatusCode: http.StatusBadRequest,
		ErrorType:  "invalid_request_error",
		Message:    "Stripe no longer recommends Accounts v1 for new Connect integrations. Create connected accounts with POST /v2/core/accounts instead.",
		Body:       `{"error":{"message":"Stripe no longer recommends Accounts v1 for new Connect integrations."}}`,
	}

	for _, original := range []error{requestErr, fmt.Errorf("request failed: %w", requestErr)} {
		err := fixtureRequestError(data, original)
		require.ErrorContains(t, err, `fixture "account" requires Accounts v1 account creation support`)
		require.ErrorContains(t, err, "POST /v1/accounts")
		require.ErrorContains(t, err, "custom fixture for Accounts v2 events")
		require.ErrorContains(t, err, "https://docs.stripe.com/cli/fixtures")
		require.ErrorContains(t, err, requestErr.Error())
		var underlying requests.RequestError
		require.ErrorAs(t, err, &underlying)
		require.Equal(t, requestErr, underlying)
	}

	t.Run("other requests", func(t *testing.T) {
		for _, request := range []FixtureRequest{
			{Path: "/v1/accounts/acct_123", Method: "post"},
			{Path: "/v2/core/accounts", Method: "post"},
			{Path: "/v1/customers", Method: "post"},
			{Path: "/v1/accounts", Method: "get"},
		} {
			require.Equal(t, requestErr, fixtureRequestError(request, requestErr))
		}
	})
	t.Run("other errors", func(t *testing.T) {
		for _, original := range []error{
			nil,
			context.Canceled,
			requests.RequestError{StatusCode: http.StatusUnauthorized},
			requests.RequestError{StatusCode: http.StatusInternalServerError},
		} {
			require.Equal(t, original, fixtureRequestError(data, original))
		}
	})
}

func TestMakeRequest(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case customersPath:
			if req.Header.Get("Idempotency-Key") == "" {
				t.Errorf("Idempotency key not sent")
			}

			if err := req.ParseForm(); err != nil {
				t.Error(err)
			}

			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))
		case capturePath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{}, []string{}, []string{}, []string{}, false)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)

	require.NotNil(t, fxt.Responses["cust_bender"])
	require.NotNil(t, fxt.Responses["char_bender"])
	require.NotNil(t, fxt.Responses["capt_bender"])

	require.Equal(t, "cust_12345", fxt.Responses["cust_bender"].Get("id").String())
	require.Equal(t, "char_12345", fxt.Responses["char_bender"].Get("id").String())
	require.True(t, fxt.Responses["char_bender"].Get("charge").Bool())
}

func TestMakeRequestWithStringFixture(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))
		case capturePath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	fxt, err := NewFixtureFromRawString(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, testFixture)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)

	require.NotNil(t, fxt.Responses["cust_bender"])
	require.NotNil(t, fxt.Responses["char_bender"])
	require.NotNil(t, fxt.Responses["capt_bender"])

	require.Equal(t, "cust_12345", fxt.Responses["cust_bender"].Get("id").String())
	require.Equal(t, "char_12345", fxt.Responses["char_bender"].Get("id").String())
	require.True(t, fxt.Responses["char_bender"].Get("charge").Bool())
}

func TestProtectedFixtureHeaders(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	for _, creds := range []stripe.Credentials{
		stripe.NewAPIKeyCredentials("sk_test_dummy_selected"),
		stripe.NewOAKCredentials("oak_dummy_selected", "dummy-context", false),
	} {
		for _, header := range []string{"Authorization", "Stripe-Livemode", "Stripe-Context", "Stripe-Account", "Stripe-Future-Scope"} {
			for _, name := range []string{header, strings.ToLower(header), strings.ToUpper(header), strings.ToLower(header[:1]) + header[1:], strings.ReplaceAll(header, "-", "_"), strings.ToLower(strings.ReplaceAll(header, "-", "_"))} {
				for _, value := range []string{"", "dummy-override-value", "${.env:FIXTURE_HEADER_SENTINEL}"} {
					t.Run(creds.Token+"/"+name+"/"+value, func(t *testing.T) {
						data := FixtureRequest{
							Name: "protected_step", Path: customersPath, Method: "post",
							ExpectedErrorType: "invalid_request_error",
							Headers:           map[string]string{name: value},
						}
						fxt := &Fixture{Credentials: creds, BaseURL: server.URL, FixtureData: FixtureData{Requests: []FixtureRequest{data}}}
						_, err := fxt.Execute(context.Background(), "")
						require.ErrorContains(t, err, data.Name)
						require.ErrorContains(t, err, name)
						require.ErrorContains(t, err, "CLI credentials and account selection")
						assert.NotContains(t, err.Error(), "dummy-override-value")
						assert.NotContains(t, err.Error(), "FIXTURE_HEADER_SENTINEL")
						category, ok := errorcategory.Get(err)
						require.True(t, ok)
						require.Equal(t, errorcategory.UserInput, category)
						_, err = fxt.makeRequest(context.Background(), data, "")
						require.ErrorContains(t, err, name)
					})
				}
			}
			data := FixtureRequest{Name: "duplicates", Path: customersPath, Method: "post", Headers: map[string]string{header: "", strings.ToLower(header): "dummy-override-value"}}
			fxt := &Fixture{Credentials: creds, BaseURL: server.URL, FixtureData: FixtureData{Requests: []FixtureRequest{data}}}
			_, err := fxt.Execute(context.Background(), "")
			require.Error(t, err)
			_, err = fxt.makeRequest(context.Background(), data, "")
			require.Error(t, err)
		}
	}
	require.Zero(t, calls.Load())
}

func TestAccountScopeHeadersMayReferenceEarlierResponses(t *testing.T) {
	var accountHeaders, contextHeaders []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountHeaders = append(accountHeaders, r.Header.Get("Stripe-Account"))
		contextHeaders = append(contextHeaders, r.Header.Get("Stripe-Context"))
		w.Write([]byte(`{"id":"acct_dummy_created"}`))
	}))
	defer server.Close()

	fxt, err := NewFixtureFromRawString(afero.NewMemMapFs(), stripe.NewAPIKeyCredentials(apiKey), "", server.URL, `{"fixtures":[
		{"name":"create_account","path":"/v1/accounts","method":"post"},
		{"name":"scoped_customer","path":"/v1/customers","method":"post","headers":{"Stripe-Account":"${create_account:id}"}},
		{"name":"scoped_context","path":"/v2/core/events","method":"get","headers":{"Stripe-Context":"${create_account:id}"}}]}`)
	require.NoError(t, err)
	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, []string{"", "acct_dummy_created", ""}, accountHeaders)
	assert.Equal(t, []string{"", "", "acct_dummy_created"}, contextHeaders)

	createAccount := fxt.FixtureData.Requests[0]
	for name, headers := range map[string]map[string]string{
		"own step":       {"Stripe-Account": "${scoped:id}"},
		"missing step":   {"Stripe-Account": "${absent:id}"},
		"default value":  {"Stripe-Account": "${create_account:id|acct_fallback}"},
		"literal prefix": {"Stripe-Account": "acct_${create_account:id}"},
		"env lookup":     {"Stripe-Context": "${.env:FIXTURE_CONTEXT}"},
	} {
		t.Run(name, func(t *testing.T) {
			data := FixtureRequest{Name: "scoped", Path: customersPath, Method: "post", Headers: headers}
			blocked := &Fixture{Credentials: stripe.NewAPIKeyCredentials(apiKey), BaseURL: server.URL,
				FixtureData: FixtureData{Requests: []FixtureRequest{createAccount, data}}}
			_, err := blocked.Execute(context.Background(), "")
			require.ErrorContains(t, err, "protected header")
		})
	}

	t.Run("later step", func(t *testing.T) {
		data := FixtureRequest{Name: "scoped", Path: customersPath, Method: "post", Headers: map[string]string{"Stripe-Account": "${create_account:id}"}}
		blocked := &Fixture{Credentials: stripe.NewAPIKeyCredentials(apiKey), BaseURL: server.URL,
			FixtureData: FixtureData{Requests: []FixtureRequest{data, createAccount}}}
		_, err := blocked.Execute(context.Background(), "")
		require.ErrorContains(t, err, "protected header")
	})

	t.Run("skipped source step", func(t *testing.T) {
		scoped := FixtureRequest{Name: "scoped", Path: customersPath, Method: "post", Headers: map[string]string{"Stripe-Account": "${create_account:id}"}}
		blocked := &Fixture{Credentials: stripe.NewAPIKeyCredentials(apiKey), BaseURL: server.URL, Skip: []string{"create_account"},
			FixtureData: FixtureData{Requests: []FixtureRequest{createAccount, scoped}}}
		_, err := blocked.Execute(context.Background(), "")
		require.ErrorContains(t, err, "protected header")
	})
}

func TestFixtureHeaderValidationBeforeEnvironmentParameters(t *testing.T) {
	t.Setenv("FIXTURE_PROCESS_SENTINEL", "dummy-process-value")
	dir := t.TempDir()
	// godotenv reads .env from the real working directory, so afero cannot
	// stand in here; t.Chdir restores the directory and forbids t.Parallel.
	t.Chdir(dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("FIXTURE_FILE_SENTINEL=dummy-file-value\n"), 0o600))

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	const raw = `{"fixtures":[
		{"name":"benign","path":"/v1/customers","method":"post"},
		{"name":"environment_step","path":"/v1/customers","method":"post","expected_error_type":"invalid_request_error",
		 "headers":{"Authorization":"Bearer dummy-untrusted"},
		 "params":{"description":"${.env:FIXTURE_PROCESS_SENTINEL}",
		           "metadata":{"value":"${.env:FIXTURE_FILE_SENTINEL}","missing":"${missing:response}"}}}]}`
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, file, []byte(raw), 0o600))
	fromFile, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", server.URL, file, nil, nil, nil, nil, false)
	require.NoError(t, err)
	fromRaw, err := NewFixtureFromRawString(fs, stripe.NewAPIKeyCredentials(apiKey), "", server.URL, raw)
	require.NoError(t, err)
	for _, fxt := range []*Fixture{fromFile, fromRaw} {
		_, err := fxt.Execute(context.Background(), "")
		require.ErrorContains(t, err, "Authorization")
		for _, value := range []string{"dummy-untrusted", "dummy-process-value", "dummy-file-value", "FIXTURE_PROCESS_SENTINEL", "FIXTURE_FILE_SENTINEL"} {
			assert.NotContains(t, err.Error(), value)
		}
		require.Empty(t, fxt.Responses)
	}
	require.Zero(t, calls.Load())

	fromRaw.Skip = []string{"environment_step"}
	_, err = fromRaw.Execute(context.Background(), "")
	require.NoError(t, err)
	require.EqualValues(t, 1, calls.Load())
}

func TestAllowedFixtureHeadersPreserveCredentials(t *testing.T) {
	t.Setenv("FIXTURE_CUSTOM_VALUE", "dummy-substitution")
	for _, creds := range []stripe.Credentials{
		stripe.NewAPIKeyCredentials("sk_test_dummy_selected"),
		stripe.NewOAKCredentials("oak_dummy_selected", "dummy-context", false),
	} {
		for _, account := range []string{"", "dummy-connected-account"} {
			t.Run(creds.Token+"/"+account, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					assert.Equal(t, "Bearer "+creds.Token, r.Header.Get("Authorization"))
					if creds.OAKLivemode != nil {
						assert.Equal(t, "false", r.Header.Get("Stripe-Livemode"))
					} else {
						assert.Empty(t, r.Header.Get("Stripe-Livemode"))
					}
					expectedScope := make(http.Header)
					creds.ApplyAccountContextHeaders(expectedScope, account, "")
					assert.Equal(t, expectedScope.Get("Stripe-Context"), r.Header.Get("Stripe-Context"))
					assert.Equal(t, expectedScope.Get("Stripe-Account"), r.Header.Get("Stripe-Account"))
					assert.Equal(t, "dummy-substitution", r.Header.Get("X-Custom"))
					assert.Equal(t, "2024-09-30.acacia", r.Header.Get("Stripe-Version"))
					assert.Equal(t, stripe.V1ContentType, r.Header.Get("Content-Type"))
					assert.Equal(t, "dummy-idempotency", r.Header.Get("Idempotency-Key"))
					assert.NoError(t, r.ParseForm())
					assert.Equal(t, "dummy-substitution", r.Form.Get("description"))
					w.Write([]byte(`{}`))
				}))
				defer server.Close()
				fxt, err := NewFixtureFromRawString(afero.NewMemMapFs(), creds, account, server.URL, `{
					"fixtures":[{"name":"allowed","path":"/v1/customers","method":"post",
					"headers":{"X-Custom":"${.env:FIXTURE_CUSTOM_VALUE}","Stripe-Version":"2024-09-30.acacia",
					"Content-Type":"application/x-www-form-urlencoded","Idempotency-Key":"dummy-idempotency"},
					"params":{"description":"${.env:FIXTURE_CUSTOM_VALUE}"}}]}`)
				require.NoError(t, err)
				_, err = fxt.Execute(context.Background(), "2020-08-27")
				require.NoError(t, err)
				require.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

func TestWithSkipMakeRequest(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{"char_bender", "capt_bender"}, []string{}, []string{}, []string{}, false)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)

	require.True(t, fxt.Responses["cust_bender"].Exists())
	require.False(t, fxt.Responses["char_bender"].Exists())
	require.False(t, fxt.Responses["capt_bender"].Exists())
}

func TestMakeRequestWithOverride(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("Failure with request body: %s", err)
		}

		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))

			require.True(t, strings.Contains(string(body), "name=Fry"))
			require.False(t, strings.Contains(string(body), "name=Bender"))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))

			require.True(t, strings.Contains(string(body), "amount=3000"))
			require.False(t, strings.Contains(string(body), "amount=100"))
		case capturePath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{}, []string{"cust_bender:name=Fry", "char_bender:amount=3000"}, []string{}, []string{}, false)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}

func TestMakeRequestWithAdd(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("Failure with request body: %s", err)
		}

		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
			require.True(t, strings.Contains(string(body), "birthdate=2996-09-04"))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))
			require.True(t, strings.Contains(string(body), "receipt_email=prof.farnsworth%40planex.com"))
		case capturePath:
			res.Write([]byte(`{}`))
			require.True(t, strings.Contains(string(body), "statement_descriptor=Fuel%3A+Beer"))
			require.True(t, strings.Contains(string(body), "nested1[nested2][nested3]=nestedValue"))

		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(
		fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file,
		[]string{}, []string{}, []string{
			"cust_bender:birthdate=2996-09-04",
			"char_bender:receipt_email=prof.farnsworth@planex.com",
			"capt_bender:statement_descriptor=Fuel: Beer",
			"capt_bender:nested1.nested2.nested3=nestedValue",
		},
		[]string{},
		false,
	)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}

func TestMakeRequestWithRemove(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("Failure with request body: %s", err)
		}

		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))

			require.False(t, strings.Contains(string(body), "phone"))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))

			require.False(t, strings.Contains(string(body), "capture"))
		case capturePath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(
		fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{}, []string{},
		[]string{}, []string{"cust_bender:phone", "char_bender:capture"},
		false)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}

func TestMakeRequestExpectedFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(402)
		res.Write([]byte(`{"error": {"type": "card_error"}}`))
	}))

	defer func() { ts.Close() }()
	afero.WriteFile(fs, "failured_test_fixture.json", []byte(failureTestFixture), os.ModePerm)
	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, "failured_test_fixture.json", []string{}, []string{}, []string{}, []string{}, false)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
	require.NotNil(t, fxt.Responses["charge_expected_failure"])
}

func TestMakeRequestUnexpectedFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(500)
		res.Write([]byte(`{"error": "Internal Failure Occurred."}`))
	}))

	defer func() { ts.Close() }()
	afero.WriteFile(fs, "failured_test_fixture.json", []byte(failureTestFixture), os.ModePerm)
	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, "failured_test_fixture.json", []string{}, []string{}, []string{}, []string{}, false)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NotNil(t, err)
}

func TestUpdateEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	fxt := Fixture{
		Fs: fs,
		Responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "char_12345"}`),
			"cust_bender": gjson.Parse(`{"id": "cust_12345"}`),
		},
	}

	wd, _ := os.Getwd()
	fs.MkdirAll(wd, os.ModePerm)
	afero.WriteFile(fs, filepath.Join(wd, ".env"), []byte(``), os.ModePerm)

	envMapping := map[string]string{
		"CHAR_ID": "${char_bender:id}",
		"CUST_ID": "${char_bender:id}",
	}

	err := fxt.updateEnv(envMapping)
	assert.Nil(t, err)

	expected := `CHAR_ID="char_12345"
CUST_ID="char_12345"`
	output, _ := afero.ReadFile(fs, filepath.Join(wd, ".env"))
	assert.Equal(t, expected, string(output))
}

func TestUpdateEnvRefusesSymlink(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	victimFile := filepath.Join(tempDir, "victim")
	require.NoError(t, os.WriteFile(victimFile, []byte("ORIGINAL=1\n"), 0o644))
	require.NoError(t, os.Symlink(victimFile, filepath.Join(tempDir, ".env")))

	fxt := Fixture{
		Fs: afero.NewOsFs(),
		Responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "char_12345"}`),
		},
	}

	err := fxt.updateEnv(map[string]string{
		"CHAR_ID": "${char_bender:id}",
	})
	require.ErrorContains(t, err, "symlink")

	victimContents, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	require.Equal(t, "ORIGINAL=1\n", string(victimContents))
}

func TestMakeRequestRejectsPathTraversal(t *testing.T) {
	fs := afero.NewMemMapFs()
	serverCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		serverCalled = true
	}))
	defer ts.Close()

	fxt := &Fixture{
		Fs:          fs,
		Credentials: stripe.NewAPIKeyCredentials("rk_test_1234"),
		BaseURL:     ts.URL,
		Responses:   make(map[string]gjson.Result),
	}

	tests := []struct {
		name string
		path string
	}{
		{"dot-segment escape", "/v2x/../v1.41/containers/create"},
		{"non-API prefix", "/docker/containers"},
		{"version without slash", "/v2x/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fxt.makeRequest(context.Background(), FixtureRequest{
				Name:   "test",
				Path:   tt.path,
				Method: "post",
				Params: map[string]interface{}{},
			}, "")
			require.Error(t, err)
			require.Contains(t, err.Error(), "not a valid Stripe API path")
		})
	}

	require.False(t, serverCalled, "no request should have been sent for invalid paths")
}

func TestMakeRequestRejectsAPIBaseOutsideStripe(t *testing.T) {
	fs := afero.NewMemMapFs()
	serverCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		serverCalled = true
	}))
	defer ts.Close()

	fxt := &Fixture{
		Fs:          fs,
		Credentials: stripe.NewAPIKeyCredentials("sk_test_1234"),
		BaseURL:     ts.URL,
		Responses:   make(map[string]gjson.Result),
	}

	_, err := fxt.makeRequest(context.Background(), FixtureRequest{
		Name:    "test",
		Path:    "/v1/customers",
		Method:  "post",
		Params:  map[string]interface{}{},
		APIBase: ts.URL,
	}, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not an allowed Stripe API URL")

	require.False(t, serverCalled, "no request should have been sent to the fixture's api_base")
}

func TestBundledFixtureAPIBasesAreAllowed(t *testing.T) {
	entries, err := triggers.ReadDir("triggers")
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	checked := 0
	for _, entry := range entries {
		fxt, err := NewFixtureFromFile(afero.NewMemMapFs(), stripe.NewAPIKeyCredentials(apiKey), "", "", "triggers/"+entry.Name(), nil, nil, nil, nil, false)
		require.NoError(t, err, entry.Name())

		for _, request := range fxt.FixtureData.Requests {
			assert.NoError(t, fxt.validateRequestHeaders(request), "%s: %s", entry.Name(), request.Name)
			if request.APIBase != "" {
				checked++
				assert.NoError(t, stripe.ValidateFixtureAPIBaseURL(request.APIBase), "%s: %s", entry.Name(), request.Name)
			}
		}
	}
	require.NotZero(t, checked, "expected some bundled fixtures to set api_base")
}

func TestEmbeddedMeterSessionAuthorization(t *testing.T) {
	for _, path := range []string{
		"triggers/v1.billing.meter.no_meter_found.json",
		"triggers/v1.billing.meter.error_report_triggered.json",
	} {
		t.Run(path, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			creds := stripe.NewOAKCredentials("oak_dummy_selected", "dummy-context", false)
			fxt, err := NewFixtureFromFile(fs, creds, "", "", path, nil, nil, nil, nil, false)
			require.NoError(t, err)
			var stream FixtureRequest
			for _, data := range fxt.FixtureData.Requests {
				if data.Name == "create_billing_meter_event_stream" {
					stream = data
				}
			}
			require.NotEmpty(t, stream.Headers["Authorization"])
			fxt.Responses["billing_meter_event_session"] = gjson.Parse(`{"authentication_token":"dummy-session-token"}`)
			req, err := http.NewRequest(http.MethodPost, stream.APIBase+stream.Path, nil)
			require.NoError(t, err)
			creds.SetRequestHeaders(req)
			require.NoError(t, fxt.addCustomHeaders(stream.Headers)(req))
			assert.Equal(t, "Bearer dummy-session-token", req.Header.Get("Authorization"))
			assert.Equal(t, "false", req.Header.Get("Stripe-Livemode"))
			assert.Equal(t, "dummy-context", req.Header.Get("Stripe-Context"))
			assert.Empty(t, req.Header.Get("Stripe-Account"))

			t.Run("copies and edits stay runnable", func(t *testing.T) {
				raw := fxt.GetFixtureFileContent()
				fromRaw, err := NewFixtureFromRawString(fs, creds, "", "", raw)
				require.NoError(t, err)
				localPath := "local/" + path
				require.NoError(t, fs.MkdirAll(filepath.Dir(localPath), 0o700))
				require.NoError(t, afero.WriteFile(fs, localPath, []byte(raw), 0o600))
				fromFile, err := NewFixtureFromFile(fs, creds, "", "", localPath, nil, nil, nil, nil, false)
				require.NoError(t, err)
				mockEdit(t)
				edited, err := NewFixtureFromFile(fs, creds, "", "", path, nil, nil, nil, nil, true)
				require.NoError(t, err)
				for _, copied := range []*Fixture{fromRaw, fromFile, edited} {
					for _, data := range copied.FixtureData.Requests {
						assert.NoError(t, copied.validateRequestHeaders(data), data.Name)
					}
				}
			})

			t.Run("skipped session step is untrusted", func(t *testing.T) {
				skipped := *fxt
				skipped.Skip = []string{"billing_meter_event_session"}
				require.ErrorContains(t, skipped.validateRequestHeaders(stream), "Authorization")
			})

			t.Run("session step after the stream step is untrusted", func(t *testing.T) {
				reordered := *fxt
				reordered.FixtureData.Requests = make([]FixtureRequest, len(fxt.FixtureData.Requests))
				copy(reordered.FixtureData.Requests, fxt.FixtureData.Requests)
				for i, data := range reordered.FixtureData.Requests {
					if data.Name == "billing_meter_event_session" {
						reordered.FixtureData.Requests = append(reordered.FixtureData.Requests[:i], reordered.FixtureData.Requests[i+1:]...)
						reordered.FixtureData.Requests = append(reordered.FixtureData.Requests, data)
						break
					}
				}
				require.ErrorContains(t, reordered.validateRequestHeaders(stream), "Authorization")
			})

			t.Run("session step with another shape is untrusted", func(t *testing.T) {
				for _, mutate := range map[string]func(*FixtureRequest){
					"path":   func(r *FixtureRequest) { r.Path = customersPath },
					"method": func(r *FixtureRequest) { r.Method = "get" },
				} {
					mutated := *fxt
					mutated.FixtureData.Requests = make([]FixtureRequest, len(fxt.FixtureData.Requests))
					copy(mutated.FixtureData.Requests, fxt.FixtureData.Requests)
					for i := range mutated.FixtureData.Requests {
						if mutated.FixtureData.Requests[i].Name == "billing_meter_event_session" {
							mutate(&mutated.FixtureData.Requests[i])
						}
					}
					require.ErrorContains(t, mutated.validateRequestHeaders(stream), "Authorization")
				}
			})

			for _, mutation := range []struct {
				name   string
				change func(*FixtureRequest)
			}{
				{"literal token", func(r *FixtureRequest) { r.Headers["Authorization"] = "Bearer dummy-other" }},
				{"environment token", func(r *FixtureRequest) { r.Headers["Authorization"] = "Bearer ${.env:FIXTURE_TOKEN}" }},
				{"defaulted token", func(r *FixtureRequest) {
					r.Headers["Authorization"] = "Bearer ${billing_meter_event_session:authentication_token|dummy-fallback}"
				}},
				{"token with literal suffix", func(r *FixtureRequest) {
					r.Headers["Authorization"] = "Bearer dummy-${billing_meter_event_session:authentication_token}"
				}},
				{"other response", func(r *FixtureRequest) { r.Headers["Authorization"] = "Bearer ${other:authentication_token}" }},
				{"own response", func(r *FixtureRequest) {
					r.Headers["Authorization"] = "Bearer ${create_billing_meter_event_stream:authentication_token}"
				}},
				{"empty token", func(r *FixtureRequest) { r.Headers["Authorization"] = "" }},
				{"duplicate authorization", func(r *FixtureRequest) { r.Headers["authorization"] = "Bearer dummy-other" }},
				{"mode", func(r *FixtureRequest) { r.Headers["Stripe-Livemode"] = "false" }},
				{"literal context", func(r *FixtureRequest) { r.Headers["Stripe-Context"] = "dummy-context" }},
				{"literal account", func(r *FixtureRequest) { r.Headers["Stripe-Account"] = "acct_dummy" }},
			} {
				t.Run(mutation.name, func(t *testing.T) {
					changed := stream
					changed.Headers = make(map[string]string)
					for k, v := range stream.Headers {
						changed.Headers[k] = v
					}
					mutation.change(&changed)
					require.Error(t, fxt.validateRequestHeaders(changed))
				})
			}

			t.Run("fixture without the session step", func(t *testing.T) {
				other, err := NewFixtureFromFile(fs, creds, "", "", "triggers/payment_intent.succeeded.json", nil, nil, nil, nil, false)
				require.NoError(t, err)
				require.ErrorContains(t, other.validateRequestHeaders(stream), "Authorization")
			})
		})
	}
}

func TestExecuteReturnsRequestNames(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))
		case capturePath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{}, []string{}, []string{}, []string{}, false)
	require.NoError(t, err)

	requestNames, err := fxt.Execute(context.Background(), "")
	require.NoError(t, err)

	require.NotNil(t, fxt.Responses["cust_bender"])
	require.NotNil(t, fxt.Responses["char_bender"])
	require.NotNil(t, fxt.Responses["capt_bender"])

	require.Equal(t, "cust_12345", fxt.Responses["cust_bender"].Get("id").Str)
	require.Equal(t, "char_12345", fxt.Responses["char_bender"].Get("id").Str)
	require.Equal(t, "", fxt.Responses["char_bender"].Get("charge").Str)

	expectedResponseNames := []string{"cust_bender", "char_bender", "capt_bender"}
	assert.Equal(t, expectedResponseNames, requestNames)
}

func TestFixtureAdd(t *testing.T) {
	t.Run("missing value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Add([]string{"price:currency"})
		var missingValue missingRewriteValueError
		assert.True(t, errors.As(err, &missingValue))
		assert.Equal(t, "price:currency", missingValue.value)
	})

	t.Run("missing fixture name", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Add([]string{"currency=usd"})
		var missingFixture missingFixtureNameError
		assert.True(t, errors.As(err, &missingFixture))
		assert.Equal(t, "currency=usd", missingFixture.value)
	})

	t.Run("existing value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Add([]string{"price:amount=1200"})
		assert.NoError(t, err)
		assert.Equal(t, fxt.FixtureData.Requests[0].Params, map[string]interface{}{"amount": 100})
	})

	t.Run("new value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Add([]string{"price:currency=usd"})
		assert.NoError(t, err)
		assert.Equal(t, fxt.FixtureData.Requests[0].Params, map[string]interface{}{"amount": 100, "currency": "usd"})
	})
}

func TestFixtureOverride(t *testing.T) {
	t.Run("missing value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Override([]string{"price:currency"})
		var missingValue missingRewriteValueError
		assert.True(t, errors.As(err, &missingValue))
		assert.Equal(t, "price:currency", missingValue.value)
	})

	t.Run("missing fixture name", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Override([]string{"currency=usd"})
		var missingFixture missingFixtureNameError
		assert.True(t, errors.As(err, &missingFixture))
		assert.Equal(t, "currency=usd", missingFixture.value)
	})

	t.Run("existing value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Override([]string{"price:amount=1200"})
		assert.NoError(t, err)
		assert.Equal(t, fxt.FixtureData.Requests[0].Params, map[string]interface{}{"amount": "1200"})
	})

	t.Run("new value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Override([]string{"price:currency=usd"})
		assert.NoError(t, err)
		assert.Equal(t, fxt.FixtureData.Requests[0].Params, map[string]interface{}{"amount": 100, "currency": "usd"})
	})
}

func TestFixtureRemove(t *testing.T) {
	t.Run("missing fixture name", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Remove([]string{"currency"})
		var missingFixture missingFixtureNameError
		assert.True(t, errors.As(err, &missingFixture))
		assert.Equal(t, "currency", missingFixture.value)
	})

	t.Run("existing value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Remove([]string{"price:amount"})
		assert.NoError(t, err)
		assert.Equal(t, fxt.FixtureData.Requests[0].Params, map[string]interface{}{})
	})

	t.Run("new value", func(t *testing.T) {
		fxt := priceFixture()
		err := fxt.Remove([]string{"price:currency"})
		assert.NoError(t, err)
		assert.Equal(t, fxt.FixtureData.Requests[0].Params, map[string]interface{}{"amount": 100})
	})
}

func priceFixture() *Fixture {
	return &Fixture{
		FixtureData: FixtureData{
			Requests: []FixtureRequest{
				{Name: "price", Params: map[string]interface{}{"amount": 100}},
			},
		},
	}
}

func TestGetFixtureFilenameWithWildcard(t *testing.T) {
	t.Run("account.updated", func(t *testing.T) {
		assert.Equal(t, "account.updated.*.json", getFixtureFilenameWithWildcard("triggers/account.updated.json"))
	})
	t.Run("charge.dispute.created", func(t *testing.T) {
		assert.Equal(t, "charge.dispute.created.*.json", getFixtureFilenameWithWildcard("triggers/charge.dispute.created.json"))
	})
	t.Run("payment_intent.amount_capturable_updated", func(t *testing.T) {
		assert.Equal(t, "payment_intent.amount_capturable_updated.*.json", getFixtureFilenameWithWildcard("triggers/payment_intent.amount_capturable_updated.json"))
	})
}

// Mock edit so that we don't try to open the fixture in an IDE during testing
func mockEdit(t *testing.T) {
	t.Helper()
	original := Edit
	t.Cleanup(func() { Edit = original })
	Edit = func(path string, filedata []byte) ([]byte, error) {
		return filedata, nil
	}
}

func TestSkipSkipFlagIfEditIsTrue(t *testing.T) {
	mockEdit(t)
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case capturePath:
			res.Write([]byte(`{}`))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{"char_bender", "capt_bender"}, []string{}, []string{}, []string{}, true)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)

	require.True(t, fxt.Responses["cust_bender"].Exists())
	require.True(t, fxt.Responses["char_bender"].Exists())
	require.True(t, fxt.Responses["capt_bender"].Exists())
}

func TestSkipOverrideFlagIfEditIsTrue(t *testing.T) {
	mockEdit(t)
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("Failure with request body: %s", err)
		}

		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))

			require.False(t, strings.Contains(string(body), "name=Fry"))
			require.True(t, strings.Contains(string(body), "name=Bender"))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))

			require.False(t, strings.Contains(string(body), "amount=3000"))
			require.True(t, strings.Contains(string(body), "amount=100"))
		case capturePath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{}, []string{"cust_bender:name=Fry", "char_bender:amount=3000"}, []string{}, []string{}, true)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}

func TestSkipAddFlagIfEditIsTrue(t *testing.T) {
	mockEdit(t)
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("Failure with request body: %s", err)
		}

		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
			require.False(t, strings.Contains(string(body), "birthdate=2996-09-04"))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))
			require.False(t, strings.Contains(string(body), "receipt_email=prof.farnsworth%40planex.com"))
		case capturePath:
			res.Write([]byte(`{}`))
			require.False(t, strings.Contains(string(body), "statement_descriptor=Fuel%3A+Beer"))
			require.False(t, strings.Contains(string(body), "nested1[nested2][nested3]=nestedValue"))

		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(
		fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file,
		[]string{}, []string{}, []string{
			"cust_bender:birthdate=2996-09-04",
			"char_bender:receipt_email=prof.farnsworth@planex.com",
			"capt_bender:statement_descriptor=Fuel: Beer",
			"capt_bender:nested1.nested2.nested3=nestedValue",
		},
		[]string{},
		true,
	)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}

func TestSkipRemoveFlagIfEditIsTrue(t *testing.T) {
	mockEdit(t)
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("Failure with request body: %s", err)
		}

		switch url := req.URL.String(); url {
		case customersPath:
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))

			require.True(t, strings.Contains(string(body), "phone"))
		case chargePath:
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))

			require.True(t, strings.Contains(string(body), "capture"))
		case capturePath:
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, file, []byte(testFixture), os.ModePerm)

	fxt, err := NewFixtureFromFile(
		fs, stripe.NewAPIKeyCredentials(apiKey), "", ts.URL, file, []string{}, []string{},
		[]string{}, []string{"cust_bender:phone", "char_bender:capture"},
		true)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}
