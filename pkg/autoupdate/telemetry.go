package autoupdate

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/version"
)

var telemetryEndpoint = "https://r.stripe.com/0"

func init() {
	if raw := os.Getenv("STRIPE_TELEMETRY_URL"); raw != "" {
		telemetryEndpoint = raw
	}
}

func sendTelemetryEvent(eventName, eventValue string) {
	if isTelemetryOptedOut() {
		return
	}

	data := url.Values{}
	data.Set("client_id", "stripe-cli")
	data.Set("event_id", uuid.NewString())
	data.Set("event_name", eventName)
	data.Set("event_value", eventValue)
	data.Set("created", fmt.Sprint(time.Now().Unix()))
	data.Set("cli_version", version.Version)
	data.Set("os", runtime.GOOS)
	data.Set("arch", runtime.GOARCH)
	data.Set("install_method", "curl")

	req, err := http.NewRequest(http.MethodPost, telemetryEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		log.Debug("autoupdate telemetry: failed to create request: ", err)
		return
	}

	req.Header.Set("origin", "stripe-cli")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug("autoupdate telemetry: failed to send event: ", err)
		return
	}
	resp.Body.Close()
}

func isTelemetryOptedOut() bool {
	for _, key := range []string{"STRIPE_CLI_TELEMETRY_OPTOUT", "DO_NOT_TRACK"} {
		val := strings.ToLower(os.Getenv(key))
		if val == "1" || val == "true" {
			return true
		}
	}
	return false
}
