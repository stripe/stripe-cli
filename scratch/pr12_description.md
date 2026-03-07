## Summary

Adds trigger fixtures for Connect application fee events:
- `application_fee.created`
- `application_fee.refunded`
- `application_fee.refund.updated`

These events require Connect destination charges to trigger.

## Context

Part of expanding Stripe CLI trigger coverage by 67 events (+49%). This PR adds 3 Connect application fee events and can be reviewed independently.

## Changes

- **1 commit**, ~152 lines added
- **3 new events** mapped in `pkg/fixtures/triggers.go`
- **3 new fixture files** in `pkg/fixtures/triggers/`

## Test plan

- [x] All fixture JSON files validated
- [x] Go build successful
- [x] All tests passing
- [ ] Manual: `stripe trigger application_fee.created` ⚠️ (requires Connect account capabilities - see note below)
- [ ] Manual: `stripe trigger application_fee.refunded` ⚠️ (requires Connect account capabilities - see note below)
- [ ] Manual: `stripe trigger application_fee.refund.updated` ⚠️ (requires Connect account capabilities - see note below)

**Note**: Application fee triggers require a Connect account with active `transfers` capability. In test mode, newly created Express accounts require manual activation of capabilities via the Dashboard or fulfillment of verification requirements. The fixtures create the account structure correctly, but the charges will fail with `insufficient_capabilities_for_transfer` until capabilities are activated.
