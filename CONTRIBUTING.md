## Installation

The [Stripe CLI](https://docs.stripe.com/cli) is built using Go. Installation instructions vary slightly based on which version of `go` you have installed locally (checked using `go version`).

For `1.18.x` or greater, run the following:

1. `git clone` this repo
2. `cd stripe-cli`
3. `go get ./...`

If you're using `v1.16` (the minimum supported) or `v1.17`, run:

```sh
$ go get -v -u github.com/stripe/stripe-cli/...
$ cd go/src/github.com/stripe/stripe-cli
```

---

No matter how you installed, you can now setup the dependencies:

```sh
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

Optionally, you can add this to your shell profile to make running the local version a little easier. Note that this command will only work when from the `stripe-cli` directory. An absolute path to the CLI folder won't work either.
```sh
alias stripe-dev='go run cmd/stripe/main.go'
```

### Linting

To run the linter, run `make lint`.

Make sure `golangci-lint` is installed: `brew install golangci/tap/golangci-lint`

New errors must include a semantic category. Use `errorcategory.New` for a
fixed message and `errorcategory.Errorf` for a formatted message:

```go
return errorcategory.New(errorcategory.UserInput, "account ID is required")
return errorcategory.Errorf(errorcategory.Auth, "profile %q has no API key", profile)
```

Adding context to an existing error is not a new error origin. Continue to use
`fmt.Errorf` with `%w` so the existing error and its category remain available:

```go
return fmt.Errorf("loading configuration: %w", err)
```

The categorized-error lint check ignores tests and generated files. In the rare
case where it cannot model a valid error origin, add a nearby
`//nolint:errorcategory` comment with a reason why no semantic category can be
assigned.

### Tests

You can run tests with:

```sh
$ make test
```

### Releasing

To release a new version, checkout `master` and then run `make release`. It'll prompt you for a version and will then push a new tag.
