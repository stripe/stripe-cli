package keyring

import (
	"os"
	"testing"

	zkr "github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	zkr.MockInit()
	os.Exit(m.Run())
}
