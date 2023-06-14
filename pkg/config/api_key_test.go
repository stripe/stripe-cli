package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewAPIKeyFromString(t *testing.T) {
	sampleLivemodeKeyString := "rk_live_1234"
	sampleTestmodeKeyString := "rk_test_1234"

	livemodeKey := NewAPIKeyFromString(sampleLivemodeKeyString, nil)
	testmodeKey := NewAPIKeyFromString(sampleTestmodeKeyString, nil)

	assert.Equal(t, sampleLivemodeKeyString, livemodeKey.Key)
	assert.True(t, livemodeKey.Livemode)
	assert.Zero(t, livemodeKey.Expiration)
	assert.Nil(t, livemodeKey.profile)

	assert.Equal(t, sampleTestmodeKeyString, testmodeKey.Key)
	assert.False(t, testmodeKey.Livemode)
	assert.Zero(t, testmodeKey.Expiration)
	assert.Nil(t, testmodeKey.profile)
}
