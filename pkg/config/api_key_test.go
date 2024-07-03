package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNewAPIKeyFromString(t *testing.T) {
	sampleLivemodeKeyString := "rk_live_1234"
	sampleTestmodeKeyString := "rk_test_1234"

	livemodeKey := NewAPIKeyFromString(sampleLivemodeKeyString)
	testmodeKey := NewAPIKeyFromString(sampleTestmodeKeyString)

	assert.Equal(t, sampleLivemodeKeyString, livemodeKey.Key)
	assert.True(t, livemodeKey.Livemode)
	assert.Zero(t, livemodeKey.Expiration)

	assert.Equal(t, sampleTestmodeKeyString, testmodeKey.Key)
	assert.False(t, testmodeKey.Livemode)
	assert.Zero(t, testmodeKey.Expiration)
}

func TestWarnIfExpirationSoon(t *testing.T) {
	t.Run("warn repeatedly when expiration is imminent", func(t *testing.T) {
		now := time.Unix(1000, 0)
		expiration := now.Add(imminentExpirationThreshold - 1*time.Hour)

		timeCleanup := setupFakeTimeNow(now)
		defer timeCleanup()

		printed, printWarningCleanup := setupFakePrintWarning()
		defer printWarningCleanup()

		k := &APIKey{
			Key:        "rk_test_1234",
			Livemode:   false,
			Expiration: expiration,
		}

		config, configCleanup := setupTestConfig(k)
		defer configCleanup()

		k.WarnIfExpirationSoon(&config.Profile)
		assert.Equal(t, 1, len(printed.messages))

		k.WarnIfExpirationSoon(&config.Profile)
		assert.Equal(t, 2, len(printed.messages))
	})

	t.Run("warn once per period when expiration is upcoming", func(t *testing.T) {
		now := time.Unix(5000, 0)
		expiration := now.Add(upcomingExpirationThreshold - 1*time.Hour)

		initialTimeCleanup := setupFakeTimeNow(now)
		defer initialTimeCleanup()

		printed, printWarningCleanup := setupFakePrintWarning()
		defer printWarningCleanup()

		k := &APIKey{
			Key:        "rk_test_1234",
			Livemode:   false,
			Expiration: expiration,
		}

		config, configCleanup := setupTestConfig(k)
		defer configCleanup()

		nextTime := now
		for i := 0; i < 4; i++ {
			nextTime = nextTime.Add(upcomingExpirationReminderFrequency + 1*time.Hour)

			advancedTimeCleanup := setupFakeTimeNow(nextTime)
			defer advancedTimeCleanup()

			k.WarnIfExpirationSoon(&config.Profile)
			assert.Equal(t, i+1, len(printed.messages))

			k.WarnIfExpirationSoon(&config.Profile)
			assert.Equal(t, i+1, len(printed.messages))

			k.WarnIfExpirationSoon(&config.Profile)
			assert.Equal(t, i+1, len(printed.messages))
		}
	})

	t.Run("do not warn when expiration is not near", func(t *testing.T) {
		now := time.Unix(900000, 0)
		expiration := now.Add(90 * 24 * time.Hour)

		initialTimeCleanup := setupFakeTimeNow(now)
		defer initialTimeCleanup()

		printed, printWarningCleanup := setupFakePrintWarning()
		defer printWarningCleanup()

		k := &APIKey{
			Key:        "rk_test_1234",
			Livemode:   false,
			Expiration: expiration,
		}

		config, configCleanup := setupTestConfig(k)
		defer configCleanup()

		k.WarnIfExpirationSoon(&config.Profile)
		assert.Equal(t, 0, len(printed.messages))
	})

	t.Run("do not warn when expiration is unset", func(t *testing.T) {
		now := time.Unix(900000, 0)

		initialTimeCleanup := setupFakeTimeNow(now)
		defer initialTimeCleanup()

		printed, printWarningCleanup := setupFakePrintWarning()
		defer printWarningCleanup()

		k := NewAPIKeyFromString("rk_test_1234")

		config, configCleanup := setupTestConfig(k)
		defer configCleanup()

		k.WarnIfExpirationSoon(&config.Profile)
		assert.Equal(t, 0, len(printed.messages))
	})
}

func setupTestConfig(testmodeKey *APIKey) (*Config, func()) {
	uniqueConfig := fmt.Sprintf("config-%d.toml", time.Now().UnixMilli())
	profilesFile := filepath.Join(os.TempDir(), "stripe", uniqueConfig)

	p := Profile{
		DeviceName:     "st-testing",
		ProfileName:    "tests",
		DisplayName:    "test-account-display-name",
		TestModeAPIKey: testmodeKey,
	}

	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	v := viper.New()
	_ = p.writeProfile(v)

	return c, func() {
		_ = os.Remove(profilesFile)
	}
}

// Mocks the result of time.Now as used in api_key.go and returns a cleanup
// function which should be called in a defer in the consuming test.
func setupFakeTimeNow(t time.Time) func() {
	original := timeNow
	timeNow = func() time.Time {
		return t
	}

	return func() {
		timeNow = original
	}
}

// This struct encapsulates the message slice since that's the most idiomatic
// way to retain a pointer to the slice outside of the mocked function
type messageRecorder struct {
	messages []string
}

func setupFakePrintWarning() (*messageRecorder, func()) {
	original := printWarning

	printed := &messageRecorder{}

	printWarning = func(message string) {
		printed.messages = append(printed.messages, message)
	}

	return printed, func() {
		printWarning = original
	}
}
