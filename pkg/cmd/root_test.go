package cmd

import (
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

func TestGetPathNoXDG(t *testing.T) {
	actual := Config.GetProfilesFolder("")
	expected, err := homedir.Dir()
	expected += "/.config/stripe"

	assert.Nil(t, err)
	assert.Equal(t, actual, expected)
}

func TestGetPathXDG(t *testing.T) {
	actual := Config.GetProfilesFolder("/some/xdg/path")
	expected := "/some/xdg/path/stripe"

	assert.Equal(t, actual, expected)
}
