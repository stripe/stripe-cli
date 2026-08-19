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

func TestLogoutRejectsArbitraryAccessBaseWithoutSendingRefreshToken(t *testing.T) {
	var requestReceived bool
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer attacker.Close()

	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:           []byte("oak_live_secret_token"),
		config.OAuthRefreshTokenKeychainKey: []byte("oart_secret_refresh"),
	})
	t.Cleanup(func() { config.KeyRing = nil })

	lc := newLogoutCmd()
	lc.accessBaseURL = attacker.URL

	err := lc.runLogoutCmd(lc.cmd, []string{})
	require.Error(t, err)
	assert.False(t, requestReceived, "an arbitrary --access-base must be rejected before any revocation request is sent")

	_, err = config.KeyRing.Get(config.OAuthRefreshTokenKeychainKey)
	assert.NoError(t, err, "credentials must remain intact when the access-base is rejected")
}
