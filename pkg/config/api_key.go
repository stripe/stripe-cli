package config

import (
	"strings"
	"time"
)

type APIKey struct {
	Key        string
	Livemode   bool
	Expiration time.Time
	profile    *Profile
}

func NewAPIKey(key string, expiration time.Time, livemode bool, profile *Profile) *APIKey {
	return &APIKey{
		Key:        key,
		Livemode:   livemode,
		Expiration: expiration,
		profile:    profile,
	}
}

func NewAPIKeyFromString(key string, profile *Profile) *APIKey {
	return &APIKey{
		Key: key,
		// Not guaranteed to be right, but we'll try our best to infer live/test mode
		// via a heuristic
		Livemode: strings.Contains(key, "live"),
		// Expiration intentionally omitted to leave it as the zero value, since
		// it's not known when e.g. a key is passed using an environment variable.
		profile: profile,
	}
}
