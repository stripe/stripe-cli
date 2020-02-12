package p400

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetParentTraceID(t *testing.T) {
	expected := "txn!1234>activateTerminal!5678"
	traceid := SetParentTraceID(1234, 5678, "activateTerminal")

	require.Equal(t, expected, traceid, "they should be equal")
}

func TestGeneratePosDeviceID(t *testing.T) {
	var seed int64 = 12345

	expected := "pos-isjlqargbit"
	posid := GeneratePOSDeviceID(seed)

	require.Equal(t, expected, posid, "they should be equal")
}

func TestSetTransactionContext(t *testing.T) {
	tsCtx := TerminalSessionContext{
		DeviceInfo: DeviceInfo{
			DeviceUUID: "pos-isjlqargbit",
		},
	}
	transCtx := SetTransactionContext(tsCtx)

	require.NotNil(t, transCtx.TerminalID, "should have TerminalID field")
	require.NotNil(t, transCtx.OperatorID, "should have OperatorID field")
	require.NotNil(t, transCtx.StartTime, "should have StartTime field")
	require.NotNil(t, transCtx.TransactionID, "should have TransactionID field")
}
