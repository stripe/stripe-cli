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

trigger_pagerduty_alert() {
    if [ "${DRYRUN:-false}" = "true" ]; then
        echo "Dry run: PagerDuty alert would have fired with:"
        echo "  summary:  Failed to install Stripe CLI on one or more operating systems. Investigate here: https://github.com/stripe/stripe-cli/actions/workflows/install-test.yml"
        echo "  timestamp: $(date)"
        echo "  source:   Stripe CLI GitHub Actions"
        echo "  severity: critical"
        return 0
    fi
    sh -c "$(curl -sL https://raw.githubusercontent.com/martindstone/pagerduty-cli/master/install.sh)"
    pd event alert --routing_key "$PAGERDUTY_INTEGRATION_KEY" \
    --summary "Failed to install Stripe CLI on one or more operating systems. Investigate here: https://github.com/stripe/stripe-cli/actions/workflows/install-test.yml" \
    --timestamp "\"$(date)\"" \
    --source "Stripe CLI GitHub Actions" \
    --severity critical
}

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
                trigger_pagerduty_alert
                exit 1
                fi
            fi
        fi
    fi
fi
