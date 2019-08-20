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
	require.Equal(t, responseMap["status"], "up")
	require.Equal(t, responseMap["message"], "All systems operational")
	require.Equal(t, responseMap["time"], "July 21, 4:00 +0:00")
	require.Nil(t, responseMap["statuses"])
}

func TestGetMapVerbose(t *testing.T) {
	response := buildResponse()

	responseMap := response.getMap(true)
	require.Equal(t, responseMap["status"], "up")
	require.Equal(t, responseMap["message"], "All systems operational")
	require.Equal(t, responseMap["time"], "July 21, 4:00 +0:00")

	statuses := responseMap["statuses"].(map[string]string)
	require.Equal(t, statuses["api"], "up")
	require.Equal(t, statuses["dashboard"], "up")
	require.Equal(t, statuses["stripejs"], "up")
	require.Equal(t, statuses["checkoutjs"], "up")
}

func TestFormatJSON(t *testing.T) {
	response := buildResponse()

	expected := `{
  "message": "All systems operational",
  "status": "up",
  "time": "July 21, 4:00 +0:00"
}`

	formatted, _ := response.FormattedMessage("json", false)
	require.Equal(t, formatted, expected)
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
	require.Equal(t, formatted, expected)
}

func TestFormatDefault(t *testing.T) {
	response := buildResponse()

	expected := `✔ All systems operational
As of: July 21, 4:00 +0:00`

	formatted, _ := response.FormattedMessage("default", false)
	require.Equal(t, formatted, expected)
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
	require.Equal(t, formatted, expected)
}

func TestEmojification(t *testing.T) {
	require.Equal(t, "✔", emojifiedStatus("up"))
	require.Equal(t, "!", emojifiedStatus("degraded"))
	require.Equal(t, "✘", emojifiedStatus("down"))
	require.Equal(t, "", emojifiedStatus("foo"))
}
