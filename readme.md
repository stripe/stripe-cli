# Stripe CLI

[![Build Status](https://travis-ci.com/stripe/stripe-cli.svg?token=eQWDVpt1sJR63TxbC1KA&branch=master)](https://travis-ci.com/stripe/stripe-cli)

### For canonical instructions on installing and use the CLI, see the [paper doc](https://paper.dropbox.com/doc/CLI-v0-docs--AbvmdSi8hEinMB3ITVeaARmNAg-5Mob9a5xpDCI16IYYu1i2)

This readme is intended for developers of the CLI, not the end-users.

## Installing

To install, run `$ go get -u github.com/stripe/stripe-cli`.

## Deploying

To deploy a new version of the CLI, run `make release` from master. This will pull the most the most recent changes, prompt for a new version, and push a new tag. All releases are hosted on github.com in the [releases](https://github.com/stripe/stripe-cli/releases) section.

## Development

We have some unofficial style guidelines we try to adhere to as part of development:

* Try to break things up into packages for easier testing
* Keep variables, functions, items, etc alphabetically sorted as much as possible
* Avoid erroring out in most cases and instead return an error. Generally, rely on the higher-level code to handle error catching

## Tests

Run tests with `$ go test ./...`
