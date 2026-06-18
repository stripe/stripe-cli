#!/bin/sh

if [ "${DRYRUN:-false}" = "true" ]; then
    echo "Dry run mode: PagerDuty alerts will be suppressed"
else
    echo "Live mode: PagerDuty alerts are enabled"
fi

send_pagerduty_event() {
    body="$1"
    if command -v curl >/dev/null 2>&1; then
        curl -s -X POST https://events.pagerduty.com/v2/enqueue \
            -H "Content-Type: application/json" \
            -d "$body"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O- --post-data="$body" --header="Content-Type: application/json" \
            https://events.pagerduty.com/v2/enqueue
    else
        echo "Error: neither curl nor wget available; cannot send PagerDuty notification" >&2
        return 1
    fi
}

trigger_pagerduty_alert() {
    if [ "${DRYRUN:-false}" = "true" ]; then
        echo "Dry run: PagerDuty alert would have fired with:"
        echo "  action:    trigger"
        echo "  dedup_key: gh-actions-stripe-cli-install-test"
        echo "  summary:   Failed to install Stripe CLI on one or more operating systems. Investigate here: https://github.com/stripe/stripe-cli/actions/workflows/install-test.yml"
        echo "  timestamp: $(date)"
        echo "  source:    Stripe CLI GitHub Actions"
        echo "  severity:  critical"
        return 0
    fi
    send_pagerduty_event '{
        "routing_key": "'"$PAGERDUTY_INTEGRATION_KEY"'",
        "event_action": "trigger",
        "dedup_key": "gh-actions-stripe-cli-install-test",
        "payload": {
            "summary": "Failed to install Stripe CLI on one or more operating systems. Investigate here: https://github.com/stripe/stripe-cli/actions/workflows/install-test.yml",
            "source": "Stripe CLI GitHub Actions",
            "severity": "critical",
            "timestamp": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"
        }
    }'
}

resolve_pagerduty_alert() {
    if [ "${DRYRUN:-false}" = "true" ]; then
        echo "Dry run: PagerDuty resolve would have fired with:"
        echo "  action:    resolve"
        echo "  dedup_key: gh-actions-stripe-cli-install-test"
        echo "  summary:   Stripe CLI installation is passing again"
        echo "  source:    Stripe CLI GitHub Actions"
        return 0
    fi
    send_pagerduty_event '{
        "routing_key": "'"$PAGERDUTY_INTEGRATION_KEY"'",
        "event_action": "resolve",
        "dedup_key": "gh-actions-stripe-cli-install-test",
        "payload": {
            "summary": "Stripe CLI installation is passing again",
            "source": "Stripe CLI GitHub Actions",
            "severity": "info"
        }
    }'
}

if [ "${OVERALL_RESULT:-failure}" = "success" ]; then
    resolve_pagerduty_alert
else
    trigger_pagerduty_alert
    exit 1
fi
