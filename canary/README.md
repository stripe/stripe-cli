# Canary Tests

End-to-end tests that invoke the compiled Stripe CLI binary to verify real-world behavior.

## Purpose

Unlike unit tests that use Cobra's `ExecuteC()` for in-process testing, canary tests physically invoke the compiled binary. This catches issues that would only appear in the actual built artifact:

- Build/packaging problems
- Binary startup issues
- Signal handling
- Environment variable handling
- Exit code behavior

## Running Tests

### Prerequisites

1. Build the CLI binary:
   ```bash
   make build
   ```

2. Set the binary path:
   ```bash
   export STRIPE_CLI_BINARY=$(pwd)/stripe
   ```

### Offline Tests (No API Key Required)

Run tests that don't require API access:

```bash
make canary-offline
```

Or directly:

```bash
STRIPE_CLI_BINARY=$(pwd)/stripe go test -v ./canary/... -run "TestOffline"
```

### All Tests (Requires API Key)

Run all tests including API tests:

```bash
export STRIPE_API_KEY=sk_test_...
make canary
```

Or directly:

```bash
STRIPE_CLI_BINARY=$(pwd)/stripe STRIPE_API_KEY=sk_test_... go test -v -timeout 15m ./canary/...
```

## Test Categories

### Offline Tests (`TestOffline*`)

Tests that verify basic CLI functionality without network access:

| Test | Command | Validates |
|------|---------|-----------|
| `TestOfflineVersion` | `stripe version` | Binary runs, outputs version |
| `TestOfflineHelp` | `stripe --help` | Core commands listed |
| `TestOfflineCompletionBash` | `stripe completion --shell bash` | Bash completion generated |
| `TestOfflineCompletionZsh` | `stripe completion --shell zsh` | Zsh completion generated |
| `TestOfflineConfig` | `stripe config --list` | Config system works |
| `TestOfflineUnknownCommand` | `stripe nonexistent` | Error handling |
| `TestOfflineStatus` | `stripe status` | Status command runs |

### API Tests (`TestAPI*`)

Tests that require a valid Stripe test API key:

| Test | Command | Validates |
|------|---------|-----------|
| `TestAPIGetBalance` | `stripe get /v1/balance` | API auth works |
| `TestAPICreateDeleteCustomer` | `stripe post/delete /v1/customers` | Write operations |
| `TestAPITrigger` | `stripe trigger customer.created` | Fixtures work |
| `TestAPICustomersList` | `stripe customers list` | Generated commands |
| `TestAPIProductsCreate` | `stripe products create` | Resource commands |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `STRIPE_CLI_BINARY` | Yes | Path to the compiled stripe binary |
| `STRIPE_API_KEY` | For API tests | Stripe test API key (`sk_test_...`) |
| `STRIPE_CLI_TELEMETRY_OPTOUT` | No | Set to `1` to disable telemetry (auto-set by runner) |

## CI Integration

Canary tests run automatically in GitHub Actions on every push and PR. The workflow:

1. Builds the binary for the target OS
2. Runs offline tests (always)
3. Runs API tests (if `STRIPE_TEST_API_KEY` secret is configured)

See `.github/workflows/canary-test.yml` for details.

## Adding New Tests

1. Add test functions to `canary/runner_test.go`
2. Use `TestOffline` prefix for tests without API requirements
3. Use `TestAPI` prefix and call `requireAPIKey(t)` for API tests
4. Use isolated config directories via `testutil.CreateTempConfigDir()`

Example:

```go
func TestOfflineNewFeature(t *testing.T) {
    runner := getRunner(t)

    result, err := runner.Run("new-command", "--flag")
    if err != nil {
        t.Fatalf("Failed to run command: %v", err)
    }

    if result.ExitCode != 0 {
        t.Errorf("Expected exit code 0, got %d", result.ExitCode)
    }
}

func TestAPINewFeature(t *testing.T) {
    runner := getRunner(t)
    requireAPIKey(t)

    runner = runner.WithEnv(map[string]string{
        "STRIPE_API_KEY": testutil.GetAPIKey(),
    })

    result, err := runner.Run("api-command")
    // ...
}
```

## Troubleshooting

### "STRIPE_CLI_BINARY environment variable not set"

Build the binary and set the path:

```bash
make build
export STRIPE_CLI_BINARY=$(pwd)/stripe
```

### API tests skipped

Set your test API key:

```bash
export STRIPE_API_KEY=sk_test_your_key_here
```

### Tests timing out

Increase the timeout:

```bash
go test -v -timeout 30m ./canary/...
```

Or for specific tests, the runner supports custom timeouts:

```go
runner = runner.WithTimeout(2 * time.Minute)
```
