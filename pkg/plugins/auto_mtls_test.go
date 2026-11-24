package plugins

import (
	"os"
	"path/filepath"
	"testing"

	hcplugin "github.com/hashicorp/go-plugin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// testPluginServeEnv, when set, makes the test binary serve as a plugin instead of running
// tests. Its value picks whether the plugin supports AutoMTLS.
const testPluginServeEnv = "STRIPE_CLI_TEST_PLUGIN_SERVE"

const (
	testPluginServeMTLS   = "mtls"
	testPluginServeNoMTLS = "no-mtls"
)

var testServedPlugin = &Plugin{
	Shortname:        "testplugin",
	Binary:           "stripe-cli-testplugin",
	MagicCookieValue: "6B1F9C4E-2D7A-4E83-9B5C-0A8F3D6E1C27",
}

type noopDispatcherGRPC struct{}

func (noopDispatcherGRPC) RunCommand(*proto.AdditionalInfo, []string) error { return nil }

// serveTestPluginIfRequested serves testServedPlugin and exits when the test binary was
// launched as a plugin by testPluginServeEnv.
func serveTestPluginIfRequested() {
	mode := os.Getenv(testPluginServeEnv)
	if mode == "" {
		return
	}

	// A plugin built before AutoMTLS never reads the client certificate, so it keeps
	// serving plaintext even though the CLI sent one.
	if mode == testPluginServeNoMTLS {
		os.Unsetenv("PLUGIN_CLIENT_CERT")
	}

	handshakeConfig, _ := testServedPlugin.getPluginInterface()
	hcplugin.Serve(&hcplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		VersionedPlugins: map[int]hcplugin.PluginSet{
			2: {"main": &CLIPluginGRPC{Impl: noopDispatcherGRPC{}}},
		},
		GRPCServer: hcplugin.DefaultGRPCServer,
	})
	os.Exit(0)
}

// installTestPlugin links the test binary into a plugins dir as testServedPlugin's local
// development build, so that dispensing it skips the checksum and runs this binary.
func installTestPlugin(t *testing.T, mode string) {
	t.Helper()

	testBinary, err := os.Executable()
	require.NoError(t, err)

	pluginsDir := t.TempDir()
	versionDir := filepath.Join(pluginsDir, testServedPlugin.Shortname, localDevelopmentVersion)
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.Symlink(testBinary, filepath.Join(versionDir, testServedPlugin.Binary+GetBinaryExtension())))

	t.Setenv("STRIPE_PLUGINS_PATH", pluginsDir)
	t.Setenv(testPluginServeEnv, mode)
}

func TestDispensePluginInterfaceWithAutoMTLS(t *testing.T) {
	installTestPlugin(t, testPluginServeMTLS)
	cfg := &TestConfig{ConfigFolder: t.TempDir()}
	cfg.InitConfig()

	raw, client, err := testServedPlugin.dispensePluginInterface(cfg, afero.NewOsFs(), localDevelopmentVersion, "", log.WithField("prefix", "test"))
	if client != nil {
		defer client.Kill()
	}
	require.NoError(t, err)

	dispatcher, ok := raw.(DispatcherGRPC)
	require.True(t, ok)
	require.NoError(t, dispatcher.RunCommand(&proto.AdditionalInfo{}, nil))
}

func TestDispensePluginInterfaceWithoutAutoMTLSAsksToUpgrade(t *testing.T) {
	installTestPlugin(t, testPluginServeNoMTLS)
	cfg := &TestConfig{ConfigFolder: t.TempDir()}
	cfg.InitConfig()

	_, client, err := testServedPlugin.dispensePluginInterface(cfg, afero.NewOsFs(), localDevelopmentVersion, "", log.WithField("prefix", "test"))
	if client != nil {
		defer client.Kill()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rebuild it against the latest plugin SDK")
	// The plugin never got to run, so the caller has to print this.
	require.False(t, PluginAlreadyReported(err))
}

func TestAutoMTLSUnsupportedErrorAsksToUpgradeRelease(t *testing.T) {
	err := testServedPlugin.autoMTLSUnsupportedError("1.2.3")
	require.EqualError(t, err, "The testplugin plugin (v1.2.3) is too old to connect to this version of the Stripe CLI. Run `stripe plugin upgrade testplugin` to update it")
}
