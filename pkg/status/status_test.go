package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, responseMap["status"], "up")
	assert.Equal(t, responseMap["message"], "All systems operational")
	assert.Equal(t, responseMap["time"], "July 21, 4:00 +0:00")
	assert.Nil(t, responseMap["statuses"])
}

func TestGetMapVerbose(t *testing.T) {
	response := buildResponse()

	responseMap := response.getMap(true)
	assert.Equal(t, responseMap["status"], "up")
	assert.Equal(t, responseMap["message"], "All systems operational")
	assert.Equal(t, responseMap["time"], "July 21, 4:00 +0:00")

	statuses := responseMap["statuses"].(map[string]string)
	assert.Equal(t, statuses["api"], "up")
	assert.Equal(t, statuses["dashboard"], "up")
	assert.Equal(t, statuses["stripejs"], "up")
	assert.Equal(t, statuses["checkoutjs"], "up")
}

func TestFormatJSON(t *testing.T) {
	response := buildResponse()

	expected := `{
  "message": "All systems operational",
  "status": "up",
  "time": "July 21, 4:00 +0:00"
}`

	formatted, _ := response.FormattedMessage("json", false)
	assert.Equal(t, formatted, expected)
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
	assert.Equal(t, formatted, expected)
}

func TestFormatDefault(t *testing.T) {
	response := buildResponse()

	expected := `✅ All systems operational
As of: July 21, 4:00 +0:00`

	formatted, _ := response.FormattedMessage("default", false)
	assert.Equal(t, formatted, expected)
}

func TestFormatDefaultVerbose(t *testing.T) {
	response := buildResponse()

	expected := `✅ All systems operational
✅ API
✅ Dashboard
✅ Stripe.js
✅ Checkout.js
As of: July 21, 4:00 +0:00`

	formatted, _ := response.FormattedMessage("default", true)
	assert.Equal(t, formatted, expected)
}

func TestEmojification(t *testing.T) {
	assert.Equal(t, "✅", emojifiedStatus("up"))
	assert.Equal(t, "⚠️", emojifiedStatus("degraged"))
	assert.Equal(t, "🔴", emojifiedStatus("down"))
	assert.Equal(t, "", emojifiedStatus("foo"))
}
