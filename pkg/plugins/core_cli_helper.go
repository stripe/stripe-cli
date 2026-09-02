package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// CoreCLIHelper is the interface that's implemented by the host and called by the plugin.
type CoreCLIHelper interface {
	Echo(input string) (string, error)
	SendAnalytics(eventName string, eventValue string) error
	KeychainGetPassword(key string) (string, bool, error)
	KeychainSetPassword(key string, value string) error
	KeychainDeletePassword(key string) (bool, error)
	KeychainFindCredentials() ([]string, error)
	RunPeerPlugin(pluginName string, args []string, cwd string) error
	ResolveCredentials(livemode bool) (token string, stripeContext string, resolvedLivemode bool, err error)
	ResolveCredentialsForAnyMode(livemode bool) (token string, stripeContext string, resolvedLivemode bool, err error)
	// SwitchContext switches the active authorized account/mode context, the same way
	// `stripe switch context` does. If accountID is empty, shows an interactive picker;
	// switched is false if the user cancels it, in which case the other return values are empty.
	SwitchContext(accountID string, livemode bool) (resultAccountID string, accountName string, resultLivemode bool, switched bool, err error)
	// Login starts a Stripe CLI login, the same way `stripe login --new-session` does when run
	// interactively: it revokes any existing OAuth session first (so this works even if the
	// stored credential is expired or revoked), then runs the normal login flow, printing the
	// same output and opening the browser only after the user presses enter.
	// timeoutSeconds bounds how long it waits for the user to complete authentication (0 waits
	// indefinitely, matching `stripe login`); loggedIn is false if that timeout elapses or the
	// attempt is otherwise canceled first, in which case the other return values are empty and
	// the caller should call Login again to keep waiting.
	Login(timeoutSeconds int32) (accountID string, accountName string, livemode bool, loggedIn bool, err error)
}

type CoreCLIHelperClient struct {
	client proto.CoreCLIHelperClient
}

func (c *CoreCLIHelperClient) Echo(input string) (string, error) {
	resp, err := c.client.Echo(context.Background(), &proto.EchoRequest{Input: input})
	if err != nil {
		return "", err
	}
	return resp.Output, nil
}

func (c *CoreCLIHelperClient) SendAnalytics(eventName string, eventValue string) error {
	_, err := c.client.SendAnalytics(context.Background(), &proto.SendAnalyticsRequest{
		EventName:  eventName,
		EventValue: eventValue,
	})
	return err
}

func (c *CoreCLIHelperClient) KeychainGetPassword(key string) (string, bool, error) {
	resp, err := c.client.KeychainGetPassword(context.Background(), &proto.KeychainGetPasswordRequest{Key: key})
	if err != nil {
		return "", false, err
	}
	return resp.Value, resp.Found, nil
}

func (c *CoreCLIHelperClient) KeychainSetPassword(key string, value string) error {
	_, err := c.client.KeychainSetPassword(context.Background(), &proto.KeychainSetPasswordRequest{
		Key:   key,
		Value: value,
	})
	return err
}

func (c *CoreCLIHelperClient) KeychainDeletePassword(key string) (bool, error) {
	resp, err := c.client.KeychainDeletePassword(context.Background(), &proto.KeychainDeletePasswordRequest{Key: key})
	if err != nil {
		return false, err
	}
	return resp.Deleted, nil
}

func (c *CoreCLIHelperClient) KeychainFindCredentials() ([]string, error) {
	resp, err := c.client.KeychainFindCredentials(context.Background(), &proto.KeychainFindCredentialsRequest{}) //nolint:staticcheck
	if err != nil {
		return nil, err
	}
	return resp.Keys, nil
}

func (c *CoreCLIHelperClient) RunPeerPlugin(pluginName string, args []string, cwd string) error {
	_, err := c.client.RunPeerPlugin(context.Background(), &proto.RunPeerPluginRequest{
		PluginName: pluginName,
		Args:       args,
		Cwd:        cwd,
	})
	return err
}

