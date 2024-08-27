package parsers

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

func TestParsePathDoNothing(t *testing.T) {
	httpPath := "/v1/charges"

	path, _ := ParsePath(httpPath, make(map[string]gjson.Result, 0))
	assert.Equal(t, httpPath, path)
}

func TestParsePathOneParam(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
	}
	httpPath := "/v1/charges/${char_bender:id}"

	path, _ := ParsePath(httpPath, queryRespMap)
	assert.Equal(t, "/v1/charges/cust_12345", path)
}

func TestParsePathReferenceErrorWithSuggestion(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
	}
	httpPath := "/v1/charges/${char:id}"

	_, err := ParsePath(httpPath, queryRespMap)

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s\nPerhaps you meant one of the following: char_bender",
		color.Red("✘ Validation error").String(),
		ansi.Bold("char"),
	)

	assert.Equal(t, expected, err)
}

func TestParsePathReferenceErrorNoSuggestion(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
	}
	httpPath := "/v1/charges/${foo:id}"

	_, err := ParsePath(httpPath, queryRespMap)

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s",
		color.Red("✘ Validation error").String(),
		ansi.Bold("foo"),
	)

	assert.Equal(t, expected, err)
}

func TestParseQueryReferenceErrorWithSuggestion(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
	}
	httpPath := "${bender:id}"

	_, err := ParsePath(httpPath, queryRespMap)

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s\nPerhaps you meant one of the following: char_bender",
		color.Red("✘ Validation error").String(),
		ansi.Bold("bender"),
	)

	assert.Equal(t, expected, err)
}

func TestParseQueryReferenceErrorNoSuggestion(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"char_bender": gjson.Parse(`{"id": "cust_12345"}`),
	}
	httpPath := "${foo:id}"

	_, err := ParsePath(httpPath, queryRespMap)

	color := ansi.Color(os.Stdout)
	expected := fmt.Errorf(
		"%s - an undeclared fixture name was referenced: %s",
		color.Red("✘ Validation error").String(),
		ansi.Bold("foo"),
	)

	assert.Equal(t, expected, err)
}

func TestParseTwoParam(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"char_bender": gjson.Parse(`{"id": "char_12345"}`),
		"cust_bender": gjson.Parse(`{"id": "cust_12345"}`),
	}

	httpPath := "/v1/charges/${char_bender:id}/capture/${cust_bender:id}"

	path, _ := ParsePath(httpPath, queryRespMap)
	assert.Equal(t, "/v1/charges/char_12345/capture/cust_12345", path)
}

func TestParsePathOneParamWithTrailing(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"char_bender": gjson.Parse(`{"id": "char_12345"}`),
	}

	httpPath := "/v1/charges/${char_bender:id}/capture"

	path, _ := ParsePath(httpPath, queryRespMap)
	assert.Equal(t, "/v1/charges/char_12345/capture", path)
}

func TestParseInterfaceFromRaw(t *testing.T) {
	var rawFixtureData = []byte(`{
		"salary": 1000000000,
		"email": "person@example.com"
	}`)

	parsedFixtureData := make(map[string]interface{})
	json.Unmarshal(rawFixtureData, &parsedFixtureData)

	output, _ := ParseToFormData(parsedFixtureData, make(map[string]gjson.Result))
	sort.Strings(output)

	require.Equal(t, len(output), 2)
	require.Equal(t, output[0], "email=person@example.com")
	require.Equal(t, output[1], "salary=1000000000")
}

func TestParseInterfaceToJSONFromRaw(t *testing.T) {
	var rawFixtureData = []byte(`{
		"salary": 1000000000,
		"email": "person@example.com"
	}`)

	parsedFixtureData := make(map[string]interface{})
	json.Unmarshal(rawFixtureData, &parsedFixtureData)

	output, _ := ParseToApplicationJSON(parsedFixtureData, make(map[string]gjson.Result))
	jsonParams, _ := json.Marshal(output)
	json := string(jsonParams)
	require.Equal(t, 1000000000, int(gjson.Get(json, "salary").Num))
	require.Equal(t, "person@example.com", gjson.Get(json, "email").String())
}

