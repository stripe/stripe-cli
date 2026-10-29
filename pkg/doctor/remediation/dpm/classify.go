package dpm

// classify.go is this topic's own per-finding judgment: doctor's generic
// judge package has no idea what "card"/"link" or payment_method_types
// even mean. classifyIntent/intentVerdict and findingsVersionFix supply
// the real CANDIDATE/SKIP/REVIEW/CAUTION judgment per finding; findings.go
// (the sole caller of both) is where that judgment actually lands on a
// remediation.Finding, for both text and --json.

import (
	"regexp"
	"strings"
)

var quotedMethodRe = regexp.MustCompile(`["']([a-z0-9_]+)["']`)

// percentArrayRe matches Ruby's unquoted %w[...]/%i[...] word/symbol
// array literal (e.g. "%w[card ideal bancontact]", "%i(card ideal)") —
// its members have no quote characters at all, the quoting is implied
// by the %w/%i prefix instead, so quotedMethodRe alone can't see them.
var percentArrayRe = regexp.MustCompile(`^%[wi]\s*[\[({<]([a-z0-9_\s]*)[\])}>]$`)

// quotedMethods extracts the static method names from a finding's value
// expression, whether written as quoted string literals (e.g.
// "['card', 'ideal']") or as a %w/%i literal (e.g. "%w[card ideal]").
func quotedMethods(value string) []string {
	v := strings.TrimSpace(value)
	if m := percentArrayRe.FindStringSubmatch(v); m != nil {
		return strings.Fields(m[1])
	}
	var out []string
	for _, m := range quotedMethodRe.FindAllStringSubmatch(v, -1) {
		out = append(out, m[1])
	}
	return out
}

var identRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

// classifyIntent inspects a finding's value expression and buckets it.
// This is dpm's own domain logic (what a hardcoded payment_method_types
// value shape implies) — doctor's judge package has no idea "card" and
// "link" are payment method names, so this can't live there.
//
//	default-shaped — static list of only card/link: the classic pre-DPM
//	                 hardcode and the migration's actual target
//	deliberate     — static single non-card method (['oxxo']): per-method
//	                 integration, almost certainly intentional
//	static         — static multi-method list beyond card/link
//	dynamic        — computed at runtime (identifiers, conditionals)
func classifyIntent(value string) string {
	v := strings.TrimSpace(value)
	// %w/%i literals have no quote characters for the dynamic-detection
	// regexes below to anchor on — e.g. stripping "%w[card ideal]"'s
	// brackets/whitespace leaves "wcardideal", which identRe would
	// wrongly read as a bare identifier reference. Handle them before
	// that check runs at all, rather than trying to make the generic
	// stripping regex aware of a Ruby-specific prefix.
	if percentArrayRe.MatchString(v) {
		return bucketMethods(quotedMethods(v))
	}
	quoted := regexp.MustCompile(`["']([a-z0-9_]+)["']`).FindAllStringSubmatch(v, -1)
	stripped := regexp.MustCompile(`["'][a-z0-9_]*["']|stripe\.StringSlice|\[\]string|Arrays\.asList|List\.of|[\[\]{},()\s]|new|List|string`).ReplaceAllString(v, "")
	if identRe.MatchString(stripped) || strings.ContainsAny(v, "?:") {
		return "dynamic"
	}
	if len(quoted) == 0 {
		return "dynamic"
	}
	return bucketMethods(quotedMethods(v))
}

// bucketMethods classifies a literal (already-extracted) list of method
// names into classifyIntent's static buckets.
func bucketMethods(methods []string) string {
	onlyCardLink := true
	for _, m := range methods {
		if m != "card" && m != "link" {
			onlyCardLink = false
		}
	}
	if onlyCardLink {
		return "default-shaped"
	}
	if len(methods) == 1 {
		return "deliberate"
	}
	return "static"
}

// intentVerdict turns a classifyIntent bucket into a CANDIDATE/SKIP/REVIEW
// judgment — the real per-finding verdict doctor's generic judge package
// can't produce on its own.
func intentVerdict(intent string) string {
	switch intent {
	case "default-shaped":
		return "CANDIDATE: hardcoded to card/link only — removing this lets the Dashboard manage payment methods instead (confirm the frontend uses Payment Element)"
	case "deliberate":
		return "SKIP: restricted to a single payment method, which looks intentional — consider excluded_payment_method_types instead"
	case "dynamic":
		return "REVIEW: this value is computed at runtime — understand the routing logic before changing it"
	default:
		return "REVIEW: hardcoded list of multiple payment methods — if this restriction is intentional, use automatic_payment_methods[enabled]=true with excluded_payment_method_types naming whatever should stay off instead of a fixed allow-list; if not, it's a CANDIDATE for removal"
	}
}

// versionFix is the uniform, account-wide fix this account's recent
// API-version evidence alone dictates: the account's API version
// determines whether removing payment_method_types alone is enough, or
// whether automatic_payment_methods[enabled]=true must be added too —
// the same way for every finding, regardless of that finding's own code
// shape. A zero-value versionFix (empty Class) means the API version
// doesn't change the fix: each finding falls through to its own
// classifyIntent/intentVerdict judgment instead (see hardcodedListsFinding
// in findings.go, the sole caller). Reason and Instruction are kept
// separate (rather than one sentence) so hardcodedListsFinding can show
// them as two paragraphs.
//
// A pre-cutoff version is CAUTION, not BLOCKED: the fix isn't "do
// nothing" or "wait for traffic to move versions" — it's "delete
// payment_method_types AND add automatic_payment_methods[enabled]=true",
// a known, actionable change, not a dead end. Only NoEvents (no version
// evidence at all) stays REVIEW, since there's no fix to name yet.
type versionFix struct {
	Class       string
	Reason      string
	Instruction string
}

func findingsVersionFix(vf versionFacts) versionFix {
	switch {
	case vf.NoEvents:
		return versionFix{
			Class:       "REVIEW",
			Reason:      "Couldn't find any recent API traffic for this account, so its API version is unknown.",
			Instruction: "Confirm it's at/after " + dpmCutoff + " before removing payment_method_types — on an older version, removing it by itself silently drops payment methods.",
		}
	case vf.VersionsMixed:
		return versionFix{
			Class:       "CAUTION",
			Reason:      "This account's recent traffic mixes API versions from before and after " + dpmCutoff + ".",
			Instruction: "Plain removal is only safe for the newer requests; on the older ones, add automatic_payment_methods[enabled]=true instead, which keeps them working on every version.",
		}
	case !vf.VersionsOK:
		return versionFix{
			Class:       "CAUTION",
			Reason:      "Recent API traffic for this account uses versions before " + dpmCutoff + ". On those versions, you must explicitly enable automatic payment methods. Removing payment_method_types alone is not enough.",
			Instruction: "In each flow, confirm the API version, then replace payment_method_types with automatic_payment_methods[enabled]=true.",
		}
	default:
		return versionFix{}
	}
}
