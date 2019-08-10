# Stripe CLI

[![Build Status](https://travis-ci.com/stripe/stripe-cli.svg?token=eQWDVpt1sJR63TxbC1KA&branch=master)](https://travis-ci.com/stripe/stripe-cli)

_🏗 The Stripe CLI is currently in beta! We're working on more features to make the experience great. If you have any [feedback](https://stri.pe/cli-feedback), find [issues](https://github.com/stripe/stripe-cli/issues), or would like to be involved in more active testing, please let us know!_

The Stripe CLI is a command-line interface for Stripe that can:

1. `login` to your Stripe account and authenticate the CLI
2. `listen` for webhooks and forward them to a local server
3. Run `get`, `post`, and `delete` commands to the Stripe API
4. `trigger` a limited set of webhook events
5. Tail your testmode API request logs
6. Pull Stripe status from status.stripe.com

The main focus for this initial release is to improve the developer experience while integrating and testing webhooks. Interactions through the CLI are currently limited to test mode only.

## Table of Contents

* [Stripe CLI](#stripe-cli)
  * [Installation](#installation)
    * [Download the CLI](#download-the-cli)
      * [macOS](#macos)
      * [Linux](#linux)
      * [Windows](#windows)
  * [Commands](#commands)
    * [login](#login)
    * [listen](#listen)
    * [get, post, and delete](#get-post-and-delete)
    * [trigger](#trigger)
    * [status](#status)
  * [Developing the Stripe CLI](#developing-the-stripe-cli)
    * [Installation](#installation-1)
    * [Tests](#tests)

## Installation

### macOS

_With homebrew:_

Run `brew install stripe/stripe-cli/stripe`

_Without homebrew:_

1. Download the latest `mac-os` tar.gz file from https://github.com/stripe/stripe-cli/releases/latest

2. Unzip the file: `tar -xvf stripe_X.X.X_mac-os_x86_64.tar.gz`

3. (optional) Move the binary to somewhere you can execute it globally, like `~/usr/local/bin`

### Linux

_With a package manager:_

**Debian/Ubuntu-based distributions**:

1. Add Bintray's GPG key to the apt sources keyring: `sudo apt-key adv --keyserver hkp://pool.sks-keyservers.net --recv-keys 379CE192D401AB61`

2. Add stripe-cli's apt repository to the apt sources list: `echo "deb https://dl.bintray.com/stripe/stripe-cli-deb stable main" | sudo tee -a /etc/apt/sources.list`

3. Update the package list: `sudo apt-get update`

4. Install the CLI: `sudo apt-get install stripe`

**RedHat/CentOS-based distributions:**

1. Add stripe-cli's yum repository to the yum sources list: `wget https://bintray.com/stripe/stripe-cli-rpm/rpm -O bintray-stripe-stripe-cli-rpm.repo && sudo mv bintray-stripe-stripe-cli-rpm.repo /etc/yum.repos.d/`

2. Update the package list: `sudo yum update`

3. Install the CLI: `sudo yum install stripe`

_Without a package manager:_

1. Download the latest `linux` tar.gz file from https://github.com/stripe/stripe-cli/releases/latest

2. Unzip the file: `tar -xvf stripe_X.X.X_linux_x86_64.tar.gz`

3. Run the executable: `./stripe`

### Windows

_With scoop:_

1. Run `scoop bucket add stripe https://github.com/stripe/scoop-stripe-cli.git`

2. Run `scoop install stripe`

_Without scoop:_

1. Download the latest `windows` tar.gz file from https://github.com/stripe/stripe-cli/releases/latest

2. Unzip the `stripe_X.X.X_windows_x86_64.tar.gz` file

3. Run the unzipped `.exe` file!

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

* `--forward-to localhost:9000`
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

### `get`, `post`, and `delete`

The CLI has three commands that let you interact with the Stripe API in test mode. You can easily make `GET`, `POST`, and `DELETE` commands with the Stripe CLI.

For example, you can retrieve a specific charge:

```sh
$ stripe get /charges/ch_1EGYgUByst5pquEtjb0EkYha
```

You can also pass data in using the `-d` flag:

```
$ stripe post /charges -d amount=100 -d source=tok_visa -d currency=usd
```

These commands support many of the features on the Stripe API (e.g. selecting a version, pagination, and expansion) through command-line flags, so you won't need to provide specific headers.

| Command | Flag | Description| Example|
|---------|------|------------|--------|
| get, post, delete | `-d`, `--data` | Data to pass for the API request | `--data id=cust_123abc` |
| get, post, delete | `-e`, `--expand` | Response attributes to expand inline. Available on all API requests, see the documentation for specific objects that support expansion | `--expand customer,charges` |
| get, post, delete | `-i`, `--idempotency` | Sets the idempotency key for your request, preventing replaying the same requests within a 24 hour period. | `--idempotency foobar123456` |
| get, post, delete | `-v`, `--api-version` | Set the Stripe API version to use for your request | `--api-version 2019-03-14` |
| get, post, delete | `--stripe-account` | Set a header identifying the connected account for which the request is being made | `--stripe-account m_1234acbd` |
| get, post, delete | `-s`, `--show-headers` | Show headers on responses to GET, POST, and DELETE requests | `--show-headers` |
| delete | `-c`, `--confirm` | Automatically confirm the command being entered. WARNING: This will result in NOT being prompted for confirmation for certain commands | `--confirm` |
| get | `-l`, `--limit` | A limit on the number of objects to be returned, between 1 and 100 (default is 10) | `--limit 50` |
| get | `-a`, `--starting-after` | Retrieve the next page in the list. This is a cursor for pagination and should be an object ID | `--starting-after cust_1234abc` |
| get | `-b`, `--ending-before` | Retrieve the previous page in the list. This is a cursor for pagination and should be an object ID | `--ending-before cust_1234abc` |

You can pipe the output of these commands to other tools. For example, you could use [jq](https://stedolan.github.io/jq/) to extract information from JSON the API returns, and then use that information to trigger other API requests.

Here’s a simple example that lists `past_due` subscriptions, extracts the IDs, and cancels those subscriptions:

```sh
$ stripe get /subscriptions -d status=past_due | jq ".data[].id" | xargs -I % -p stripe delete /subscriptions/%
```

### `trigger`

The CLI will allow you to trigger a few test webhook events to conduct local testing. These test webhook events are real objects on the API and may trigger other webhook events as part of the test (e.g. triggering `payment_intent.succeeded` will also trigger `payment_intent.created`).

The webhook events we currently support are:

* `charge.captured`
* `charge.failed`
* `charge.succeeded`
* `customer.created`
* `customer.updated`
* `customer.source.created`
* `customer.source.updated`
* `customer.subscription.updated`
* `invoice.created`
* `invoice.finalized`
* `invoice.payment_succeeded`
* `invoice.updated`
* `payment_intent.created`
* `payment_intent.payment_failed`
* `payment_intent.succeeded`
* `payment_method.attached`

To trigger an event, run:

```sh
$ stripe trigger <event>
```

### `logs tail`
`logs tail` establishes a direct connection with Stripe and enables you to tail your testmode Stripe API request logs in real-time from your terminal.

By default, `logs tail` will display all of your testmode request logs. To begin log tailing, run:

```sh
$ stripe logs tail
```

A number of built-in filtering options are also supported:
* `--filter-account`, *(connect only)* supports `connect_in` (incoming connect requests), `connect_out` (outgoing connect requests), and `self` (non-connect requests)
* `--filter-ip-address`, supports a direct match with any ip address
* `--filter-http-method`, supports `GET`, `POST`, and `DELETE`
* `--filter-request-path`, supports a direct match to any Stripe path (e.g., `/v1/charges`)
* `--filter-request-status`, supports `succeeded` and `failed`
* `--filter-source`, supports `api` and `dashboard`
* `--filter-status-code`, supports any status code that is a `200`, `400`, or `500` (e.g., `404`)
* `--filter-status-code-type`, supports `200`, `400`, and `500`

Multiple filters can be used together, where a log must match all filters to be shown:

```sh
$ stripe logs tail --filter-http-method POST --filter-status-code-type 400
```

### `status`

You can load Stripe status from the CLI instead of going to status.stripe.com. The CLI status loads from the status site, which is the canonical source of truth.

To load status, run:
```
$ stripe status
✅ All services are online.
As of: July 23, 2019 @ 07:52PM +00:00
```

The status command supports several different flags:
1. `--verbose` lists out individual Stripe system status using.
2. `--format json` has the CLI render the status as a JSON blob for easier grepping and for using with tools like `jq`.
3. `--poll` will continuously check the status site for changes
4. `--poll-rate` let's you specify how often to check the status site. The default is once every 60 seconds and this can be modified down to once every 5 seconds.
5. `--hide-spinner` will hide the spinner that's shown when polling.

## Developing the Stripe CLI

### Installation

The Stripe CLI is built using Go. To download and compile the source code, run:

```sh
$ go get -u github.com/stripe/stripe-cli/...
```

### Tests

You can run tests with:

```sh
$ make test
```
