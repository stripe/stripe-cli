package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRemoveKey(t *testing.T) {
	v := viper.New()
	v.Set("remove", "me")
	v.Set("stay", "here")

	nv, err := removeKey(v, "remove")
	require.NoError(t, err)

	require.EqualValues(t, []string{"stay"}, nv.AllKeys())
	require.ElementsMatch(t, []string{"stay", "remove"}, v.AllKeys())
}