func (c *CoreCLIHelperClient) ResolveCredentials(livemode bool) (string, string, bool, error) {
	resp, err := c.client.ResolveCredentials(context.Background(), &proto.ResolveCredentialsRequest{Livemode: livemode})
	if err != nil {
		return "", "", false, err
	}
	return resp.Token, resp.StripeContext, resp.Livemode, nil
}

func (c *CoreCLIHelperClient) ResolveCredentialsForAnyMode(livemode bool) (string, string, bool, error) {
	resp, err := c.client.ResolveCredentialsForAnyMode(context.Background(), &proto.ResolveCredentialsRequest{Livemode: livemode})
	if err != nil {
		return "", "", false, err
	}
	return resp.Token, resp.StripeContext, resp.Livemode, nil
}

func (c *CoreCLIHelperClient) SwitchContext(accountID string, livemode bool) (string, string, bool, bool, error) {
	resp, err := c.client.SwitchContext(context.Background(), &proto.SwitchContextRequest{AccountId: accountID, Livemode: livemode})
	if err != nil {
		return "", "", false, false, err
	}
	return resp.AccountId, resp.AccountName, resp.Livemode, resp.Switched, nil
}

func (c *CoreCLIHelperClient) Login(timeoutSeconds int32) (string, string, bool, bool, error) {
	resp, err := c.client.Login(context.Background(), &proto.LoginRequest{TimeoutSeconds: timeoutSeconds})
	if err != nil {
		return "", "", false, false, err
	}
	return resp.AccountId, resp.AccountName, resp.Livemode, resp.LoggedIn, nil
}

type CoreCLIHelperServer struct {
	proto.CoreCLIHelperServer
	Impl CoreCLIHelper
}

func (s *CoreCLIHelperServer) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	output, err := s.Impl.Echo(req.Input)
	if err != nil {
		return nil, err
	}
	return &proto.EchoResponse{Output: output}, nil
}

func (s *CoreCLIHelperServer) SendAnalytics(ctx context.Context, req *proto.SendAnalyticsRequest) (*proto.SendAnalyticsResponse, error) {
	err := s.Impl.SendAnalytics(req.EventName, req.EventValue)
	if err != nil {
		return nil, err
	}
	return &proto.SendAnalyticsResponse{}, nil
}

func (s *CoreCLIHelperServer) KeychainGetPassword(ctx context.Context, req *proto.KeychainGetPasswordRequest) (*proto.KeychainGetPasswordResponse, error) {
	value, found, err := s.Impl.KeychainGetPassword(req.Key)
	if err != nil {
		return nil, err
	}
	return &proto.KeychainGetPasswordResponse{Value: value, Found: found}, nil
}

func (s *CoreCLIHelperServer) KeychainSetPassword(ctx context.Context, req *proto.KeychainSetPasswordRequest) (*proto.KeychainSetPasswordResponse, error) {
	err := s.Impl.KeychainSetPassword(req.Key, req.Value)
	if err != nil {
		return nil, err
	}
	return &proto.KeychainSetPasswordResponse{}, nil
}

func (s *CoreCLIHelperServer) KeychainDeletePassword(ctx context.Context, req *proto.KeychainDeletePasswordRequest) (*proto.KeychainDeletePasswordResponse, error) {
	deleted, err := s.Impl.KeychainDeletePassword(req.Key)
	if err != nil {
		return nil, err
	}
	return &proto.KeychainDeletePasswordResponse{Deleted: deleted}, nil
}

func (s *CoreCLIHelperServer) KeychainFindCredentials(ctx context.Context, req *proto.KeychainFindCredentialsRequest) (*proto.KeychainFindCredentialsResponse, error) { //nolint:staticcheck
	keys, err := s.Impl.KeychainFindCredentials()
	if err != nil {
		return nil, err
	}
	return &proto.KeychainFindCredentialsResponse{Keys: keys}, nil //nolint:staticcheck
}

func (s *CoreCLIHelperServer) RunPeerPlugin(ctx context.Context, req *proto.RunPeerPluginRequest) (*proto.RunPeerPluginResponse, error) {
	err := s.Impl.RunPeerPlugin(req.PluginName, req.Args, req.Cwd)
	if err != nil {
		return nil, err
	}
	return &proto.RunPeerPluginResponse{}, nil
}

