# Stripe CLI

[![Build Status](https://travis-ci.com/stripe/stripe-cli.svg?token=eQWDVpt1sJR63TxbC1KA&branch=master)](https://travis-ci.com/stripe/stripe-cli)

_🏗 The Stripe CLI is currently in beta! We're working on more features to make the experience great. If you have any [feedback](https://stri.pe/cli-feedback), find [issues](https://github.com/stripe/stripe-cli/issues), or would like to be involved in more active testing, please let us know!_

The Stripe CLI is a command-line interface for Stripe that can:

1. `login` to your Stripe account and authenticate the CLI
2. `listen` for webhooks and forward them to a local server
3. Run resource commands for things like `stripe charges create`
4. Run `get`, `post`, and `delete` commands to the Stripe API
5. `trigger` a limited set of webhook events
6. Tail your test mode API request logs
7. Pull Stripe status from status.stripe.com

The main focus for this initial release is to improve the developer experience while integrating and testing webhooks. Interactions through the CLI are currently limited to test mode only.

## Table of Contents

* [Stripe CLI](#stripe-cli)
  * [Installation](#installation)
    * [macOS](#macos)
    * [Linux](#linux)
    * [Windows](#windows)
    * [Docker](#docker)
  * [Commands](#commands)
    * [login](#login)
    * [listen](#listen)
    * [Resource commands](#resource-commands)
    * [get, post, and delete](#get-post-and-delete)
    * [trigger](#trigger)
    * [logs tail](#logs-tail)
    * [samples](#samples)
    * [status](#status)
    * [config](#config)
    * [open](#open)
  * [Developing the Stripe CLI](#developing-the-stripe-cli)
    * [Installation](#installation-1)
    * [Linting](#linting)
    * [Tests](#tests)
    * [Releasing](#releasing)

## Installation

### macOS

_homebrew:_

Run:

```sh
$ brew install stripe/stripe-cli/stripe
```

### Linux

_Debian/Ubuntu-based distributions_:

1. Add Bintray's GPG key to the apt sources keyring:

```sh
$ sudo apt-key adv --keyserver hkp://pool.sks-keyservers.net --recv-keys 379CE192D401AB61
```

2. Add stripe-cli's apt repository to the apt sources list:

```sh
$ echo "deb https://dl.bintray.com/stripe/stripe-cli-deb stable main" | sudo tee -a /etc/apt/sources.list
```

3. Update the package list:

```sh
$ sudo apt-get update
```

4. Install the CLI:

```sh
$ sudo apt-get install stripe
```

_RedHat/CentOS-based distributions:_

1. Add stripe-cli's yum repository to the yum sources list:

```sh
$ wget https://bintray.com/stripe/stripe-cli-rpm/rpm -O bintray-stripe-stripe-cli-rpm.repo && sudo mv bintray-stripe-stripe-cli-rpm.repo /etc/yum.repos.d/
```

2. Update the package list:

```sh
$ sudo yum update
```

3. Install the CLI:

```sh
$ sudo yum install stripe
```

### Windows

_scoop:_

1. Run:

```sh
$ scoop bucket add stripe https://github.com/stripe/scoop-stripe-cli.git
```

2. Run

```sh
$ scoop install stripe
```

### Docker

The CLI is also available as a Docker image: [`stripe/stripe-cli`](https://hub.docker.com/r/stripe/stripe-cli).

```sh
$ docker run --rm -it stripe/stripe-cli version
stripe version x.y.z (beta)
```

### Without package managers

Instructions are also available for installing and using the CLI [without a package manager](https://github.com/stripe/stripe-cli/wiki/Installing-and-updating#without-a-package-manager).

## Commands

### `login`

The Stripe CLI runs commands using a global configuration or project-specific configuration. To configure the CLI globally, run:

```sh
$ stripe login
```

You'll be redirected to the Stripe dashboard to confirm that you want to give access to your account to the CLI. After confirming, a new API key will be created and returned to the CLI.

You can create project-specific configurations with the `--project-name` flag, which can be used in any context. To create an initial configuration:

```sh
$ stripe login --project-name=rocket-rides
```

If you do not provide the `--project-name` flag for a command, it will default to the global configuration.

All configurations are stored in `~/.config/stripe/config.toml` but you can use the [`XDG_CONFIG_HOME`](https://wiki.archlinux.org/index.php/XDG_Base_Directory) environment variable to override this location.

You can also provide an API key manually by passing the `--interactive` flag:

```sh
$ stripe login --interactive
Enter your API key: sk_test_foobar
Your API key is: sk_test_**obar
How would you like to identify this device in the Stripe Dashboard? [default: st-tomer1]
You're configured and all set to get started
```

### `listen`

The `listen` command establishes a direct connection with Stripe, delivering webhook events to your computer directly. Stripe will forward all webhooks tied to the Stripe account for the a given API key.

> **Note:** You do not need to configure any webhook endpoints in your Dashboard to receive webhooks with the CLI.

By default, `listen` accepts all webhook events displays them in your terminal. To forward events to your local app, use the `--forward-to` flag with the location:

* `--forward-to localhost:3000`
* `--forward-to https://example.com/hooks`

Using `--forward-to` will return a [webhook signing secret](https://stripe.com/docs/webhooks/signatures), which you can add to your application's configuration:

```sh
$ stripe listen --forward-to https://example.com/hooks
> Ready! Your webhook signing secret is whsec_oZ8nus9PHnoltEtWZ3pGITZdeHWHoqnL (^C to quit)
```

The webhook signing secret provided will not change between restarts to the `listen` command.

You can specify which events you want to listen to using `--events` with a comma-separated [list of Stripe events](https://stripe.com/docs/api/events/list).

```sh
$ stripe listen --events=payment_intent.created,payment_intent.succeeded
```

You may have webhook endpoints you've already configured with specific Stripe events in your Dashboard. The Stripe CLI can automatically listen to those events with the `--load-from-webhooks-api` flag, used alongside the `--forward-to` flag. This will read any endpoints configured in test mode for your account and forward associated events to the provided URL:

```sh
$ stripe listen --load-from-webhooks-api --forward-to https://example.com/hooks
```

> **Note:** You will receive events for all interactions on your Stripe account. There is currently no way to limit events to only those that a specific user created.

Should you need to also listen to connect events for all connected accounts, you can use the separate `--forward-connect-to` flag:

```sh
$ stripe listen --forward-to localhost:3000/webhook --forward-connect-to localhost:3000/connect_webhook
```

### Resource commands

You can easily make API requests using the CLI:

```sh
$ stripe charges retrieve ch_123
$ stripe charges create amount=100 currency=usd source=tok_visa
```

For a full list of available resources, type `stripe resources` or the [wiki page](https://github.com/stripe/stripe-cli/wiki/resources:-available-commands#available-commands).

### `get`, `post`, and `delete`

The CLI has three commands that let you interact with the Stripe API in test mode. You can easily make `GET`, `POST`, and `DELETE` commands with the Stripe CLI.

For example, you can retrieve a specific charge:

```sh
$ stripe get /charges/ch_123
```

You can also pass data in using the `-d` flag:

```
$ stripe post /charges -d amount=100 -d source=tok_visa -d currency=usd
```

These commands support many of the features on the Stripe API (e.g. selecting a version, pagination, and expansion) through command-line flags, so you won't need to provide specific headers. For a full list of supported flags, see the [wiki page](https://github.com/stripe/stripe-cli/wiki/http-commands:-advanced-options).

You can pipe the output of these commands to other tools. For example, you could use [jq](https://stedolan.github.io/jq/) to extract information from JSON the API returns, and then use that information to trigger other API requests.

Here’s a simple example that lists `past_due` subscriptions, extracts the IDs, and cancels those subscriptions:

```sh
$ stripe get /subscriptions -d status=past_due | jq ".data[].id" | xargs -I % -p stripe delete /subscriptions/%
```

### `trigger`

The CLI will allow you to trigger a few test webhook events to conduct local testing. These test webhook events are real objects on the API and may trigger other webhook events as part of the test (e.g. triggering `payment_intent.succeeded` will also trigger `payment_intent.created`).

The webhook events we currently support are listed in the CLI help or on the [wiki page](https://github.com/stripe/stripe-cli/wiki/trigger:-supported-events).

To trigger an event, run:

```sh
$ stripe trigger <event>
```

### `logs tail`
`logs tail` establishes a direct connection with Stripe and enables you to tail your test mode Stripe API request logs in real-time from your terminal.

By default, `logs tail` will display all of your test mode request logs. To begin log tailing, run:

```sh
$ stripe logs tail
```

A number of built-in filtering options are also supported:
* `--filter-account`, *(Connect only)* supports `connect_in` (incoming Connect requests), `connect_out` (outgoing Connect requests), and `self` (non-Connect requests)
* `--filter-ip-address`, supports a direct match with any ip address
* `--filter-http-method`, supports `GET`, `POST`, and `DELETE`
* `--filter-request-path`, supports a direct match to any Stripe path (e.g., `/v1/charges`)
* `--filter-request-status`, supports `succeeded` and `failed`
* `--filter-source`, supports `api` and `dashboard`
* `--filter-status-code`, supports any status code that is a `200`, `400`, or `500` (e.g., `404`)
* `--filter-status-code-type`, supports `2XX`, `4XX`, and `5XX`

Multiple filters can be used together, where a log must match all filters to be shown:

```sh
$ stripe logs tail --filter-http-method POST --filter-status-code-type 4XX
```

Multiple values for a single filter can also be specified as a comma-separated list, where a log only needs to match one of the values:

```sh
$ stripe logs tail --filter-http-method GET,POST
```

### `samples`

With [Stripe Samples](https://stripe.dev/samples), you can experience fully-functional sample Stripe integrations covering different integration styles, languages, and frameworks. The CLI supports downloading and configuring specific samples locally, letting you quickly get up-and-running with a sample.

To see a list of samples supported by your version of the CLI, run:

```sh
$ stripe samples list
```

To create a new sample locally, select one of the samples from the list and run:

```sh
$ stripe samples create <name>
```

The CLI will configure the sample with the API key used after logging in as well the webhook signing secret from running the `listen` command.

### `status`

You can load Stripe status from the CLI instead of going to status.stripe.com. The CLI status loads from the status site, which is the canonical source of truth.

To load status, run:
```
$ stripe status
✔ All services are online.
As of: July 23, 2019 @ 07:52PM +00:00
```

The status command supports several different flags:
1. `--verbose` lists out individual Stripe system status using.
2. `--format json` has the CLI render the status as a JSON blob for easier grepping and for using with tools like `jq`.
3. `--poll` will continuously check the status site for changes
4. `--poll-rate` let's you specify how often to check the status site. The default is once every 60 seconds and this can be modified down to once every 5 seconds.
5. `--hide-spinner` will hide the spinner that's shown when polling.

### `config`

If you need, you can manually set configuration values for the CLI using the `config` command. The config command supports:

* Setting values
* Unsetting values
* Listing config values
* Opening the editor to the config file

All operations support the `--project-name` global flag to manipulate specific projects.

To set values, run `stripe config` with the key name and the value.

```sh
$ stripe config <name> <value>
```

To unset a value, pass the `--unset` flag with the name:

```sh
$ stripe config --unset <name>
```

To list all config values, run with `--list`:

```sh
$ stripe config --list
```

To open your editor at the config file, using `--edit` or `-e`:

```sh
$ stripe config -e
```

### `open`

The `open` command is a shortcut available for users to quickly open up different parts of the Stripe docs website and dashboard. To run it, invoke:

```sh
$ stripe open <shortcut>
```

All of the available shortcuts are listed in the [wiki page](https://github.com/stripe/stripe-cli/wiki/open:-supported-shortcuts).

For dashboard pages, you can also add the `--livemode` flag to open the page directly in live mode.

## Developing the Stripe CLI

If you're working on developing the CLI, it's recommended that you alias the go command to run the dev version. Place this in your shell rc file (such as `.bashrc` or `.zshrc`)

### Installation

The Stripe CLI is built using Go. To download and compile the source code, run:

```sh
$ go get -u github.com/stripe/stripe-cli/...
```

After installing, `cd` into the directory and setup the dependencies:

```sh
$ cd go/src/github.com/stripe/stripe-cli
$ make setup
```

Once setup, run the test suite to make sure everything works as expected:

```sh
$ make test
```

You can invoke the local version of the CLI by running:

```sh
$ go run cmd/stripe/main.go
```

Optionally, you can add this to your shell profile to make running the local version a little easier.
```sh
alias stripe-dev='go run cmd/stripe/main.go'
```

### Linting

To run the linter, run `make lint`.

Make sure `golangci-lint` is installed: `brew install golangci/tap/golangci-lint`

### Tests

You can run tests with:

```sh
$ make test
```

### Releasing

To release a new version, checkout `master` and then run `make release`. It'll prompt you for a version and will then push a new tag.
