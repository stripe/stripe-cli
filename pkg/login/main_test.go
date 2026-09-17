package login

import (
	"os"
	"testing"

	zkr "github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	zkr.MockInit()
	if os.Getenv("STRIPE_HANDOFF_TEST_PROCESS") == "1" {
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "cli-auth-tests-")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
