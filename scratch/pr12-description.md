## Summary

Adds support for triggering application fee events and introduces a `--param` flag with pre-flight validation for triggers that require user-provided configuration.

## Background

Some Stripe events require configuration that can't be pre-filled in fixtures. For example, application fee events require a Connect account ID with the `transfers` capability enabled. Previously, these fixtures would:
- Fail at API call time with unclear errors
- Require users to manually edit fixture JSON
- Have no validation until the API request was made

This PR addresses this by adding a `--param` flag that validates required parameters before making API calls, providing early feedback and clear error messages.

## What's Included

### New `--param` Flag

Adds pre-flight validation for trigger parameters:

```bash
# Provide required parameters validated before API calls
stripe trigger application_fee.created \
  --param charge:transfer_data.destination=acct_123

# Clear error if parameter is missing
stripe trigger application_fee.created
# Error: ✘ Missing required parameters
#
#   charge:transfer_data.destination - Connect account ID with transfers capability enabled
#   Example:
#
#      stripe trigger application_fee.created \
#         --param charge:transfer_data.destination=acct_1ABC234DEF567GHI
```

**Features:**
- Uses same syntax as `--override`: `fixtureName:path.to.field=value`
- Validates against `required_params` metadata in fixture files
- Explicit errors for malformed syntax (missing `=`, empty values)
- Parameters take precedence over `--override` values
- Help text shows which events require parameters

### New Application Fee Triggers

Adds 3 new Connect-related event triggers:
- `application_fee.created` - When an application fee is created on a charge
- `application_fee.refunded` - When an application fee is fully refunded
- `application_fee.refund.updated` - When an application fee refund is updated

All three require a Connect account ID via the new `--param` flag.

## Implementation Details

- **Fixture metadata**: Added `required_params` array to `_meta` block in fixture JSON
  - Field renamed from `example` to `placeholder` for semantic clarity
- **Validation logic**: New `ValidateRequiredParams()` function with comprehensive error messages
  - Error messages include the actual event name in usage examples
  - Example error output:
    ```
    ✘ Missing required parameters

      charge:transfer_data.destination - Connect account ID
      Example:

         stripe trigger application_fee.created \
            --param charge:transfer_data.destination=acct_123
    ```
- **Test coverage**: 15 tests (11 unit tests for validation logic, 4 integration tests with fixture execution)
- **Help text**: Shows parameter requirements inline with event names. Vertical alignment is calculated from events that require params only (not all events), keeping the `--param` column tight at 82 chars instead of 93.

## Output Changes

### `stripe trigger --help`: Event list

Three new application fee events appear in the event list. Events that require parameters show the `--param` syntax inline, vertically aligned. Alignment is calculated only from events with params (30 chars), keeping lines at 82 chars instead of 93:

```diff
 Supported events:
   account.application.deauthorized
   account.updated
+  application_fee.created         --param charge:transfer_data.destination=<value>
+  application_fee.refund.updated  --param charge:transfer_data.destination=<value>
+  application_fee.refunded        --param charge:transfer_data.destination=<value>
   balance.available
   billing_portal.configuration.created
   billing_portal.configuration.updated
   billing_portal.session.created
   cash_balance.funds_available
```

### `stripe trigger --help`: Examples section

The examples section now includes descriptive comments, a parameterized usage example, and blank line separation between examples:

```diff
 Examples:
-  stripe trigger payment_intent.created
+  # Trigger a basic event
+  stripe trigger payment_intent.created
+
+  # Trigger an event that requires parameters
+  stripe trigger application_fee.created --param charge:transfer_data.destination=acct_123

 Flags:
       --add stringArray         Add params to the trigger
       --api-version string      Specify API version for trigger
       --edit                    Edit the trigger directly in your default IDE
   -h, --help                    help for trigger
```

### `stripe trigger --help`: New `--param` flag

A new `--param` flag for pre-flight parameter validation appears between `--override` and `--raw`:

```diff
       --override stringArray    Override params in the trigger
+      --param stringArray       Set required parameters (validated before
+                                execution)
       --raw string              Raw fixture in string format to replace
                                 all default fixtures
       --remove stringArray      Remove params from the trigger
```

## Spicy Decision: Params as a Layer on Top of Overrides

This PR deliberately does **not** introduce a formal concept of "params" in the fixture system itself. Instead, `--param` is a validation layer on top of the existing `--override` mechanism — params use the exact same syntax and are merged into overrides before execution.

This means fixture files remain unchanged at the execution level. The `_meta.required_params` block is only read by the trigger command for validation and help text. The fixture runner doesn't know the difference between a `--param` and an `--override`.

**The question for reviewers:** Is this the right approach, or should we formalize parameterization as a first-class concept in the fixture system?

Arguments for the current approach:
- It works today with zero changes to the fixture runner
- Users get pre-flight validation and actionable errors right now
- The `--param` / `--override` distinction is a UX concern, not a data model concern

Arguments for formalizing params in fixtures:
- Other consumers of fixtures (not just `stripe trigger`) could benefit from the same validation
- The validation logic currently lives in `triggers.go` — if fixtures supported params natively, this would move to the fixture layer where it arguably belongs

**What can be deferred vs. what needs to be decided now?**

The refactoring question — whether validation lives in `triggers.go` or moves down into the fixture runner — is entirely internal. If we later want fixtures to support parameterization as a general concept, that refactoring can be done without breaking users who have already started using `--param`. The flag, its syntax, and its behavior all stay the same; only the internal plumbing moves.

However, there are decisions in this PR that **are** hard to change later because they form the user-facing contract:

1. **Is `--param` the right name for this concept?** Once users start scripting against it, renaming is a breaking change.

2. **Should params use fixture JSON paths (`charge:transfer_data.destination`) or semantic names (`connected_account_id`)?** I considered semantic names earlier — they're shorter and describe the *purpose* of an input (similar to function argument names). But the downside is you lose the relationship to where that value is placed in the API call, and that relationship is a fairly central part of the fixture mental model. This PR uses fixture paths, which means users can look at the fixture JSON and see exactly where their value ends up — same as `--override`.

## Testing

```bash
# Run fixture tests
go test ./pkg/fixtures/... -v

# Test the trigger command
stripe trigger application_fee.created --param charge:transfer_data.destination=acct_test123
```

All tests pass. The `--param` flag provides robust validation before API calls, catching configuration errors early with actionable error messages.
