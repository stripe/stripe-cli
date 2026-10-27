package login

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// ResolveParentAccounts looks up the parent (live) account for each sandbox
// among accounts via the Stripe CLI context status API, so callers can group
// sandboxes under their parent (see PrintGroupedAuthorizedAccounts and
// classifyAccountsByParent). The bool result is false only when the lookup
// itself failed (as opposed to succeeding with nothing to report); callers
// should fall back to an ungrouped list in that case rather than treating
// every sandbox as having no known parent.
//
// The request must be authenticated with the active account/livemode as the
// OAK context (the same credentials GetUserInfo uses) — the API rejects a
// bare UAT with no Stripe-Context/Stripe-Livemode with a 403, since it can't
// otherwise resolve which permission grant applies. Without an active
// context there's no context to authenticate with, so the lookup is skipped
// (this isn't a failure: there's simply nothing to group beyond what's
// already derivable locally).
func ResolveParentAccounts(ctx context.Context, apiBaseURL string, profile *config.Profile, uat string, ac *config.ActiveContext, accounts []config.AuthorizedAccount) (map[string]string, bool) {
	if ac == nil {
		return nil, true
	}

	var queries []requests.ContextStatusQuery
	for _, a := range accounts {
		if accountHasLiveMode(a.Modes) {
			continue
		}
		queries = append(queries, requests.ContextStatusQuery{AccountID: a.ID, Livemode: false})
	}
	if len(queries) == 0 {
		return nil, true
	}

	creds := stripe.NewOAKCredentials(uat, ac.AccountID, ac.Livemode)
	statuses, err := requests.GetContextStatuses(ctx, apiBaseURL, profile, creds, queries)
	if err != nil {
		return nil, false
	}

	parentAccountIDs := make(map[string]string, len(statuses))
	for _, s := range statuses {
		if s.ParentAccountID != "" {
			parentAccountIDs[s.AccountID] = s.ParentAccountID
		}
	}
	return parentAccountIDs, true
}

// PrintAuthorizedAccounts prints accounts grouped by parent when parent info
// can be resolved via the context status API, falling back to the plain,
// ungrouped list when the lookup itself fails.
func PrintAuthorizedAccounts(ctx context.Context, apiBaseURL string, profile *config.Profile, uat string, accounts []config.AuthorizedAccount) {
	ac, _ := profile.GetActiveContext()
	if parentAccountIDs, ok := ResolveParentAccounts(ctx, apiBaseURL, profile, uat, ac, accounts); ok {
		PrintGroupedAuthorizedAccounts(profile, accounts, parentAccountIDs)
	} else {
		// The context status lookup itself failed (as opposed to succeeding
		// with nothing to report), so we have no trustworthy grouping
		// information. Fall back to the plain list rather than dumping
		// every sandbox under "Other sandboxes", which would look like we
		// checked and found no parent rather than couldn't check at all.
		PrintAuthorizedContextsList(profile, accounts)
	}
}
