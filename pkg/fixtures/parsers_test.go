package fixtures

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestParsePathDoNothing(t *testing.T) {
	fxt := Fixture{}
	http := fixture{
		Path: "/v1/charges",
	}

	path := fxt.parsePath(http)
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

	path := fxt.parsePath(http)
	assert.Equal(t, "/v1/charges/cust_12345", path)
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

	path := fxt.parsePath(http)
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

	path := fxt.parsePath(http)
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

	output := fxt.parseInterface(parsedFixtureData)
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

	output := (fxt.parseInterface(data))
	sort.Strings(output)

	require.Equal(t, len(output), 8)
	require.Equal(t, output[0], "address[city]=New New York")
	require.Equal(t, output[1], "address[line1]=1 Planet Express St")
	require.Equal(t, output[2], "email=bender@planex.com")
	require.Equal(t, output[3], "name=Bender Bending Rodriguez")
	require.Equal(t, output[4], "tax_id_data[0][type]=type_0")
	require.Equal(t, output[5], "tax_id_data[0][value]=value_0")
	require.Equal(t, output[6], "tax_id_data[1][type]=type_1")
	require.Equal(t, output[7], "tax_id_data[1][value]=value_1")
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

	output := (fxt.parseInterface(data))
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

	output := fxt.parseInterface(data)

	require.Equal(t, len(output), 1)
	require.Equal(t, "currency=usd", output[0])
}

func TestParseNoEnv(t *testing.T) {
	fxt := Fixture{}
	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_NOT_SET|+1234567890}"

	output := fxt.parseInterface(data)

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

	output := fxt.parseInterface(data)

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
	output := fxt.parseInterface(data)

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234", output[0])

	fs.Remove(envPath)
}