func TestParseInterfaceDeeplyNested(t *testing.T) {
	label := make(map[string]interface{})
	label["custom"] = "First Name"
	label["type"] = "custom"

	customField := make(map[string]interface{})
	customField["label"] = label

	customFields := make([]interface{}, 1)
	customFields[0] = customField

	data := make(map[string]interface{})
	data["custom_fields"] = customFields

	output, _ := ParseToFormData(data, make(map[string]gjson.Result))
	sort.Strings(output)

	require.Equal(t, 2, len(output))
	require.Equal(t, "custom_fields[0][label][custom]=First Name", output[0])
	require.Equal(t, "custom_fields[0][label][type]=custom", output[1])
}

func TestParseInterfaceToJSONDeeplyNested(t *testing.T) {
	label := make(map[string]interface{})
	label["custom"] = "First Name"
	label["type"] = "custom"

	customField := make(map[string]interface{})
	customField["label"] = label

	customFields := make([]interface{}, 1)
	customFields[0] = customField

	data := make(map[string]interface{})
	data["custom_fields"] = customFields

	output, _ := ParseToApplicationJSON(data, make(map[string]gjson.Result))
	jsonParams, _ := json.Marshal(output)
	json := string(jsonParams)
	require.Equal(t, "custom", gjson.Get(json, "custom_fields.0.label.type").String())
	require.Equal(t, "First Name", gjson.Get(json, "custom_fields.0.label.custom").String())
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
	data["email"] = "bender@test.com"
	data["address"] = address
	data["tax_id_data"] = taxIDData

	output, _ := ParseToApplicationJSON(data, make(map[string]gjson.Result))
	jsonParams, _ := json.Marshal(output)
	json := string(jsonParams)
	require.Equal(t, "New New York", gjson.Get(json, "address.city").String())
	require.Equal(t, "1 Planet Express St", gjson.Get(json, "address.line1").String())
	require.Equal(t, "bender@test.com", gjson.Get(json, "email").String())
	require.Equal(t, "Bender Bending Rodriguez", gjson.Get(json, "name").String())
	require.Equal(t, "type_0", gjson.Get(json, "tax_id_data.0.type").String())
	require.Equal(t, "value_0", gjson.Get(json, "tax_id_data.0.value").String())
	require.Equal(t, "type_1", gjson.Get(json, "tax_id_data.1.type").String())
	require.Equal(t, "value_1", gjson.Get(json, "tax_id_data.1.value").String())
}

func TestParseInterfaceToJSON(t *testing.T) {
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
	data["email"] = "bender@test.com"
	data["address"] = address
	data["tax_id_data"] = taxIDData

	output, _ := ParseToFormData(data, make(map[string]gjson.Result))
	sort.Strings(output)

	require.Equal(t, len(output), 8)
	require.Equal(t, "address[city]=New New York", output[0])
	require.Equal(t, "address[line1]=1 Planet Express St", output[1])
	require.Equal(t, "email=bender@test.com", output[2])
	require.Equal(t, "name=Bender Bending Rodriguez", output[3])
	require.Equal(t, "tax_id_data[0][type]=type_0", output[4])
	require.Equal(t, "tax_id_data[0][value]=value_0", output[5])
	require.Equal(t, "tax_id_data[1][type]=type_1", output[6])
	require.Equal(t, "tax_id_data[1][value]=value_1", output[7])
}

func TestParseWithQueryIgnoreDefault(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"cust_bender": gjson.Parse(`{"id": "cust_bend123456789", "currency": "eur"}`),
	}

	data := make(map[string]interface{})
	data["customer"] = "${cust_bender:id}"
	data["source"] = "tok_visa"
	data["amount"] = "100"
	data["currency"] = "${cust_bender:currency|usd}"

	output, _ := ParseToFormData(data, queryRespMap)
	sort.Strings(output)

	require.Equal(t, len(output), 4)
	require.Equal(t, "amount=100", output[0])
	require.Equal(t, "currency=eur", output[1])
	require.Equal(t, "customer=cust_bend123456789", output[2])
	require.Equal(t, "source=tok_visa", output[3])
}

