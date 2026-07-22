# Stripe CLI

![GitHub release (latest by date)](https://img.shields.io/github/v/release/stripe/stripe-cli)
![Build Status](https://github.com/stripe/stripe-cli/actions/workflows/release.yml/badge.svg)

The Stripe CLI helps you build, test, and manage your Stripe integration right from the terminal.

**With the CLI, you can:**

- Securely test webhooks without relying on 3rd party software
- Trigger webhook events or resend events for easy testing
- Tail your API request logs in real-time
- Create, retrieve, update, or delete API objects.

![demo](docs/demo.gif)

## Installation

Stripe CLI is available for macOS, Windows, and Linux for distros like Ubuntu, Debian, RedHat and CentOS.

### npm (macOS, Linux, Windows)

If you have Node.js >= 18 installed, you can install via `npm`:

```sh
npm install -g @stripe/cli
```

You can also directly execute commands via `npx`, although this won't add `stripe` to your `PATH`:

```sh
npx @stripe/cli login
```

### macOS

**Homebrew:**

```sh
brew install stripe
```

### Linux

**apt (Debian, Ubuntu):**

```sh
curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee -a /etc/apt/sources.list.d/stripe.list
sudo apt update
sudo apt install stripe
```

**yum/dnf (RedHat, Fedora, CentOS):**

```sh
echo -e "[Stripe]\nname=stripe\nbaseurl=https://packages.stripe.dev/stripe-cli-rpm-local/\nenabled=1\ngpgcheck=0" | sudo tee -a /etc/yum.repos.d/stripe.repo
sudo yum install stripe
```

### Windows

**WinGet:**

```sh
winget install Stripe.StripeCLI
```

**Scoop:**

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

**Password Store Setup with Docker**

While test mode doesn’t require password store, you will need to set it up if you wish to perform live mode requests.

> You can also make live mode requests on a per command basis by attaching the `--api-key` flag.

1. Create `entrypoint.sh`

```sh
#!/bin/sh
if ! [ -f ~/.gnupg/trustdb.gpg ] ; then
  chmod 700 ~/.gnupg/
  gpg --quick-generate-key stripe-live # This will generate a gpg key called "stripe-live"
fi
if ! [ -f ~/.password-store/.gpg-id ] ; then
  pass init stripe-live # This will initialize a password store record named "stripe-live", using the gpg key above
  pass insert stripe-live # This will insert value for the password store "stripe-live", which we will put Stripe Live Secret Key in
fi

string="$@"
liveflag="--live"

if [ -z "${string##*$liveflag*}" ] ;then
  OPTS="--api-key $(pass show stripe-live)" # This will use the content of the password store "stripe-live" which was inserted in line 8
fi

#pass insert stripe-live
/bin/stripe  $@ $OPTS
```

2. Create a docker file `Dockerfile-cli`

```sh
FROM  stripe/stripe-cli:vx.x.x
RUN  apk  add  pass  gpg-agent
COPY  ./entrypoint.sh  /entrypoint.sh
ENTRYPOINT  [ "/entrypoint.sh" ]
```

3. Build the docker image

```sh
docker build -t stripe-cli -f Dockerfile-cli .
```

4. Run the docker image with password volumes, replacing `$command` with the appropraite Stripe CLI command (i.e `customers list`)

```sh
docker run --rm -it -v stripe-config://root/.config/stripe/ -v stripe-gpg://root/.gnupg/ -v stripe-pass://root/.password-store/ stripe-cli $command
```

> For live mode requests append `--live` after `$command`.

### Without package managers

Download the latest release for your platform from the [GitHub Releases page](https://github.com/stripe/stripe-cli/releases/latest).

**macOS:**

```sh
tar -xvf stripe_X.X.X_mac-os_ARCH.tar.gz
```

Optionally move the `stripe` binary to `/usr/local/bin` for global access.

**Linux:**

```sh
tar -xvf stripe_X.X.X_linux_x86_64.tar.gz
```

Move the `stripe` binary to a directory on your `PATH`.

**Windows:**

Unzip `stripe_X.X.X_windows_x86_64.zip` and add the path containing `stripe.exe` to your `Path` environment variable.

> **Note:** Anti-virus software may flag the binary as unsafe. This is a false positive; see [issue #692](https://github.com/stripe/stripe-cli/issues/692) for details.

## Upgrading

### npm (macOS, Linux, Windows)

```sh
npm install -g @stripe/cli
```

### macOS

**Homebrew:**

```sh
brew upgrade stripe
```

### Linux

**apt (Debian, Ubuntu):**

```sh
sudo apt update && sudo apt upgrade stripe
```

**yum/dnf (RedHat, Fedora, CentOS):**

```sh
yum update stripe
```

### Windows

**WinGet:**

```sh
winget upgrade Stripe.StripeCLI
```

**Scoop:**

```sh
scoop update stripe
```

### Docker

```sh
docker pull stripe/stripe-cli:latest
```
Because Docker containers are ephemeral, the `stripe login` command isn't supported. Use the [`--api-key` flag](https://docs.stripe.com/cli/api_keys) instead.

### Without package managers

Download the latest release for your platform from the [GitHub Releases page](https://github.com/stripe/stripe-cli/releases/latest) and replace your existing binary.

## Usage

Installing the CLI provides access to the `stripe` command.

```sh-session
stripe [command]

# Run `--help` for detailed information about CLI commands
stripe [command] help
```

## Commands

The Stripe CLI supports a broad range of commands. Below are some of the most used ones:
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

