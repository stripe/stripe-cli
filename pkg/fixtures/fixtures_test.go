package fixtures

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/spf13/afero"
	"github.com/thedevsaddam/gojsonq"

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

func TestParseInterface(t *testing.T) {
	address := make(map[string]interface{})
	address["line1"] = "1 Planet Express St"
	address["city"] = "New New York"

	data := make(map[string]interface{})
	data["name"] = "Bender Bending Rodriguez"
	data["email"] = "bender@planex.com"
	data["address"] = address

	fxt := Fixture{}

	output := (fxt.parseInterface(data))
	sort.Strings(output)

	require.Equal(t, len(output), 4)
	require.Equal(t, output[0], "address[city]=New New York")
	require.Equal(t, output[1], "address[line1]=1 Planet Express St")
	require.Equal(t, output[2], "email=bender@planex.com")
	require.Equal(t, output[3], "name=Bender Bending Rodriguez")
}

func TestParseWithQueryIgnoreDefault(t *testing.T) {
	jsonData := gojsonq.New().JSONString(`{"id": "cust_bend123456789", "currency": "eur"}`)

	fxt := Fixture{}
	fxt.responses = make(map[string]*gojsonq.JSONQ)
	fxt.responses["cust_bender"] = jsonData

	data := make(map[string]interface{})
	data["customer"] = "${cust_bender:id}"
	data["source"] = "tok_visa"
	data["amount"] = "100"
	data["currency"] = "${cust_bender:currency|usd}"

	output := (fxt.parseInterface(data))
	sort.Strings(output)

	require.Equal(t, len(output), 4)
	require.Equal(t, "amount=100", output[0])
	require.Equal(t, "currency=eur", output[1])
	require.Equal(t, "customer=cust_bend123456789", output[2])
	require.Equal(t, "source=tok_visa", output[3])
}

func TestParseWithQueryDefaultValue(t *testing.T) {
	jsonData := gojsonq.New().JSONString(`{"id": "cust_bend123456789"}`)

	fxt := Fixture{}
	fxt.responses = make(map[string]*gojsonq.JSONQ)
	fxt.responses["cust_bender"] = jsonData

	data := make(map[string]interface{})
	data["currency"] = "${cust_bender:currency|usd}"

	output := (fxt.parseInterface(data))

	require.Equal(t, len(output), 1)
	require.Equal(t, "currency=usd", output[0])
}

func TestParseNoEnv(t *testing.T) {
	fxt := Fixture{}
	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_NOT_SET|+1234567890}"

	output := (fxt.parseInterface(data))

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234567890", output[0])
}

func TestParseWithLocalEnv(t *testing.T) {
	fxt := Fixture{}
	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_LOCAL|+1234567890}"

	os.Setenv("CUST_ID", "cust_12345")
	os.Setenv("PHONE_LOCAL", "+1234")

	http := fixture{
		Path: "/v1/customers/${.env:CUST_ID}",
	}

	path := fxt.parsePath(http)
	assert.Equal(t, "/v1/customers/cust_12345", path)

	output := (fxt.parseInterface(data))

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234", output[0])

	os.Unsetenv("PHONE_LOCAL")
}

func TestParseWithEnvFile(t *testing.T) {
	fs := afero.NewOsFs()
	wd, _ := os.Getwd()
	envPath := path.Join(wd, ".env")
	afero.WriteFile(fs, envPath, []byte(`PHONE_FILE="+1234"`), os.ModePerm)

	fxt := Fixture{}
	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_FILE|+1234567890}"
	output := (fxt.parseInterface(data))

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234", output[0])

	fs.Remove(envPath)
}

func TestMakeRequest(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case "/v1/customers":
			res.Write([]byte(`{"id": "cust_12345", "foo": "bar"}`))
		case "/v1/charges":
			res.Write([]byte(`{"charge": true, "id": "char_12345"}`))
		case "/v1/charges/char_12345/capture":
			// Do nothing, we just want to verify this request came in
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	defer func() { ts.Close() }()

	afero.WriteFile(fs, "test_fixture.json", []byte(testFixture), os.ModePerm)

	fxt, err := NewFixture(fs, "sk_test_1234", "", ts.URL, "test_fixture.json")
	require.NoError(t, err)

	err = fxt.Execute()
	require.NoError(t, err)

	require.NotNil(t, fxt.responses["cust_bender"])
	require.NotNil(t, fxt.responses["char_bender"])
	require.NotNil(t, fxt.responses["capt_bender"])

	// After you make a `Find` request you need `Reset` the gojsonq object
	fxt.responses["cust_bender"].Reset()
	require.Equal(t, "cust_12345", fxt.responses["cust_bender"].Find("id"))

	fxt.responses["char_bender"].Reset()
	require.Equal(t, "char_12345", fxt.responses["char_bender"].Find("id"))

	fxt.responses["char_bender"].Reset()
	require.True(t, fxt.responses["char_bender"].Find("charge").(bool))
}

func TestMakeRequestExpectedFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(402)
		res.Write([]byte(`{"error": {"type": "card_error"}}`))
	}))

	defer func() { ts.Close() }()
	afero.WriteFile(fs, "failured_test_fixture.json", []byte(failureTestFixture), os.ModePerm)
	fxt, err := NewFixture(fs, "sk_test_1234", "", ts.URL, "failured_test_fixture.json")
	require.NoError(t, err)

	err = fxt.Execute()
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
	fxt, err := NewFixture(fs, "sk_test_1234", "", ts.URL, "failured_test_fixture.json")
	require.NoError(t, err)

	err = fxt.Execute()
	require.NotNil(t, err)
}

func TestParsePathDoNothing(t *testing.T) {
	fxt := Fixture{}
	http := fixture{
		Path: "/v1/charges",
	}

	path := fxt.parsePath(http)
	assert.Equal(t, http.Path, path)
}

func TestParseOneParam(t *testing.T) {
	fxt := Fixture{
		responses: map[string]*gojsonq.JSONQ{
			"char_bender": gojsonq.New().FromString(`{"id": "cust_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${char_bender:id}",
	}

	path := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/cust_12345", path)
}

func TestParseOneParamWithTrailing(t *testing.T) {
	fxt := Fixture{
		responses: map[string]*gojsonq.JSONQ{
			"char_bender": gojsonq.New().FromString(`{"id": "char_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${char_bender:id}/capture",
	}

	path := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/char_12345/capture", path)
}

func TestParseTwoParam(t *testing.T) {
	fxt := Fixture{
		responses: map[string]*gojsonq.JSONQ{
			"char_bender": gojsonq.New().FromString(`{"id": "char_12345"}`),
			"cust_bender": gojsonq.New().FromString(`{"id": "cust_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${char_bender:id}/capture/${cust_bender:id}",
	}

	path := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/char_12345/capture/cust_12345", path)
}

func TestUpdateEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	fxt := Fixture{
		Fs: fs,
		responses: map[string]*gojsonq.JSONQ{
			"char_bender": gojsonq.New().FromString(`{"id": "char_12345"}`),
			"cust_bender": gojsonq.New().FromString(`{"id": "cust_12345"}`),
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
			fixtureQuery{"char_bender", "id", ""},
			true,
		},
		{
			"${.env:PHONE_NOT_SET|+1234567890}",
			fixtureQuery{".env", "PHONE_NOT_SET", "+1234567890"},
			true,
		},
		{
			"/v1/customers/${.env:CUST_ID}",
			fixtureQuery{".env", "CUST_ID", ""},
			true,
		},
		{
			"${.env:CUST_ID}",
			fixtureQuery{".env", "CUST_ID", ""},
			true,
		},
		{
			"${cust_bender:subscriptions.data.[0].id}",
			fixtureQuery{"cust_bender", "subscriptions.data.[0].id", ""},
			true,
		},
		{
			"${cust_bender:subscriptions.data.[0].name|Unknown Person}",
			fixtureQuery{"cust_bender", "subscriptions.data.[0].name", "Unknown Person"},
			true,
		},
		{
			"${cust_bender:billing_details.address.country}",
			fixtureQuery{"cust_bender", "billing_details.address.country", ""},
			true,
		},
		{
			"${cust_bender:billing_details.address.country|San Mateo}",
			fixtureQuery{"cust_bender", "billing_details.address.country", "San Mateo"},
			true,
		},
	}

	for _, test := range tests {
		actualQuery, actualDidMatch := toFixtureQuery(test.input)
		assert.Equal(t, test.expected, actualQuery)
		assert.Equal(t, test.didMatch, actualDidMatch)
	}
}