func TestParseJSONWithQueryIgnoreDefault(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"cust_bender": gjson.Parse(`{"id": "cust_bend123456789", "currency": "eur"}`),
	}

	data := make(map[string]interface{})
	data["customer"] = "${cust_bender:id}"
	data["source"] = "tok_visa"
	data["amount"] = "100"
	data["currency"] = "${cust_bender:currency|usd}"

	output, _ := ParseToApplicationJSON(data, queryRespMap)
	jsonParams, _ := json.Marshal(output)
	json := string(jsonParams)

	require.Equal(t, "100", gjson.Get(json, "amount").String())
	require.Equal(t, "eur", gjson.Get(json, "currency").String())
	require.Equal(t, "cust_bend123456789", gjson.Get(json, "customer").String())
	require.Equal(t, "tok_visa", gjson.Get(json, "source").String())
}

func TestParseWithQueryDefaultValue(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"cust_bender": gjson.Parse(`{"id": "cust_bend123456789"}`),
	}

	data := make(map[string]interface{})
	data["currency"] = "${cust_bender:currency|usd}"

	output, _ := ParseToFormData(data, queryRespMap)

	require.Equal(t, len(output), 1)
	require.Equal(t, "currency=usd", output[0])
}

func TestParseNoEnv(t *testing.T) {
	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_NOT_SET|+1234567890}"

	output, _ := ParseToFormData(data, make(map[string]gjson.Result))

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234567890", output[0])
}

func TestParseWithLocalEnv(t *testing.T) {
	queryRespMap := map[string]gjson.Result{}

	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_LOCAL|+1234567890}"

	os.Setenv("CUST_ID", "cust_12345")
	os.Setenv("PHONE_LOCAL", "+1234")

	httpPath := "/v1/customers/${.env:CUST_ID}"

	path, _ := ParsePath(httpPath, queryRespMap)
	assert.Equal(t, "/v1/customers/cust_12345", path)

	output, _ := ParseToFormData(data, queryRespMap)

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234", output[0])

	os.Unsetenv("PHONE_LOCAL")
}

func TestParseWithEnvFile(t *testing.T) {
	fs := afero.NewOsFs()
	wd, _ := os.Getwd()
	envPath := path.Join(wd, ".env")
	afero.WriteFile(fs, envPath, []byte(`PHONE_FILE="+1234"`), os.ModePerm)

	data := make(map[string]interface{})
	data["phone"] = "${.env:PHONE_FILE|+1234567890}"
	output, _ := ParseToFormData(data, make(map[string]gjson.Result))

	require.Equal(t, len(output), 1)
	require.Equal(t, "phone=+1234", output[0])

	fs.Remove(envPath)
}

func TestParseWithEnvSubstring(t *testing.T) {
	fs := afero.NewOsFs()
	wd, _ := os.Getwd()
	envPath := path.Join(wd, ".env")
	afero.WriteFile(fs, envPath, []byte(`BASE_API_URL="https://myexample.com"`), os.ModePerm)

	data := make(map[string]interface{})
	data["url"] = "${.env:BASE_API_URL}/hook/stripe"
	output, _ := ParseToFormData(data, make(map[string]gjson.Result))

	require.Equal(t, len(output), 1)
	require.Equal(t, "url=https://myexample.com/hook/stripe", output[0])

	fs.Remove(envPath)
}

func TestParseWithTimeNow(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"cust_bender": gjson.Parse(`{"id": "cust_bend123456789"}`),
	}

	data := make(map[string]interface{})
	data["time"] = "${time-now-RFC3339}"

	output, _ := ParseToFormData(data, queryRespMap)
	expectedOutput := time.Now().Format(time.RFC3339)
	require.Equal(t, len(output), 1)
	// Check for equality except for seconds
	require.True(t, strings.HasPrefix(output[0], "time="+expectedOutput[:len(expectedOutput)-3]))
}

func TestParseWithUUID(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"cust_bender": gjson.Parse(`{"id": "cust_bend123456789"}`),
	}

	data := make(map[string]interface{})
	data["identifier"] = "${generate-uuid}"

	output, _ := ParseToFormData(data, queryRespMap)
	require.Equal(t, len(output), 1)
	require.True(t, strings.HasPrefix(output[0], "identifier="))
	generatedUUID := strings.Split(output[0], "=")
	res, err := uuid.Parse(generatedUUID[1])
	require.Nil(t, err)
	require.NotNil(t, res)
}

