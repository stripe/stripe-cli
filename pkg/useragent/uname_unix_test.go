// +build !windows

package useragent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnameNotEmpty(t *testing.T) {
	u := getUname()
	require.NotEmpty(t, u)
}
