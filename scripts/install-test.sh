#!/bin/sh

PACKAGE_MANAGER=${1:-}

if [ $# -eq 0 ]; then
  echo "Error! Missing package manager argument"
  exit 1
fi

if [ "${DRYRUN:-false}" = "true" ]; then
    echo "Dry run mode: PagerDuty alerts will be suppressed"
else
    echo "Live mode: PagerDuty alerts are enabled"
fi

run_install() {
    case $PACKAGE_MANAGER in
    homebrew)
        brew install stripe/stripe-cli/stripe
        stripe --version
    ;;

    homebrew-core)
        brew install stripe
        stripe --version
    ;;

    apt)
        curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg
        echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee -a /etc/apt/sources.list.d/stripe.list
        sudo apt update
        sudo apt install stripe
        stripe --version
    ;;

    yum)
        yum -y install stripe
        stripe --version
    ;;

    scoop)
        scoop bucket add stripe https://github.com/stripe/scoop-stripe-cli.git
        scoop install stripe
        stripe --version
    ;;

    winget)
        winget install --id Stripe.StripeCli --accept-source-agreements --accept-package-agreements
        # winget modifies PATH but the current shell process doesn't inherit the change;
        # explicitly add the WinGet Links directory where the stripe alias was created.
        PATH="$PATH:$(cygpath -u "$LOCALAPPDATA/Microsoft/WinGet/Links")"
        export PATH
        stripe --version
    ;;

    docker)
        # The workflow runs this script inside the stripe/stripe-cli:latest container,
        # so the stripe binary is already present. No install step needed.
        stripe --version
    ;;

    npm)
        npm install -g @stripe/cli
        stripe --version
    ;;

    npm-no-optional)
        npm install -g @stripe/cli --no-optional
        stripe --version
    ;;

    npx)
        npx --yes @stripe/cli --version
    ;;

    *)
        echo "Error! Invalid package manager supplied"
        echo ""
        echo_help
        exit 1
        ;;
    esac
}

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

if [ "$PACKAGE_MANAGER" = "notify" ]; then
    if [ "${OVERALL_RESULT:-failure}" = "success" ]; then
        resolve_pagerduty_alert
    else
        trigger_pagerduty_alert
        exit 1
    fi
    exit 0
fi

if ! run_install
then
    echo "Install failed. Retrying in 30 seconds..."
    sleep 30
    if ! run_install
    then
        echo "Install failed again. Retrying in 60 seconds..."
        sleep 60
        if ! run_install
        then
            echo "Install failed again. Retrying in 120 seconds..."
            sleep 120
            if ! run_install
            then
                echo "Install failed again. Retrying for the last time in 180 seconds..."
                sleep 180
                run_install
                if ! run_install
                then
                exit 1
                fi
            fi
        fi
    fi
fi
