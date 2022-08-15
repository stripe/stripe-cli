# `stripe-cli` Architecture

The Stripe CLI acts as an ergonomic wrapper around most of the [Stripe API](https://stripe.com/docs/api). It's written in [go](https://go.dev/) and is compiled into a binary for distribution.

## Commands

Its functionality is organized into individual command files, found in `pkg/`. Most files correspond to a command that can be run via the CLI; for example, the `stripe login` is powered by `pkg/login.go`. `pkg/` also includes some test files (e.g. `pkg/open_test.go` has the tests for `pkg/open.go`, per language convention). Some commands have _further_ subcommands, which are defined in their folder under `pkg`. For instance, `pkg/plugin.go` only serves to collect the sub-commands in `pkg/plugins/*.go`.

The CLI uses the [Cobra library](https://github.com/spf13/cobra) for command structure and [Viper](https://github.com/spf13/viper) for flag parsing. Commands are registered in `pkg/cmd/root.go` in the `init` function (look for the big `rootCmd.AddCommand(...)` block). If a command isn't registered there, it'll give an error when invoked from the CLI (e.g. `stripe missing_command` gives an "unknown command" error).

`root.go`'s `Execute` function handles actually invoking `Cobra`, via the `rootCmd.ExecuteContext` function. CLI input is parsed and routed to the appropriate command, which runs and/or errors accordingly. If there's an error, `Execute` may print additional help text before exiting.

### Auto-Generated Resources

In addition to the handwritten commands, the CLI also has many auto-generated resources that correspond to base [API resources](https://stripe.com/docs/api/charges). These commands are registered in `pkg/cmd/resources_cmds.go`, calling the generic `NewNamespaceCmd` function for each resource. These auto-generated commands hold no functionality on their own and mostly rely on [operation commands](#operation-commands) to do anything (see below).

The big list of commands found in `resources_cmds.go` is _iself_ is an auto-generated file. It's built by running `pkg/gen/gen_resource_cmds.go`, which gets a big list of resources from OpenAPI via `api/openapi-spec/spec3.sdk.json` and generates the big command file via the `pkg/gen/resources_cmds.go.tpl` template.

### Operation Commands

Under each generated resource are a set of operation commands, corresponding to CRUD operations for that resource (`retrieve`, `create`, etc). They are also registered in `resources_cmds.go`, but use the `NewOperationCmd` method to build a generic HTTP call and response. These sub-commands are only used for [resources](#auto-generated-resources).

## API Calls

HTTP requests to the Stripe API are routed through the `requests` package, starting in `pkg/requests/base.go`. Under the hood, they use an instance of `pkg/stripe/client.go`'s `Client` and it's `PerformRequest` method (which wraps native go HTTP code).
