package fixtures

import (
	"context"
	"errors"
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{}, []string{}, []string{}, []string{}, false)
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

	fxt, err := NewFixtureFromRawString(fs, apiKey, "", ts.URL, testFixture)
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{"char_bender", "capt_bender"}, []string{}, []string{}, []string{}, false)
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{}, []string{"cust_bender:name=Fry", "char_bender:amount=3000"}, []string{}, []string{}, false)
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
		fs, apiKey, "", ts.URL, file,
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
		fs, apiKey, "", ts.URL, file, []string{}, []string{},
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
	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, "failured_test_fixture.json", []string{}, []string{}, []string{}, []string{}, false)
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
	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, "failured_test_fixture.json", []string{}, []string{}, []string{}, []string{}, false)
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{}, []string{}, []string{}, []string{}, false)
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
		assert.Equal(t, getFixtureFilenameWithWildcard("triggers/account.updated.json"), "account.updated.*.json")
	})
	t.Run("charge.dispute.created", func(t *testing.T) {
		assert.Equal(t, getFixtureFilenameWithWildcard("triggers/charge.dispute.created.json"), "charge.dispute.created.*.json")
	})
	t.Run("payment_intent.amount_capturable_updated", func(t *testing.T) {
		assert.Equal(t, getFixtureFilenameWithWildcard("triggers/payment_intent.amount_capturable_updated.json"), "payment_intent.amount_capturable_updated.*.json")
	})
}

// Mock edit so that we don't try to open the fixture in an IDE during testing
func mockEdit() {
	Edit = func(path string, filedata []byte) ([]byte, error) {
		return filedata, nil
	}
}

func TestSkipSkipFlagIfEditIsTrue(t *testing.T) {
	mockEdit()
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{"char_bender", "capt_bender"}, []string{}, []string{}, []string{}, true)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)

	require.True(t, fxt.Responses["cust_bender"].Exists())
	require.True(t, fxt.Responses["char_bender"].Exists())
	require.True(t, fxt.Responses["capt_bender"].Exists())
}

func TestSkipOverrideFlagIfEditIsTrue(t *testing.T) {
	mockEdit()
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

	fxt, err := NewFixtureFromFile(fs, apiKey, "", ts.URL, file, []string{}, []string{"cust_bender:name=Fry", "char_bender:amount=3000"}, []string{}, []string{}, true)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}

func TestSkipAddFlagIfEditIsTrue(t *testing.T) {
	mockEdit()
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
		fs, apiKey, "", ts.URL, file,
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
	mockEdit()
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
		fs, apiKey, "", ts.URL, file, []string{}, []string{},
		[]string{}, []string{"cust_bender:phone", "char_bender:capture"},
		true)
	require.NoError(t, err)

	_, err = fxt.Execute(context.Background(), "")
	require.NoError(t, err)
}
