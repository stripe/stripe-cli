# `stripe-cli` Architecture

The Stripe CLI acts as an ergonomic wrapper around most of the [Stripe API](https://stripe.com/docs/api). It's written in [go](https://go.dev/) and is compiled into a binary for distribution.

## Commands

Its functionality is organized into individual command files, found in `pkg/`. Most files correspond to a command that can be run via the CLI; for example, the `stripe get` is powered by `pkg/get.go`. `pkg/` also includes some test files (`pkg/resources_test.go` has the tests for `pkg/resources.go`, per go convention).

The CLI uses the [Cobra library](https://github.com/spf13/cobra) for flag parsing, command `struct`s, and generating help text. Commands are registered in `pkg/cmd/root.go` in the `init` function (look for the big `rootCmd.AddCommand(...)` block). If a command isn't registered there, it'll give an error when invoked from the CLI (e.g. `stripe blahblah` gives an "unknown command" error).

`root.go`'s `Execute` function handles actually invoking `Cobra`, via the `rootCmd.ExecuteContext` function. CLI input is parsed and routed to the appropriate command, which runs and/or errors accordingly. If there's an error, `Execute` may print additional help text before exiting.

### Auto-Generated Resources

In addition to the handwritten commands, the CLI also has many auto-generated resources that correspond to base [API resources](https://stripe.com/docs/api/charges). These are generated in `pkg/cmd/resource/resource.go`, mostly via the `NewResourceCmd` function. Those calls come from `pkg/cmd/resources_cmds.go`, which _iself_ is an auto-generated file. It gets a big list of resources from OpenAPI via `spec3.sdk.json` and runs them through `pkg/gen/gen_resource_cmds.go` to write the `.go` files.

### Operation Commands

Under each resource are a set of basic operations, corresponding to CRUD operations for that resource (`retrieve`, `create`, etc). They are built in `pkg/cmd/resource/operation.go` via calls in `resources_cmds.go` to the `NewOperationCmd` method.

## API Calls

HTTP requests to the Stripe API are routed through the `requests` package, starting in `pkg/requests/base.go`. Under the hood, they use an instance of `pkg/stripe/client.go`'s `Client` and it's `PerformRequest` method (which wraps native go HTTP code).
