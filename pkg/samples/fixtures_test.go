package samples

import (
	"net/http"
	"net/http/httptest"
	"os"
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
		"template_version": "0"
	},
	"fixtures": [
		{
			"name": "cust_bender",
			"http": {
				"path": "/v1/customers",
				"method": "post"
			},
			"data": {
				"name": "Bender Bending Rodriguez",
				"email": "bender@planex.com",
				"address": {
					"line1": "1 Planet Express St",
					"city": "New New York"
				}
			}
		},
		{
			"name": "char_bender",
			"http": {
				"path": "/v1/charges",
				"method": "post"
			},
			"data": {
				"customer": "#$cust_bender:id",
				"source": "tok_visa",
				"amount": "100",
				"currency": "usd",
				"capture": false
			}
		},
		{
			"name": "capt_bender",
			"http": {
				"path": "/v1/charges/:charge/capture",
				"method": "post",
				"params": {
					":charge": "#$char_bender:id"
				}
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

func TestParseWithQuery(t *testing.T) {
	jsonData := gojsonq.New().JSONString(`{"id": "cust_bend123456789"}`)

	fxt := Fixture{}
	fxt.responses = make(map[string]*gojsonq.JSONQ)
	fxt.responses["cust_bender"] = jsonData

	data := make(map[string]interface{})
	data["customer"] = "#$cust_bender:id"
	data["source"] = "tok_visa"
	data["amount"] = "100"
	data["currency"] = "usd"

	output := (fxt.parseInterface(data))
	sort.Strings(output)

	require.Equal(t, len(output), 4)
	require.Equal(t, output[0], "amount=100")
	require.Equal(t, output[1], "currency=usd")
	require.Equal(t, output[2], "customer=cust_bend123456789")
	require.Equal(t, output[3], "source=tok_visa")
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

	fxt := Fixture{
		Fs:      fs,
		BaseURL: ts.URL,
		APIKey:  "sk_test_1234",
	}

	err := fxt.NewFixture("test_fixture.json")
	require.Nil(t, err)

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

func TestParsePathDoNothing(t *testing.T) {
	fxt := Fixture{}
	http := fixtureHTTP{
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
	http := fixtureHTTP{
		Path: "/v1/charges/:charge",
		Params: map[string]string{
			":charge": "#$char_bender:id",
		},
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
	http := fixtureHTTP{
		Path: "/v1/charges/:charge/capture",
		Params: map[string]string{
			":charge": "#$char_bender:id",
		},
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
	http := fixtureHTTP{
		Path: "/v1/charges/:charge/capture/:cust",
		Params: map[string]string{
			":charge": "#$char_bender:id",
			":cust":   "#$cust_bender:id",
		},
	}

	path := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/char_12345/capture/cust_12345", path)
}