func (s *CoreCLIHelperServer) ResolveCredentials(ctx context.Context, req *proto.ResolveCredentialsRequest) (*proto.ResolveCredentialsResponse, error) {
	token, stripeContext, livemode, err := s.Impl.ResolveCredentials(req.Livemode)
	if err != nil {
		return nil, err
	}
	return &proto.ResolveCredentialsResponse{Token: token, StripeContext: stripeContext, Livemode: livemode}, nil
}

func (s *CoreCLIHelperServer) ResolveCredentialsForAnyMode(ctx context.Context, req *proto.ResolveCredentialsRequest) (*proto.ResolveCredentialsResponse, error) {
	token, stripeContext, livemode, err := s.Impl.ResolveCredentialsForAnyMode(req.Livemode)
	if err != nil {
		return nil, err
	}
	return &proto.ResolveCredentialsResponse{Token: token, StripeContext: stripeContext, Livemode: livemode}, nil
}

func (s *CoreCLIHelperServer) SwitchContext(ctx context.Context, req *proto.SwitchContextRequest) (*proto.SwitchContextResponse, error) {
	accountID, accountName, livemode, switched, err := s.Impl.SwitchContext(req.AccountId, req.Livemode)
	if err != nil {
		return nil, err
	}
	return &proto.SwitchContextResponse{AccountId: accountID, AccountName: accountName, Livemode: livemode, Switched: switched}, nil
}

func (s *CoreCLIHelperServer) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	accountID, accountName, livemode, loggedIn, err := s.Impl.Login(req.TimeoutSeconds)
	if err != nil {
		return nil, err
	}
	return &proto.LoginResponse{AccountId: accountID, AccountName: accountName, Livemode: livemode, LoggedIn: loggedIn}, nil
}

// coreCLIHelper is the real implementation of the CoreCLIHelper interface.
type coreCLIHelper struct {
	ctx    context.Context
	config config.IConfig
	fs     afero.Fs

	// apiBaseURL, dashboardBaseURL, and accessBaseURL are the --api-base/--dashboard-base/
	// --access-base values the user explicitly passed to the CLI, if any. They're forwarded
	// to peer plugins run via RunPeerPlugin and used for SwitchContext, so a plugin-triggered
	// action targets the same environment as the CLI that launched the original plugin.
	apiBaseURL       string
	dashboardBaseURL string
	accessBaseURL    string
}

var _ CoreCLIHelper = &coreCLIHelper{}

type pendingKeychainValue struct {
	value     string
	expiresAt time.Time
}

var (
	keychainVisibilityRetryTimeout  = 1500 * time.Millisecond
	keychainVisibilityRetryEnabled  = runtime.GOOS == "darwin"
	keychainVisibilityNow           = time.Now
	keychainVisibilityPendingMu     sync.Mutex
	keychainVisibilityPendingValues = map[string]pendingKeychainValue{}
)

func readKeychainPassword(key string) (string, bool, error) {
	data, err := config.KeyRing.Get(key)
	if err == nil {
		return string(data), true, nil
	}
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return "", false, nil
	}
	return "", false, err
}

func rememberPendingKeychainValue(key string, value string) {
	if !keychainVisibilityRetryEnabled {
		return
	}

	keychainVisibilityPendingMu.Lock()
	defer keychainVisibilityPendingMu.Unlock()

	keychainVisibilityPendingValues[key] = pendingKeychainValue{
		value:     value,
		expiresAt: keychainVisibilityNow().Add(keychainVisibilityRetryTimeout),
	}
}

func pendingKeychainValueFor(key string) (string, bool) {
	if !keychainVisibilityRetryEnabled {
		return "", false
	}

	keychainVisibilityPendingMu.Lock()
	defer keychainVisibilityPendingMu.Unlock()

	pending, ok := keychainVisibilityPendingValues[key]
	if !ok {
		return "", false
	}
	if !keychainVisibilityNow().Before(pending.expiresAt) {
		delete(keychainVisibilityPendingValues, key)
		return "", false
	}

	return pending.value, true
}

