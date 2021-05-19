# Stripe CLI

![GitHub release (latest by date)](https://img.shields.io/github/v/release/stripe/stripe-cli)
[![Build Status](https://travis-ci.org/stripe/stripe-cli.svg?branch=master)](https://travis-ci.org/stripe/stripe-cli)

The Stripe CLI helps you build, test, and manage your Stripe integration right from the terminal.

**With the CLI, you can:**

- Securely test webhooks without relying on 3rd party software
- Trigger webhook events or resend events for easy testing
- Tail your API request logs in real-time
- Create, retrieve, update, or delete API objects.

![demo](docs/demo.gif)

## Installation

Stripe CLI is available for macOS, Windows, and Linux for distros like Ubuntu, Debian, RedHat and CentOS.

### macOS

Stripe CLI is available on macOS via [Homebrew](https://brew.sh/):

```sh
brew install stripe/stripe-cli/stripe
```

### Linux

Stripe CLI is available for Deb & RPM-based systems via repositories on [Balto](https://stripe.baltorepo.com/stripe-cli/):

#### Debian/Ubuntu (apt)

```sh
curl https://baltocdn.com/stripe/signing.asc | sudo apt-key add -
sudo apt-get install apt-transport-https --yes
echo "deb https://baltocdn.com/stripe/stripe-cli/debian/ all main" | sudo tee /etc/apt/sources.list.d/stripe-stripe-cli-debian.list
sudo apt-get update
sudo apt-get install stripe
```

#### RedHat/CentOS (yum)

```sh
sudo yum install yum-utils
sudo yum-config-manager --add-repo https://baltocdn.com/stripe/stripe-cli/rpm/stripe-stripe-cli-rpm-all.repo
sudo yum install stripe
```

#### Fedora (dnf)

```sh
sudo dnf install 'dnf-command(config-manager)'
sudo dnf config-manager --add-repo https://baltocdn.com/stripe/stripe-cli/rpm/stripe-stripe-cli-rpm-all.repo
sudo dnf install stripe
```

#### OpenSUSE (zypper)

```sh
sudo zypper addrepo https://baltocdn.com/stripe/stripe-cli/rpm/stripe-stripe-cli-rpm-all.repo
sudo zypper refresh
sudo zypper install stripe
```

To use without a package manager, refer to the [installation instructions](https://stripe.com/docs/stripe-cli#install).

### Windows

Stripe CLI is available on Windows via the [Scoop](https://scoop.sh/) package manager:

```sh
scoop bucket add stripe https://github.com/stripe/scoop-stripe-cli.git
scoop install stripe
```

### Docker

The CLI is also available as a Docker image: [`stripe/stripe-cli`](https://hub.docker.com/r/stripe/stripe-cli).

```sh
docker run --rm -it stripe/stripe-cli version
stripe version x.y.z (beta)
```

### Without package managers

Instructions are also available for installing and using the CLI [without a package manager](https://github.com/stripe/stripe-cli/wiki/Installing-and-updating#without-a-package-manager).

## Usage

Installing the CLI provides access to the `stripe` command.

```sh-session
stripe [command]

# Run `--help` for detailed information about CLI commands
stripe [command] help
```

## Commands

The Stripe CLI supports a broad range of commands. Below is some of the most used ones:
- [`login`](https://stripe.com/docs/cli/login)
- [`listen`](https://stripe.com/docs/cli/listen)
- [`trigger`](https://stripe.com/docs/cli/trigger)
- [`logs tail`](https://stripe.com/docs/cli/logs/tail)
- [`events resend`](https://stripe.com/docs/cli/events/resend)
- [`samples`](https://stripe.com/docs/cli/intro_stripe_samples)
- [`serve`](https://stripe.com/docs/cli/serve)
- [`status`](https://stripe.com/docs/cli/status)
- [`config`](https://stripe.com/docs/cli/config)
- [`open`](https://stripe.com/docs/cli/open)
- [`get`, `post` & `delete` commands](https://stripe.com/docs/cli/get)
- [`resource` commands](https://stripe.com/docs/cli/resources)

## Documentation

For a full reference, see the [CLI reference site](https://stripe.com/docs/cli)

## Telemetry

The Stripe CLI includes a telemetry feature that collects some usage data. See our [telemetry reference](https://stripe.com/docs/cli/telemetry) for details.

## Feedback

Got feedback for us? Please don't hesitate to tell us on [feedback](https://stri.pe/cli-feedback).

## Contributing

See [Developing the Stripe CLI](../../wiki/developing-the-stripe-cli) for more info on how to make contributions to this project.

## License
Copyright (c) Stripe. All rights reserved.

Licensed under the [Apache License 2.0 license](blob/master/LICENSE).

