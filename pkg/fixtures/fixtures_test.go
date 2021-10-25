package fixtures

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/tidwall/gjson"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMakeRequest(t *testing.T) {
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{}, []string{}, []string{}, []string{})
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
	require.NoError(t, err)

	require.NotNil(t, fxt.responses["cust_bender"])
	require.NotNil(t, fxt.responses["char_bender"])
	require.NotNil(t, fxt.responses["capt_bender"])

	require.Equal(t, "cust_12345", fxt.responses["cust_bender"].Get("id").String())
	require.Equal(t, "char_12345", fxt.responses["char_bender"].Get("id").String())
	require.True(t, fxt.responses["char_bender"].Get("charge").Bool())
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

	fxt, err := NewFixtureFromRawString(fs, apiKey, "", ts.URL, testFixture)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
	require.NoError(t, err)

	require.NotNil(t, fxt.responses["cust_bender"])
	require.NotNil(t, fxt.responses["char_bender"])
	require.NotNil(t, fxt.responses["capt_bender"])

	require.Equal(t, "cust_12345", fxt.responses["cust_bender"].Get("id").String())
	require.Equal(t, "char_12345", fxt.responses["char_bender"].Get("id").String())
	require.True(t, fxt.responses["char_bender"].Get("charge").Bool())
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{"char_bender", "capt_bender"}, []string{}, []string{}, []string{})
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
	require.NoError(t, err)

	require.True(t, fxt.responses["cust_bender"].Exists())
	require.False(t, fxt.responses["char_bender"].Exists())
	require.False(t, fxt.responses["capt_bender"].Exists())
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{}, []string{"cust_bender:name=Fry", "char_bender:amount=3000"}, []string{}, []string{})
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
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
		fs, apiKey, "", ts.URL, file,
		[]string{}, []string{}, []string{
			"cust_bender:birthdate=2996-09-04",
			"char_bender:receipt_email=prof.farnsworth@planex.com",
			"capt_bender:statement_descriptor=Fuel: Beer",
			"capt_bender:nested1.nested2.nested3=nestedValue",
		},
		[]string{},
	)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
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
		fs, apiKey, "", ts.URL, file, []string{}, []string{},
		[]string{}, []string{"cust_bender:phone", "char_bender:capture"},
	)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
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
	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, "failured_test_fixture.json", []string{}, []string{}, []string{}, []string{})
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
	require.NoError(t, err)
	require.NotNil(t, fxt.responses["charge_expected_failure"])
}

func TestMakeRequestUnexpectedFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(500)
		res.Write([]byte(`{"error": "Internal Failure Occurred."}`))
	}))

	defer func() { ts.Close() }()
	afero.WriteFile(fs, "failured_test_fixture.json", []byte(failureTestFixture), os.ModePerm)
	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, "failured_test_fixture.json", []string{}, []string{}, []string{}, []string{})
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background())
	require.NotNil(t, err)
}

func TestUpdateEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	fxt := Fixture{
		Fs: fs,
		responses: map[string]gjson.Result{
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

func TestToFixtureQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected fixtureQuery
		didMatch bool
	}{
		{
			"/v1/charges",
			fixtureQuery{},
			false,
		},
		{
			"/v1/charges/${char_bender:id}/capture",
			fixtureQuery{"${char_bender:id}", "char_bender", "id", ""},
			true,
		},
		{
			"${.env:PHONE_NOT_SET|+1234567890}",
			fixtureQuery{"${.env:PHONE_NOT_SET|+1234567890}", ".env", "PHONE_NOT_SET", "+1234567890"},
			true,
		},
		{
			"/v1/customers/${.env:CUST_ID}",
			fixtureQuery{"${.env:CUST_ID}", ".env", "CUST_ID", ""},
			true,
		},
		{
			"${.env:CUST_ID}",
			fixtureQuery{"${.env:CUST_ID}", ".env", "CUST_ID", ""},
			true,
		},
		{
			"${cust_bender:subscriptions.data.[0].id}",
			fixtureQuery{"${cust_bender:subscriptions.data.[0].id}", "cust_bender", "subscriptions.data.[0].id", ""},
			true,
		},
		{
			"${cust_bender:subscriptions.data.[0].name|Unknown Person}",
			fixtureQuery{"${cust_bender:subscriptions.data.[0].name|Unknown Person}", "cust_bender", "subscriptions.data.[0].name", "Unknown Person"},
			true,
		},
		{
			"${cust_bender:billing_details.address.country}",
			fixtureQuery{"${cust_bender:billing_details.address.country}", "cust_bender", "billing_details.address.country", ""},
			true,
		},
		{
			"${cust_bender:billing_details.address.country|San Mateo}",
			fixtureQuery{"${cust_bender:billing_details.address.country|San Mateo}", "cust_bender", "billing_details.address.country", "San Mateo"},
			true,
		},
	}

	for _, test := range tests {
		actualQuery, actualDidMatch := toFixtureQuery(test.input)
		assert.Equal(t, test.expected, actualQuery)
		assert.Equal(t, test.didMatch, actualDidMatch)
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{}, []string{}, []string{}, []string{})
	require.NoError(t, err)

	requestNames, err := fxt.Execute(context.Background())
	require.NoError(t, err)

	require.NotNil(t, fxt.responses["cust_bender"])
	require.NotNil(t, fxt.responses["char_bender"])
	require.NotNil(t, fxt.responses["capt_bender"])

	require.Equal(t, "cust_12345", fxt.responses["cust_bender"].Get("id").Str)
	require.Equal(t, "char_12345", fxt.responses["char_bender"].Get("id").Str)
	require.Equal(t, "", fxt.responses["char_bender"].Get("charge").Str)

	expectedResponseNames := []string{"cust_bender", "char_bender", "capt_bender"}
	assert.Equal(t, expectedResponseNames, requestNames)
}
