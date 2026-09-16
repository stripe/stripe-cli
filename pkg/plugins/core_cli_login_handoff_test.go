package plugins

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

type loginHandoffTransport struct {
	target    *url.URL
	transport http.RoundTripper
}

func (t loginHandoffTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.URL.Scheme, c.URL.Host = t.target.Scheme, t.target.Host
	return t.transport.RoundTrip(c)
}

func TestResumableLoginThroughGRPC(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg := &config.Config{LogLevel: "info", Profile: config.Profile{ProfileName: "default"}, ProfilesFile: filepath.Join(t.TempDir(), "config.toml")}
	cfg.InitConfig()
	config.KeyRing = keyring.NewMemoryStore(nil)
	t.Cleanup(func() { config.KeyRing = nil; viper.Reset() })
	var starts, exchanges atomic.Int32
	var approved atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stripecli/oauth2/device/authorization":
			starts.Add(1)
			_ = json.NewEncoder(w).Encode(login.DeviceAuthResponse{DeviceCode: "private-device", UserCode: "CODE", VerificationURI: "https://access.stripe.com/verify", ExpiresIn: 600, Interval: 5})
		case "/stripecli/oauth2/token":
			exchanges.Add(1)
			if !approved.Load() {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(login.OAuthTokenResponse{AccessToken: "oak_rpc_fixture", RefreshToken: "private-refresh", TokenType: "Bearer", ExpiresIn: 3600})
		case "/stripecli/oauth2/token/accounts":
			_, _ = w.Write([]byte(`{"accounts":[{"id":"acct_rpc","name":"RPC fixture","modes":["live"]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(upstream.Close)
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	oldTransport := http.DefaultTransport
	http.DefaultTransport = loginHandoffTransport{target: target, transport: oldTransport}
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	proto.RegisterCoreCLIHelperServer(server, &CoreCLIHelperServer{Impl: NewCoreCLIHelper(context.Background(), cfg, afero.NewOsFs(), "", "", login.DefaultAccessBaseURL)})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := proto.NewCoreCLIHelperClient(conn)
	goClient := &CoreCLIHelperClient{client: client}
	first, err := client.BeginOrResumeLogin(context.Background(), &proto.BeginOrResumeLoginRequest{})
	require.NoError(t, err)
	second, err := client.BeginOrResumeLogin(context.Background(), &proto.BeginOrResumeLoginRequest{})
	require.NoError(t, err)
	assert.Equal(t, first.HandoffId, second.HandoffId)
	assert.True(t, second.Reused)
	assert.EqualValues(t, 1, starts.Load())
	assert.NotContains(t, protojson.Format(first), "private-device")
	approved.Store(true)
	result, err := client.CheckLogin(context.Background(), &proto.CheckLoginRequest{HandoffId: first.HandoffId})
	require.NoError(t, err)
	assert.Equal(t, proto.LoginHandoffState_LOGIN_HANDOFF_STATE_AUTHENTICATED, result.State)
	assert.Equal(t, "acct_rpc", result.AccountId)
	assert.NotContains(t, protojson.Format(result), "private-refresh")
	assert.NotContains(t, protojson.Format(result), "oak_rpc_fixture")
	_, err = client.CheckLogin(context.Background(), &proto.CheckLoginRequest{HandoffId: first.HandoffId})
	require.NoError(t, err)
	assert.EqualValues(t, 1, exchanges.Load())
	decoded, err := goClient.CheckLogin(context.Background(), first.HandoffId)
	require.NoError(t, err)
	assert.Equal(t, login.LoginHandoffAuthenticated, decoded.State)
	_, err = goClient.CheckLogin(context.Background(), "not-a-handoff")
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	details := status.Convert(err).Details()
	require.Len(t, details, 1)
	assert.Equal(t, "invalid_handoff_id", details[0].(*errdetails.ErrorInfo).Reason)
}