func TestParseArray(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"cust_bender": gjson.Parse(`{"id": "cust_bend123456789", "timezones": ["Europe/Brussels", "Europe/Berlin"]}`),
	}

	data := make(map[string]interface{})
	data["customer"] = "${cust_bender:id}"
	data["timezones"] = "${cust_bender:timezones}"
	data["first_timezone"] = "${cust_bender:timezones.0}"
	data["second_timezone"] = "${cust_bender:timezones.1}"
	data["third_timezone"] = "${cust_bender:timezones.2|notimezonefound}"

	output, _ := ParseToFormData(data, queryRespMap)
	sort.Strings(output)

	require.Equal(t, len(output), 5)
	require.Equal(t, "customer=cust_bend123456789", output[0])
	require.Equal(t, "first_timezone=Europe/Brussels", output[1])
	require.Equal(t, "second_timezone=Europe/Berlin", output[2])
	require.Equal(t, "third_timezone=notimezonefound", output[3])
	require.Equal(t, "timezones=[\"Europe/Brussels\", \"Europe/Berlin\"]", output[4])
}

func TestParseArrayToJSON(t *testing.T) {
	queryRespMap := map[string]gjson.Result{
		"cust_bender": gjson.Parse(`{"id": "cust_bend123456789", "timezones": ["Europe/Brussels","Europe/Berlin"]}`),
	}

	data := make(map[string]interface{})
	data["customer"] = "${cust_bender:id}"
	data["timezones"] = "${cust_bender:timezones}"
	data["first_timezone"] = "${cust_bender:timezones.0}"
	data["second_timezone"] = "${cust_bender:timezones.1}"
	data["third_timezone"] = "${cust_bender:timezones.2|notimezonefound}"

	output, _ := ParseToApplicationJSON(data, queryRespMap)
	jsonParams, _ := json.Marshal(output)
	json := string(jsonParams)

	require.Equal(t, "cust_bend123456789", gjson.Get(json, "customer").String())
	require.Equal(t, "Europe/Brussels", gjson.Get(json, "first_timezone").String())
	require.Equal(t, "Europe/Berlin", gjson.Get(json, "second_timezone").String())
	require.Equal(t, "notimezonefound", gjson.Get(json, "third_timezone").String())
	require.Equal(t, "[\"Europe/Brussels\",\"Europe/Berlin\"]", gjson.Get(json, "timezones").String())
}

func TestToFixtureQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected FixtureQuery
		didMatch bool
	}{
		{
			"/v1/charges",
			FixtureQuery{},
			false,
		},
		{
			"/v1/charges/${char_bender:id}/capture",
			FixtureQuery{"${char_bender:id}", "char_bender", "id", ""},
			true,
		},
		{
			"${.env:PHONE_NOT_SET|+1234567890}",
			FixtureQuery{"${.env:PHONE_NOT_SET|+1234567890}", ".env", "PHONE_NOT_SET", "+1234567890"},
			true,
		},
		{
			"/v1/customers/${.env:CUST_ID}",
			FixtureQuery{"${.env:CUST_ID}", ".env", "CUST_ID", ""},
			true,
		},
		{
			"${.env:CUST_ID}",
			FixtureQuery{"${.env:CUST_ID}", ".env", "CUST_ID", ""},
			true,
		},
		{
			"${cust_bender:subscriptions.data.[0].id}",
			FixtureQuery{"${cust_bender:subscriptions.data.[0].id}", "cust_bender", "subscriptions.data.[0].id", ""},
			true,
		},
		{
			"${cust_bender:subscriptions.data.[0].name|Unknown Person}",
			FixtureQuery{"${cust_bender:subscriptions.data.[0].name|Unknown Person}", "cust_bender", "subscriptions.data.[0].name", "Unknown Person"},
			true,
		},
		{
			"${cust_bender:billing_details.address.country}",
			FixtureQuery{"${cust_bender:billing_details.address.country}", "cust_bender", "billing_details.address.country", ""},
			true,
		},
		{
			"${cust_bender:billing_details.address.country|San Mateo}",
			FixtureQuery{"${cust_bender:billing_details.address.country|San Mateo}", "cust_bender", "billing_details.address.country", "San Mateo"},
			true,
		},
	}

	for _, test := range tests {
		actualQuery, actualDidMatch := ToFixtureQuery(test.input)
		assert.Equal(t, test.expected, actualQuery)
		assert.Equal(t, test.didMatch, actualDidMatch)
	}
}
