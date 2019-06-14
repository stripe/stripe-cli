package cmd

import (
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

func TestGetPathNoXDG(t *testing.T) {
	actual := profile.GetConfigFolder("")
	expected, err := homedir.Dir()
	expected += "/.config/stripe"

	assert.Nil(t, err)
	assert.Equal(t, actual, expected)
}

func TestGetPathXDG(t *testing.T) {
	actual := profile.GetConfigFolder("/some/xdg/path")
	expected := "/some/xdg/path/stripe"

	assert.Equal(t, actual, expected)
}
