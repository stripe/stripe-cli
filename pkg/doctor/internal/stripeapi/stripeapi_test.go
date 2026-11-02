package stripeapi

import (
	"testing"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestLoadCredentialsRejectsTestModeOverride(t *testing.T) {
	cfg := &config.Config{Profile: config.Profile{APIKey: "sk_test_abcdefghij123456"}}
	if _, err := LoadCredentials(cfg); err == nil {
		t.Fatal("expected an error for a test-mode API key override, got nil")
	}
}

func TestLoadCredentialsAcceptsLiveModeOverride(t *testing.T) {
	cfg := &config.Config{Profile: config.Profile{APIKey: "sk_live_abcdefghij123456"}}
	creds, err := LoadCredentials(cfg)
	if err != nil {
		t.Fatalf("unexpected error for a live-mode API key override: %v", err)
	}
	if !creds.Livemode() {
		t.Error("expected resolved credentials to report livemode")
	}
}

func TestLoadCredentialsNilConfig(t *testing.T) {
	if _, err := LoadCredentials(nil); err == nil {
		t.Fatal("expected an error for a nil config, got nil")
	}
}
