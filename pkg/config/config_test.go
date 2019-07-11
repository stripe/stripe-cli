package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestRemoveKey(t *testing.T) {
	v := viper.New()
	v.Set("remove", "me")
	v.Set("stay", "here")

	nv, err := removeKey(v, "remove")
	assert.NoError(t, err)

	assert.EqualValues(t, []string{"stay"}, nv.AllKeys())
	assert.ElementsMatch(t, []string{"stay", "remove"}, v.AllKeys())
}
