package plugins

import (
	"os"
	"testing"

	zkr "github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	zkr.MockInit()
	dir, err := os.MkdirTemp("", "cli-auth-tests-")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
