package autoupdate

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/installmethod"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/version"
)

// Event names. Kept as constants so the set an analyst has to know about is
// readable in one place, and so a rename cannot miss a call site.
const (
	eventAvailable   = "Auto-Update Available"
	eventSucceeded   = "Auto-Update Succeeded"
	eventFailed      = "Auto-Update Failed"
	eventCheckFailed = "Auto-Update Check Failed"
	eventSkipped     = "Auto-Update Skipped"
	eventPanicked    = "Auto-Update Panicked"
)

// Reasons attached to eventFailed, naming the stage that failed rather than
// leaving an analyst to match on error text.
const (
	reasonDownload = "download"
	reasonStatus   = "status"
	reasonChecksum = "checksum"
	reasonExtract  = "extract"
	reasonReplace  = "replace"
)

// Reasons attached to eventCheckFailed and eventSkipped.
const (
	reasonReleaseFetch = "release_fetch"
	reasonAssetMissing = "asset_missing"
	// A release that published no checksums file, or a checksums fetch that
	// failed: #2070 refuses to stage an update it cannot verify, and this is how
	// often that happens.
	reasonChecksumMissing = "checksum_missing"
	reasonMajorVersion    = "major_version"
)

var telemetryEndpoint = "https://r.stripe.com/0"

func init() {
	if raw := os.Getenv("STRIPE_TELEMETRY_URL"); raw != "" {
		telemetryEndpoint = raw
	}
}

// eventValue renders fields as the space-separated key=value string the
// event_value column already carries, with keys sorted so the same set of
// fields always produces the same string regardless of map iteration order.
func eventValue(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, fields[k]))
	}

	return strings.Join(parts, " ")
}

func sendEvent(eventName string, fields map[string]string) {
	sendTelemetryEvent(eventName, eventValue(fields))
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
	// Detected rather than hardcoded: the CLI's own analytics and the install
	// script both report what pkg/installmethod reports, and a second name for
	// the same install would split every dashboard that groups on this.
	data.Set("install_method", installmethod.Detect(installmethod.OSEnv()))

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

// isTelemetryOptedOut reads the same two variables, through the same helper, as
// cmd/stripe and pkg/stripe. Auto-update runs before the telemetry client in the
// command context exists, so it cannot share that client -- but it can share the
// decision about whether to send anything at all, which is the part that must
// not drift.
func isTelemetryOptedOut() bool {
	return stripe.TelemetryOptedOut(os.Getenv("STRIPE_CLI_TELEMETRY_OPTOUT")) ||
		stripe.TelemetryOptedOut(os.Getenv("DO_NOT_TRACK"))
}
