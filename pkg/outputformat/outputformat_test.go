package outputformat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalTOONUsesJSONTags(t *testing.T) {
	type payload struct {
		ID      string `json:"id"`
		Ignored string `json:"ignored,omitempty"`
		Amount  int    `json:"amount"`
	}

	got, err := Marshal(payload{
		ID:     "ch_123",
		Amount: 42,
	}, FormatTOON)
	require.NoError(t, err)

	require.Equal(t, "id: ch_123\namount: 42", string(got))
}

func TestRenderJSONTOONPreservesFieldOrder(t *testing.T) {
	raw := []byte(`{"object":"charge","id":"ch_123","items":[{"sku":"sku_1","qty":1},{"sku":"sku_2","qty":2}]}`)

	got, err := RenderJSON(raw, FormatTOON, false, nil)
	require.NoError(t, err)

	require.Equal(t, "object: charge\nid: ch_123\nitems[2]{sku,qty}:\n  sku_1,1\n  sku_2,2", got)
}

func TestValidateRejectsUnknownFormats(t *testing.T) {
	err := Validate("xml")
	require.EqualError(t, err, `invalid format "xml", must be one of 'json' or 'toon'`)
}

func TestRequestFlagUsageListsFormats(t *testing.T) {
	usage := RequestFlagUsage()
	require.Contains(t, usage, "'json' - Output the response in JSON format (default)")
	require.Contains(t, usage, "'toon' - Output the response in TOON format")
}

func TestStructuredFlagUsageListsFormats(t *testing.T) {
	usage := StructuredFlagUsage("webhook events")
	require.Contains(t, usage, "'json' - Output webhook events in JSON format")
	require.Contains(t, usage, "'toon' - Output webhook events in TOON format")
}
