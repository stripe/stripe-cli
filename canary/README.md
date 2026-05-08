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
| `TestOfflineListenHelp` | `stripe listen --help` | Listen command available |
| `TestOfflineLogsTailHelp` | `stripe logs tail --help` | Logs tail command available |
| `TestOfflineLoginHelp` | `stripe login --help` | Login command available |

### API Tests (`TestAPI*`)

Tests that require a valid Stripe test API key:

| Test | Command | Validates |
|------|---------|-----------|
| `TestAPIGetBalance` | `stripe get /v1/balance` | API auth works |
| `TestAPICreateDeleteCustomer` | `stripe post/delete /v1/customers` | Write operations |
| `TestAPITrigger` | `stripe trigger customer.created` | Fixtures work |
| `TestAPICustomersList` | `stripe customers list` | Generated commands |
| `TestAPIProductsCreate` | `stripe products create` | Resource commands |
| `TestAPIListenPrintSecret` | `stripe listen --print-secret` | Webhook listener connects |
| `TestAPIListenWithEvents` | `stripe listen --events ... --print-secret` | Event filtering works |
| `TestAPIListenForwardTo` | `stripe listen --forward-to` + trigger | Webhook forwarding end-to-end |
| `TestAPIListenOutputFormat` | `stripe listen --format JSON` + trigger | Listen shows events correctly |
| `TestAPILogsTailStartup` | `stripe logs tail` | Log streaming connects |
| `TestAPILogsTailWithFilters` | `stripe logs tail --filter-http-method POST` | Log filtering works |
| `TestAPILogsTailCapture` | `stripe logs tail --format JSON` + API call | Logs tail captures requests |
| `TestAPIConfigSetAndUseAPIKey` | `stripe config --set` + API call | Config-based auth |
| `TestAPIConfigMultipleProfiles` | `--project-name` flag | Multi-profile support |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `STRIPE_CLI_BINARY` | Yes | Path to the compiled stripe binary |
| `STRIPE_API_KEY` | For API tests | Stripe test API key (`sk_test_...`) |
| `STRIPE_CLI_TELEMETRY_OPTOUT` | No | Set to `1` to disable telemetry (auto-set by runner) |

## CI Integration

Canary tests run automatically in GitHub Actions:

- **Push to master**: Runs directly via `canary-test.yml`
- **Release (tag `v*`)**: Runs as a gate before release builds — all three OS builds (`build-mac`, `build-linux`, `build-windows`) depend on canary tests passing
- **Manual dispatch**: Available via the GitHub Actions UI for ad-hoc use

The workflow builds the binary for each target OS, runs offline tests (always), and runs API tests when secrets are available.

See `.github/workflows/canary-test.yml` for details.

### Manual Trigger for Reviewers

Reviewers can manually trigger canary tests via the GitHub Actions UI:

1. Go to **Actions** > **Canary Tests**
2. Click **Run workflow**
3. Select the branch to test
4. Check "Run API tests" to include API tests

This is useful when reviewing PRs from external contributors where you want to verify the code works correctly with real API calls.

## Security Model

These tests are designed to run safely in CI, including on PRs from external contributors.

### Output Sanitization

All test output is sanitized to prevent accidental exposure of secrets:

- API keys (`sk_test_*`, `sk_live_*`, `rk_*`)
- Webhook signing secrets (`whsec_*`)
- Access tokens and bearer tokens

The `testutil.SanitizeOutput()` function redacts these patterns before logging. All test helper functions (`fatalf`, `errorf`, `logSanitizedf`) use sanitization automatically.

### CI Secret Handling

- **Offline tests**: Always run (no secrets needed)
- **API tests**: Only run when secrets are available:
  - Push to master
  - Release builds (called via `workflow_call`)
  - Manual workflow_dispatch trigger

Canary tests do not run on PRs. Unit tests provide coverage for PR validation.

## Adding New Tests

1. Add test functions to the appropriate file:
   - `basic_test.go` - Version, help, completion tests
   - `api_test.go` - API resource tests
   - `listen_test.go` - Webhook listener tests
   - `logs_test.go` - Log streaming tests
   - `config_test.go` - Configuration tests
   - `login_test.go` - Authentication tests
2. Use `TestOffline` prefix for tests without API requirements
3. Use `TestAPI` prefix and call `requireAPIKey(t)` for API tests
4. Use isolated config directories via `testutil.CreateTempConfigDir()`
5. **Always use sanitized logging**: Use `fatalf()`, `errorf()`, and `logSanitizedf()` instead of `t.Fatalf()`, `t.Errorf()`, and `t.Logf()`

Example:

```go
func TestOfflineNewFeature(t *testing.T) {
    runner := getRunner(t)

    result, err := runner.Run("new-command", "--flag")
    if err != nil {
        fatalf(t, "Failed to run command: %v", err)
    }

    if result.ExitCode != 0 {
        errorf(t, "Expected exit code 0, got %d", result.ExitCode)
    }
}

func TestAPINewFeature(t *testing.T) {
    runner := getRunner(t)
    requireAPIKey(t)

    runner = runner.WithEnv(map[string]string{
        "STRIPE_API_KEY": testutil.GetAPIKey(),
    })

    result, err := runner.Run("api-command")
    // Use sanitized logging for output that may contain secrets
    logSanitizedf(t, "Result: %s", result.Stdout)
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
