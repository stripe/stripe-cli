package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

func TestSwitchContextRejectsArbitraryAccessBaseWithoutSendingUAT(t *testing.T) {
	var requestReceived bool
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		assert.Empty(t, r.Header.Get("Authorization"), "attacker-controlled server must never see the UAT")
		w.WriteHeader(http.StatusOK)
	}))
	defer attacker.Close()

	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey: []byte("oak_live_secret_token"),
	})
	t.Cleanup(func() { config.KeyRing = nil })

	sc := &switchContextCmd{accessBaseURL: attacker.URL}
	sc.cmd = newSwitchCmd().cmd.Commands()[0]

	err := sc.run(sc.cmd, []string{})
	require.Error(t, err)
	assert.False(t, requestReceived, "an arbitrary --access-base must be rejected before any request is sent")
}
