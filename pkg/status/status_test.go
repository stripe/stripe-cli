package status

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func buildResponse() Response {
	return Response{
		LargeStatus: "up",
		Message:     "All systems operational",
		Time:        "July 21, 4:00 +0:00",
		Statuses: statuses{
			API:        "up",
			Dashboard:  "up",
			Stripejs:   "up",
			Checkoutjs: "up",
			Webhooks:   "up",
			Emails:     "up",
		},
	}
}

func TestGetMap(t *testing.T) {
	response := buildResponse()

	responseMap := response.getMap(false)
	require.Equal(t, "up", responseMap["status"])
	require.Equal(t, "All systems operational", responseMap["message"])
	require.Equal(t, "July 21, 4:00 +0:00", responseMap["time"])
	require.Nil(t, responseMap["statuses"])
}

func TestGetMapVerbose(t *testing.T) {
	response := buildResponse()

	responseMap := response.getMap(true)
	require.Equal(t, "up", responseMap["status"])
	require.Equal(t, "All systems operational", responseMap["message"])
	require.Equal(t, "July 21, 4:00 +0:00", responseMap["time"])

	statuses := responseMap["statuses"].(map[string]string)
	require.Equal(t, "up", statuses["api"])
	require.Equal(t, "up", statuses["dashboard"])
	require.Equal(t, "up", statuses["stripejs"])
	require.Equal(t, "up", statuses["checkoutjs"])
}

func TestFormatJSON(t *testing.T) {
	response := buildResponse()

	expected := `{
  "message": "All systems operational",
  "status": "up",
  "time": "July 21, 4:00 +0:00"
}`

	formatted, _ := response.FormattedMessage("json", false)
	require.Equal(t, expected, formatted)
}

func TestFormatJSONVerbose(t *testing.T) {
	response := buildResponse()

	expected := `{
  "message": "All systems operational",
  "status": "up",
  "statuses": {
    "api": "up",
    "checkoutjs": "up",
    "dashboard": "up",
    "stripejs": "up"
  },
  "time": "July 21, 4:00 +0:00"
}`

	formatted, _ := response.FormattedMessage("json", true)
	require.Equal(t, expected, formatted)
}

func TestFormatDefault(t *testing.T) {
	response := buildResponse()

	expected := `✔ All systems operational
As of: July 21, 4:00 +0:00`

	formatted, _ := response.FormattedMessage("default", false)
	require.Equal(t, expected, formatted)
}

func TestFormatDefaultVerbose(t *testing.T) {
	response := buildResponse()

	expected := `✔ All systems operational
✔ API
✔ Dashboard
✔ Stripe.js
✔ Checkout.js
As of: July 21, 4:00 +0:00`

	formatted, _ := response.FormattedMessage("default", true)
	require.Equal(t, expected, formatted)
}

func TestEmojification(t *testing.T) {
	require.Equal(t, "✔", emojifiedStatus("up"))
	require.Equal(t, "!", emojifiedStatus("degraded"))
	require.Equal(t, "✘", emojifiedStatus("down"))
	require.Equal(t, "", emojifiedStatus("foo"))
}
