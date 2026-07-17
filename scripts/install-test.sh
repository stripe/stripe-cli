#!/bin/sh

PACKAGE_MANAGER=${1:-}

if [ $# -eq 0 ]; then
  echo "Error! Missing package manager argument"
  exit 1
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
        # Re-register the WinGet source AppX package to fix 0x8a15000f "data required is missing".
        powershell.exe -Command 'Add-AppxPackage -DisableDevelopmentMode -Register (Get-AppxPackage Microsoft.DesktopAppInstaller).InstallLocation\AppXManifest.xml -Verbose'
        # Reset the source index to avoid 0x8a15000f "data required is missing" on fresh runners.
        winget source reset --force
        winget source update winget
        # The GitHub Actions Windows image includes the Microsoft Store source,
        # which can block non-interactive installs after a reset by prompting
        # for terms and region data. Remove it and install from the community
        # source directly.
        if winget source list | grep -q 'msstore'; then
            winget source remove msstore
        fi
        winget install --exact --id Stripe.StripeCli --source winget --accept-source-agreements --accept-package-agreements
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
                if ! run_install
                then
                    exit 1
                fi
            fi
        fi
    fi
fi
