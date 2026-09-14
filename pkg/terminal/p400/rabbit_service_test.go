package p400

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateRabbitServicePayload(t *testing.T) {
	tsCtx := TerminalSessionContext{
		APIKey: "sk_123",
		DeviceInfo: DeviceInfo{
			DeviceClass:   "POS",
			DeviceUUID:    "pos-1234",
			HostOSVersion: "Mac OS",
			HardwareModel: HardwareModel{
				POSInfo: POSInfo{
					Description: "Mac OS:StripeCLI",
				},
			},
			AppModel: AppModel{
				AppID:      "Stripe-CLI-Terminal-Quickstart",
				AppVersion: "https://stripe.com/docs/stripe-cli",
			},
		},
	}

	payload := CreateRabbitServicePayload("clearReaderDisplay", "base64string=", "txn>23456", tsCtx)
	require.NotEmpty(t, payload)
}
