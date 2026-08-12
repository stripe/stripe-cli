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

### Tests

You can run tests with:

```sh
$ make test
```

### Error categories

Errors reported by the CLI include a category. Add an explicit category at the boundary that knows the error's user-facing meaning, rather than at every layer through which the error passes.

Use `errorcategory.New` for a categorized leaf error:

```go
return errorcategory.New(errorcategory.UserInput, "a profile name is required")
```

Use `errorcategory.Errorf` when a boundary adds context and categorizes a wrapped error:

```go
return errorcategory.Errorf(
    errorcategory.Filesystem,
    "reading configuration %q: %w",
    path,
    err,
)
```

Use `errorcategory.With` to categorize an existing error without changing its message. Plain `fmt.Errorf("loading configuration: %w", err)` is appropriate when an intermediate layer only adds context; explicit categories and recognizable error types remain discoverable through the unwrap chain. Plain `errors.New` is also appropriate for private implementation details and sentinel errors whose category is not intrinsic.

Known error types, such as `os.PathError`, are categorized automatically and do not need an explicit annotation:

```go
return &os.PathError{Op: "open", Path: path, Err: err}
```

Do not recategorize errors merely because they pass through another layer. An outer category takes precedence, so use an override only when it is intentional:

```go
return errorcategory.With(err, errorcategory.Auth)
```

### Releasing

To release a new version, checkout `master` and then run `make release`. It'll prompt you for a version and will then push a new tag.