func clearPendingKeychainValue(key string) {
	keychainVisibilityPendingMu.Lock()
	defer keychainVisibilityPendingMu.Unlock()
	delete(keychainVisibilityPendingValues, key)
}

// loginSwitchContext is a package variable so tests can stub out the network/keychain calls
// made by coreCLIHelper.SwitchContext.
var loginSwitchContext = login.SwitchContext

// loginRevokeToken and loginLogin are package variables so tests can stub out the network/
// keychain calls made by coreCLIHelper.Login.
var (
	loginRevokeToken = login.RevokeToken
	loginLogin       = login.Login
)

// NewCoreCLIHelper creates a new CoreCLIHelper with the given context, config, and filesystem.
// apiBaseURL, dashboardBaseURL, and accessBaseURL should be empty unless the user explicitly
// passed --api-base/--dashboard-base/--access-base to the CLI.
func NewCoreCLIHelper(ctx context.Context, cfg config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL, accessBaseURL string) CoreCLIHelper {
	return &coreCLIHelper{
		ctx:              ctx,
		config:           cfg,
		fs:               fs,
		apiBaseURL:       apiBaseURL,
		dashboardBaseURL: dashboardBaseURL,
		accessBaseURL:    accessBaseURL,
	}
}

// Echo echoes the input string.
func (h *coreCLIHelper) Echo(input string) (string, error) {
	fmt.Printf("[ECHO] %s\n", input)
	return input, nil
}

// SendAnalytics sends a telemetry event to the analytics service.
func (h *coreCLIHelper) SendAnalytics(eventName string, eventValue string) error {
	// Get the telemetry client from the context
	telemetryClient := stripe.GetTelemetryClient(h.ctx)
	if telemetryClient == nil {
		// If no telemetry client is available, silently skip
		return nil
	}

	// Send the event via the telemetry client
	telemetryClient.SendEvent(h.ctx, eventName, eventValue)
	return nil
}

// KeychainGetPassword retrieves a password from the system keychain.
func (h *coreCLIHelper) KeychainGetPassword(key string) (string, bool, error) {
	value, found, err := readKeychainPassword(key)
	if err != nil {
		return "", false, err
	}

	pendingValue, hasPendingValue := pendingKeychainValueFor(key)
	switch {
	case hasPendingValue && found && value == pendingValue:
		clearPendingKeychainValue(key)
		return value, true, nil
	case hasPendingValue:
		return pendingValue, true, nil
	default:
		return value, found, nil
	}
}

// KeychainSetPassword stores a password in the system keychain.
func (h *coreCLIHelper) KeychainSetPassword(key string, value string) error {
	if err := config.KeyRing.Set(key, []byte(value), ""); err != nil {
		return err
	}

	rememberPendingKeychainValue(key, value)
	return nil
}

