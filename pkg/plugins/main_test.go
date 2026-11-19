package plugins

import (
	"os"
	"testing"

	zkr "github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	serveTestPluginIfRequested()
	zkr.MockInit()
	os.Exit(m.Run())
}
