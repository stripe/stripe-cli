package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

func TestParsePathDoNothing(t *testing.T) {
	fxt := Fixture{}
	http := fixture{
		Path: "/v1/charges",
	}

	path, _ := fxt.parsePath(http)
	assert.Equal(t, http.Path, path)
}

func TestParsePathOneParam(t *testing.T) {
	fxt := Fixture{
		responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${char_bender:id}",
	}

	path, _ := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/cust_12345", path)
}

func TestParsePathReferenceErrorWithSuggestion(t *testing.T) {
	fxt := Fixture{
		responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${char:id}",
	}

	_, err := fxt.parsePath(http)

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s\nPerhaps you meant one of the following: char_bender",
		color.Red("✘ Validation error").String(),
		ansi.Bold("char"),
	)

	assert.Equal(t, expected, err)
}

func TestParsePathReferenceErrorNoSuggestion(t *testing.T) {
	fxt := Fixture{
		responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${foo:id}",
	}

	_, err := fxt.parsePath(http)

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s",
		color.Red("✘ Validation error").String(),
		ansi.Bold("foo"),
	)

	assert.Equal(t, expected, err)
}

func TestParseQueryReferenceErrorWithSuggestion(t *testing.T) {
	fxt := Fixture{
		responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
		},
	}

	_, err := fxt.parseQuery("${bender:id}")

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s\nPerhaps you meant one of the following: char_bender",
		color.Red("✘ Validation error").String(),
		ansi.Bold("bender"),
	)

	assert.Equal(t, expected, err)
}

func TestParseQueryReferenceErrorNoSuggestion(t *testing.T) {
	fxt := Fixture{
		responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
		},
	}

	_, err := fxt.parseQuery("${foo:id}")

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s",
		color.Red("✘ Validation error").String(),
		ansi.Bold("foo"),
	)

	assert.Equal(t, expected, err)
}

func TestParseTwoParam(t *testing.T) {
	fxt := Fixture{
		responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "char_12345"}`),
			"cust_bender": gjson.Parse(`{"id": "cust_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${char_bender:id}/capture/${cust_bender:id}",
	}

	path, _ := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/char_12345/capture/cust_12345", path)
}

func TestParsePathOneParamWithTrailing(t *testing.T) {
	fxt := Fixture{
		responses: map[string]gjson.Result{
			"char_bender": gjson.Parse(`{"id": "char_12345"}`),
		},
	}
	http := fixture{
		Path: "/v1/charges/${char_bender:id}/capture",
	}

	path, _ := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/char_12345/capture", path)
}

func TestParseInterfaceFromRaw(t *testing.T) {
	var rawFixtureData = []byte(`{
		"salary": 1000000000,
		"email": "person@example.com"
	}`)

	parsedFixtureData := make(map[string]interface{})
	json.Unmarshal(rawFixtureData, &parsedFixtureData)

	fxt := Fixture{}

	output, _ := fxt.parseInterface(parsedFixtureData)
	sort.Strings(output)

	require.Equal(t, len(output), 2)
	require.Equal(t, output[0], "email=person@example.com")
	require.Equal(t, output[1], "salary=1000000000")
}

func TestParseInterface(t *testing.T) {
	address := make(map[string]interface{})
	address["line1"] = "1 Planet Express St"
	address["city"] = "New New York"

	// array of hashes
	taxIDData := make([]interface{}, 2)
	taxIDZero := make(map[string]interface{})
	taxIDZero["type"] = "type_0"
	taxIDZero["value"] = "value_0"
	taxIDOne := make(map[string]interface{})
	taxIDOne["type"] = "type_1"
	taxIDOne["value"] = "value_1"
	taxIDData[0] = taxIDZero
	taxIDData[1] = taxIDOne

	data := make(map[string]interface{})
	data["name"] = "Bender Bending Rodriguez"
	data["email"] = "bender@planex.com"
	data["address"] = address
	data["tax_id_data"] = taxIDData
	fxt := Fixture{}

	output, _ := fxt.parseInterface(data)
	sort.Strings(output)

	require.Equal(t, len(output), 8)
	require.Equal(t, "address[city]=New New York", output[0])
	require.Equal(t, "address[line1]=1 Planet Express St", output[1])
	require.Equal(t, "email=bender@planex.com", output[2])
	require.Equal(t, "name=Bender Bending Rodriguez", output[3])
	require.Equal(t, "tax_id_data[0][type]=type_0", output[4])
	require.Equal(t, "tax_id_data[0][value]=value_0", output[5])
	require.Equal(t, "tax_id_data[1][type]=type_1", output[6])
	require.Equal(t, "tax_id_data[1][value]=value_1", output[7])
}

func TestParseWithQueryIgnoreDefault(t *testing.T) {
	jsonData := gjson.Parse(`{"id": "cust_bend123456789", "currency": "eur"}`)

	fxt := Fixture{}
	fxt.responses = make(map[string]gjson.Result)
	fxt.responses["cust_bender"] = jsonData

	data := make(map[string]interface{})
	data["customer"] = "${cust_bender:id}"
	data["source"] = "tok_visa"
	data["amount"] = "100"
	data["currency"] = "${cust_bender:currency|usd}"

	output, _ := fxt.parseInterface(data)
	sort.Strings(output)

	require.Equal(t, len(output), 4)
	require.Equal(t, "amount=100", output[0])
	require.Equal(t, "currency=eur", output[1])
	require.Equal(t, "customer=cust_bend123456789", output[2])
	require.Equal(t, "source=tok_visa", output[3])
}

func TestParseWithQueryDefaultValue(t *testing.T) {
	jsonData := gjson.Parse(`{"id": "cust_bend123456789"}`)

	fxt := Fixture{}
	fxt.responses = make(map[string]gjson.Result)
	fxt.responses["cust_bender"] = jsonData

	data := make(map[string]interface{})
	data["currency"] = "${cust_bender:currency|usd}"

	output, _ := fxt.parseInterface(data)

	require.Equal(t, len(output), 1)
	require.Equal(t, "currency=usd", output[0])
}

func TestParseNoEnv(t *testing.T) {
	fxt := Fixture{}
	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_NOT_SET|+1234567890}"

	output, _ := fxt.parseInterface(data)

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

	path, _ := fxt.parsePath(http)
	assert.Equal(t, "/v1/customers/cust_12345", path)

	output, _ := fxt.parseInterface(data)

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
	output, _ := fxt.parseInterface(data)

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234", output[0])

	fs.Remove(envPath)
}

func TestParseWithEnvSubstring(t *testing.T) {
	fs := afero.NewOsFs()
	wd, _ := os.Getwd()
	envPath := path.Join(wd, ".env")
	afero.WriteFile(fs, envPath, []byte(`BASE_API_URL="https://myexample.com"`), os.ModePerm)

	fxt := Fixture{}
	data := make(map[string]interface{})
	data["url"] = "${.env:BASE_API_URL}/hook/stripe"
	output, _ := (fxt.parseInterface(data))

	require.Equal(t, len(output), 1)
	require.Equal(t, "url=https://myexample.com/hook/stripe", output[0])

	fs.Remove(envPath)
}
