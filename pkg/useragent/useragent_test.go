package useragent

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnameNotEmpty(t *testing.T) {
	if runtime.GOOS != "windows" {
		u := getUname()
		require.NotEmpty(t, u)
		t.Log(u)
	}

	// Unfortunately we don't extract info from windows (GetSystemInfo) at this time
}