// KeychainDeletePassword removes a password from the system keychain.
func (h *coreCLIHelper) KeychainDeletePassword(key string) (bool, error) {
	clearPendingKeychainValue(key)

	err := config.KeyRing.Remove(key)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// KeychainFindCredentials returns keychain keys that are present in the credential store.
// It takes a best-effort approach: probing for the one key plugins are most likely to need
// (the default profile's live mode API key), covering both the OS keychain and the
// plain-file fallback via readKeychainPassword.
//
// Deprecated: full OS-level keychain enumeration is complex and platform-specific.
func (h *coreCLIHelper) KeychainFindCredentials() ([]string, error) {
	// "default" is hardcoded for best-effort backwards compatibility
	key := "default." + config.LiveModeAPIKeyName
	_, exists, err := readKeychainPassword(key)
	if err != nil || !exists {
		return []string{}, err
	}
	return []string{key}, nil
}

// ResolveCredentials delegates to Profile.ResolveCredentials and returns the token,
// Stripe-Context header value, and effective livemode. stripeContext is empty for plain API keys.
func (h *coreCLIHelper) ResolveCredentials(livemode bool) (string, string, bool, error) {
	creds, err := h.config.GetProfile().ResolveCredentials(livemode)
	return credentialsResult(creds, err)
}

// ResolveCredentialsForAnyMode delegates to Profile.ResolveCredentialsForAnyMode and
// returns the token, Stripe-Context header value, and effective livemode. Unlike
// ResolveCredentials, it resolves credentials for whichever mode is actually active
// if the requested livemode doesn't match, instead of failing.
func (h *coreCLIHelper) ResolveCredentialsForAnyMode(livemode bool) (string, string, bool, error) {
	creds, err := h.config.GetProfile().ResolveCredentialsForAnyMode(livemode)
	return credentialsResult(creds, err)
}

func credentialsResult(creds stripe.Credentials, err error) (string, string, bool, error) {
	if err != nil {
		return "", "", false, err
	}
	resolvedLivemode := false
	if creds.OAKLivemode != nil {
		resolvedLivemode = *creds.OAKLivemode
	}
	return creds.Token, creds.OAKContext, resolvedLivemode, nil
}

// RunPeerPlugin looks up and runs the named plugin with the given arguments.
// cwd sets the working directory for the plugin process; an empty string uses the current directory.
func (h *coreCLIHelper) RunPeerPlugin(pluginName string, args []string, cwd string) error {
	plugin, err := LookUpPlugin(h.ctx, h.config, h.fs, pluginName)
	if err != nil {
		return fmt.Errorf("peer plugin %q not found: %w", pluginName, err)
	}
	cfg, ok := h.config.(*config.Config)
	if !ok {
		return errorcategory.Errorf(errorcategory.Internal, "could not run peer plugin %q: config type mismatch", pluginName)
	}
	return plugin.Run(h.ctx, cfg, h.fs, args, cwd, "", h.apiBaseURL, h.dashboardBaseURL, h.accessBaseURL)
}

// SwitchContext switches the active authorized account/mode context, the same way
// `stripe switch context` does.
func (h *coreCLIHelper) SwitchContext(accountID string, livemode bool) (string, string, bool, bool, error) {
	cfg, ok := h.config.(*config.Config)
	if !ok {
		return "", "", false, false, errorcategory.Errorf(errorcategory.Internal, "could not switch context: config type mismatch")
	}
	accessBaseURL := h.accessBaseURL
	if accessBaseURL == "" {
		accessBaseURL = login.DefaultAccessBaseURL
	}
	result, err := loginSwitchContext(h.ctx, accessBaseURL, cfg, accountID, livemode)
	if err != nil {
		return "", "", false, false, err
	}
	if result == nil {
		return "", "", false, false, nil
	}
	return result.Account.ID, result.Account.Name, result.Mode == "live", true, nil
}

// Login starts a Stripe CLI login, the same way `stripe login --new-session` does when run
// interactively.
func (h *coreCLIHelper) Login(timeoutSeconds int32) (string, string, bool, bool, error) {
	cfg, ok := h.config.(*config.Config)
	if !ok {
		return "", "", false, false, errorcategory.Errorf(errorcategory.Internal, "could not log in: config type mismatch")
	}
	dashboardBaseURL := h.dashboardBaseURL
	if dashboardBaseURL == "" {
		dashboardBaseURL = stripe.DefaultDashboardBaseURL
	}
	accessBaseURL := h.accessBaseURL
	if accessBaseURL == "" {
		accessBaseURL = login.DefaultAccessBaseURL
	}

	ctx := h.ctx
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(h.ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	// Same as `stripe login --new-session`: revoke any existing OAuth session before starting a
	// new one, so this works even if the stored credential is expired or revoked.
	if uat, _ := cfg.Profile.GetUAT(); strings.HasPrefix(uat, "oak_") {
		if err := loginRevokeToken(ctx, accessBaseURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: token revocation failed: %s\n", err)
		}
	}

	if err := loginLogin(ctx, dashboardBaseURL, accessBaseURL, cfg); err != nil {
		if ctx.Err() != nil {
			return "", "", false, false, nil
		}
		return "", "", false, false, err
	}

	livemode := false
	if ac, _ := config.GetActiveContext(); ac != nil {
		livemode = ac.Livemode
	}
	return cfg.Profile.AccountID, cfg.Profile.DisplayName, livemode, true, nil
}
