package plugins

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestBuildAdditionalInfoSendsCLIVersion(t *testing.T) {
	withCoreVersion(t, "1.53.0")

	info := buildAdditionalInfo(log.NewEntry(log.StandardLogger()), "https://api.example.test", "", "")

	require.Equal(t, "1.53.0", info.GetCliVersion())
	require.Equal(t, "https://api.example.test", info.GetApiBaseUrl())
}

func TestBuildAdditionalInfoSendsSourceBuildVersionUnchanged(t *testing.T) {
	withCoreVersion(t, "master")

	info := buildAdditionalInfo(log.NewEntry(log.StandardLogger()), "", "", "")

	require.Equal(t, "master", info.GetCliVersion())
}
