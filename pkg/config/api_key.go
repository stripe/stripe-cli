package config

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

const LiveModeKeyLastExpirationWarningField = "live_mode_api_key_last_expiration_warning"
const TestModeKeyLastExpirationWarningField = "test_mode_api_key_last_expiration_warning"

const upcomingExpirationThreshold = 14 * 24 * time.Hour
const imminentExpirationThreshold = 24 * time.Hour

const upcomingExpirationReminderFrequency = 12 * time.Hour

// Useful for stubbing in tests
var timeNow = time.Now
var printWarning = printWarningMessage

type APIKey struct {
	Key        string
	Livemode   bool
	Expiration time.Time
}

func NewAPIKey(key string, expiration time.Time, livemode bool) *APIKey {
	if key == "" {
		return nil
	}

	return &APIKey{
		Key:        key,
		Livemode:   livemode,
		Expiration: expiration,
	}
}

func NewAPIKeyFromString(key string) *APIKey {
	if key == "" {
		return nil
	}

	return &APIKey{
		Key: key,
		// Not guaranteed to be right, but we'll try our best to infer live/test mode
		// via a heuristic
		Livemode: strings.Contains(key, "live"),
		// Expiration intentionally omitted to leave it as the zero value, since
		// it's not known when e.g. a key is passed using an environment variable.
	}
}

func (k *APIKey) WarnIfExpirationSoon(profile *Profile) {
	if k.Expiration.IsZero() {
		return
	}

	remainingValidity := k.Expiration.Sub(timeNow())
	if k.shouldShowImminentExpirationWarning() {
		warnMsg := fmt.Sprintf("Your API key will expire in less than %.0f hours. You can obtain a new key from the Dashboard or `stripe login`.", imminentExpirationThreshold.Hours())
		printWarning(warnMsg)
		_ = k.setLastExpirationWarning(timeNow(), profile)
	} else if k.shouldShowUpcomingExpirationWarning(profile) {
		remainingDays := int(math.Round(remainingValidity.Hours() / 24.0))
		warnMsg := fmt.Sprintf("Your API key will expire in %d days. You can obtain a new key from the Dashboard or `stripe login`.", remainingDays)
		printWarning(warnMsg)
		_ = k.setLastExpirationWarning(timeNow(), profile)
	} else {

	}
}

func (k *APIKey) shouldShowImminentExpirationWarning() bool {
	remainingValidity := k.Expiration.Sub(timeNow())
	return remainingValidity < imminentExpirationThreshold
}

func (k *APIKey) shouldShowUpcomingExpirationWarning(profile *Profile) bool {
	remainingValidity := k.Expiration.Sub(timeNow())
	if remainingValidity < upcomingExpirationThreshold {
		lastWarning := k.fetchLastExpirationWarning(profile)

		if timeNow().Sub(lastWarning) > upcomingExpirationReminderFrequency {
			return true
		}
	}

	return false
}

func (k *APIKey) fetchLastExpirationWarning(profile *Profile) time.Time {
	configKey := profile.GetConfigField(k.expirationWarningField())
	lastWarningTimeString := viper.GetString(configKey)
	lastWarningUnixTime, err := strconv.ParseInt(lastWarningTimeString, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(lastWarningUnixTime, 0)
}

func (k *APIKey) setLastExpirationWarning(warningTime time.Time, profile *Profile) error {
	timeStr := strconv.FormatInt(warningTime.Unix(), 10)
	return profile.WriteConfigField(k.expirationWarningField(), timeStr)
}

func (k *APIKey) expirationWarningField() string {
	if k.Livemode {
		return LiveModeKeyLastExpirationWarningField
	} else {
		return TestModeKeyLastExpirationWarningField
	}
}

func printWarningMessage(message string) {
	formattedMessage := ansi.Color(os.Stderr).Yellow(message).Bold()
	_, err := fmt.Fprintln(os.Stderr, formattedMessage)
	_ = err
}
