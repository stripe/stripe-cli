package scan

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
)

// plausible decides whether an unresolved key — in key position, on the
// rule's nesting path, but not traced to a matching request — is still
// reported, as a possible finding, rather than dropped. Resolution is
// heuristic dataflow analysis, so failing to trace a bag isn't evidence
// that it's never sent; only two things are treated as conclusive:
//
//   - the file never mentions the match's resource at all, and
//   - the bag's own enclosing call or literal settles it: a Stripe SDK
//     call (see stripeCall) other than this match's operation —
//     Stripe::PaymentIntent.modify(...), Stripe::Subscription.create(...)
//     — or a typed literal whose declared type isn't one of this match's
//     params types (new UnrelatedOptions { ... }). An app's own call
//     (PaymentIntentAudit.create(...), a helper) doesn't settle it: it
//     may forward the bag to Stripe.
//
// Everything else — shadowing, overwrites, deletions, helpers, aliases
// the resolver doesn't follow — stays possible.
func plausible(key *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens, methods, allTokens []string) bool {
	if !mentionsAny(src, tokens) {
		return false
	}
	anchor := climbAnchor(key, spec, lang, src)
	if anchor == nil {
		return true
	}
	if spec.anchorStyle == anchorCall {
		callee, _, _ := splitCall(anchor, lang, src)
		knownStripeCall := stripeCall(callee) || calleeNamesResource(spec, callee, allTokens)
		return !knownStripeCall || operationMatches(spec, callee, tokens, methods)
	}
	if t, ok := statedType(anchor, spec, lang, src); ok {
		return containsAny(t, tokens)
	}
	return true
}

// statedType returns the type a typed anchor states for itself — the
// constructed type of a literal or builder-chain root, or for C#'s
// target-typed new() the type of the declaration it initializes — and
// whether any was stated at all (a chain rooted at a plain variable, b in
// b.addPaymentMethodType(...), states none).
func statedType(anchor *ts.Node, spec langSpec, lang *ts.Language, src []byte) (string, bool) {
	root := chainRoot(anchor, spec, lang)
	if t := fieldChild(root, lang, "type"); t != nil {
		return nodeText(t, src), true
	}
	if !sameSpan(root, anchor) {
		return "", false
	}
	for cur := parentOf(anchor); cur != nil; cur = parentOf(cur) {
		if containsStr(spec.funcKinds, cur.Type(lang)) {
			break
		}
		if containsStr(spec.declKinds, cur.Type(lang)) {
			if t, _, ok := declaredType(cur, lang, src); ok {
				return t, true
			}
			break
		}
	}
	return "", false
}

// stripeCall reports whether callee is recognizably a Stripe SDK call:
// rooted at the SDK's own namespace or client (Stripe::X, stripe.X,
// $stripe->x, \Stripe\X), or a StripeClient chain through a .v1/.v2
// namespace (client.v1.payment_intents).
func stripeCall(callee string) bool {
	parts := calleeComponents(callee)
	if len(parts) < 2 {
		return false
	}
	if strings.EqualFold(strings.TrimPrefix(parts[0], "$"), "stripe") {
		return true
	}
	for _, p := range parts[1 : len(parts)-1] {
		if p == "v1" || p == "v2" {
			return true
		}
	}
	return false
}

// mentionsAny reports whether src contains any token as a whole word.
func mentionsAny(src []byte, tokens []string) bool {
	s := string(src)
	for _, t := range tokens {
		if hasWord(s, t) {
			return true
		}
	}
	return false
}

// unresolvedAnchor is the context shown for a possible finding: its
// enclosing call or literal, else the assignment binding its bag.
func unresolvedAnchor(key *ts.Node, spec langSpec, lang *ts.Language, src []byte) string {
	if a := climbAnchor(key, spec, lang, src); a != nil {
		return firstLine(nodeText(a, src))
	}
	if _, assign := boundVariable(key, spec, lang, src); assign != nil {
		return firstLine(nodeText(assign, src))
	}
	return ""
}

func newFinding(key *ts.Node, path, param string, rule catalog.Rule, spec langSpec, lang *ts.Language, src []byte) catalog.Finding {
	p := key.StartPoint()
	return catalog.Finding{
		Value:    extractValue(key, spec, lang, src),
		File:     path,
		Line:     int(p.Row) + 1,
		Col:      int(p.Column) + 1,
		RuleID:   rule.ID,
		Severity: rule.Severity,
		Param:    param,
		Message:  rule.Message,
		Docs:     rule.Docs,
	}
}
