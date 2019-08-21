package samples

import (
	"sort"
	"testing"

	"github.com/thedevsaddam/gojsonq"

	"github.com/stretchr/testify/require"
)

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
