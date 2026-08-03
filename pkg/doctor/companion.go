package doctor

// The companion fork: version-gated replacement of a removed parameter,
// server-side-confirmation handling, and the code evidence that feeds it.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stripe/stripe-cli/pkg/config"
)

// plain-remove because the account's traffic is entirely at/after the cutoff.
type companionDecision struct {
	insert  bool
	reason  string
	oldest  string
	account string
	// returnURL, when set (--return-url), is inserted alongside the companion
	// at server-side-confirmation sites so redirect-based payment methods
	// keep working; without it those sites get allow_redirects:"never".
	returnURL string
}

// versionPinRe finds an API version pinned in code near a version-ish
// keyword: apiVersion: '2022-11-15', stripe.api_version = "...",
// .setApiVersion("..."), Stripe-Version headers. Named suffixes
// (.dahlia) compare by date prefix.
var versionPinRe = regexp.MustCompile(`(?i)(stripe[-_ ]?version|api[-_ ]?version)[^\n]{0,40}?(20\d\d-\d\d-\d\d)`)

// scanVersionPins walks the scan tree for pinned API versions in code —
// stronger evidence than the events census, which only reflects the ACCOUNT
// DEFAULT version (an SDK pinning an older Stripe-Version runs below it
// invisibly). Returns the oldest pin and where it was found.
func scanVersionPins(root string) (oldest, at string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "vendor", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := specs[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(src), "\n") {
			// Only lines that name Stripe count: AWS/GitHub/Azure clients and
			// prose pin their own date-shaped API versions too.
			if !strings.Contains(strings.ToLower(line), "stripe") {
				continue
			}
			for _, m := range versionPinRe.FindAllStringSubmatch(line, -1) {
				v := m[2]
				if oldest == "" || v < oldest {
					oldest, at = v, path
				}
			}
		}
		return nil
	})
	return oldest, at
}

// resolveCompanion decides the fork from account facts plus code evidence.
// Every unknowable branch (offline, no credentials, no recent events)
// resolves to INSERT: the companion is semantically a no-op at/after the
// cutoff and load-bearing below it, so inserting is the only choice that is
// correct at any version. A version pinned in CODE below the cutoff
// overrides an omit verdict — the census only sees the account default.
func resolveCompanion(rule Rule, cfg *config.Config, stripeAccount, root string, offline bool) *companionDecision {
	unknown := func(why string) *companionDecision {
		return &companionDecision{insert: true,
			reason: why + " — traffic API versions unknown; inserting " + rule.Companion.Param + " (a no-op at/after " + rule.IntroducedIn + ", required below it)"}
	}
	dec := func() *companionDecision {
		if offline {
			return unknown("--offline")
		}
		key, err := loadTestKey(cfg)
		if err != nil {
			return unknown("no credentials (" + err.Error() + ")")
		}
		facts, err := fetchAccountFacts(key, stripeAccount)
		if err != nil {
			return unknown("account lookup failed (" + err.Error() + ")")
		}
		d := decideCompanion(rule, facts)
		d.account = facts.AccountID
		return d
	}()
	if pin, at := scanVersionPins(root); pin != "" && datePrefix(pin) < rule.IntroducedIn {
		if !dec.insert {
			dec.insert = true
			dec.reason = "code pins Stripe-Version " + pin + " (" + at + ") — below " + rule.IntroducedIn + "; the events census only reflects the account default, so the pin wins and " + rule.Companion.Param + " is inserted"
		} else {
			dec.reason += "; code also pins " + pin + " (" + at + ")"
		}
	}
	return dec
}

// decideCompanion is the pure fork: it recomputes the version census against
// THIS rule's cutoff (accountFacts' own VersionsOK is bound to the dpm
// constant, which only coincidentally matches) and folds in the Dashboard-
// configuration caveat the doctor would raise on the same facts.
func decideCompanion(rule Rule, facts *accountFacts) *companionDecision {
	cutoff := rule.IntroducedIn
	total, atOrAfter := 0, 0
	for v, n := range facts.EventVersions {
		total += n
		if datePrefix(v) >= cutoff {
			atOrAfter += n
		}
	}
	configNote := ""
	if !facts.ConfiguredOK {
		configNote = "; NOTE: no active Dashboard payment-method configuration — doctor reports BLOCKED until methods are configured"
	}
	switch {
	case total == 0:
		return &companionDecision{insert: true,
			reason: "no recent events — traffic API versions unknown; inserting " + rule.Companion.Param + " (a no-op at/after " + cutoff + ", required below it)" + configNote}
	case atOrAfter == total:
		return &companionDecision{insert: false, oldest: facts.OldestVersion,
			reason: "all recent traffic runs at/after " + cutoff + " — " + rule.Companion.Param + " is default-enabled there; plain removal is behavior-preserving" + configNote}
	default:
		return &companionDecision{insert: true, oldest: facts.OldestVersion,
			reason: "recent traffic runs below " + cutoff + " (oldest " + facts.OldestVersion + ") — bare removal would silently drop methods; inserting " + rule.Companion.Param + configNote}
	}
}

// companionResourceFor maps a removal site onto one of the companion's
// eligible API resources ("" = not eligible). Eligibility needs CREATE
// evidence, not just the resource name: automatic_payment_methods is a
// create-only parameter, so update/confirm/modify sites must never gain it
// (the API rejects it there), and a Checkout Session builder that merely
// mentions PaymentIntentData must not match payment_intents.
func companionResourceFor(spec langSpec, anchor, funcText string, resources []string) string {
	for _, r := range resources {
		single := pascalSingular(r)
		switch spec.name {
		case "java", "csharp":
			// The create params class is the token; Update/Confirm classes
			// are distinct and never match.
			if strings.Contains(anchor, single+"CreateParams") || strings.Contains(anchor, single+"CreateOptions") {
				return r
			}
		case "go":
			// stripe-go shares one params struct across create/update/confirm;
			// the verb lives at the call site. Require create evidence (.New)
			// and no update/confirm evidence in the enclosing function.
			if strings.Contains(anchor, single+"Params") &&
				strings.Contains(funcText, ".New(") &&
				!strings.Contains(funcText, ".Update(") && !strings.Contains(funcText, ".Confirm(") {
				return r
			}
		default:
			// Dynamic SDKs name the verb in the call itself.
			if (strings.Contains(anchor, single) || strings.Contains(anchor, lowerCamelPlural(r))) &&
				strings.Contains(strings.ToLower(anchor), "create") &&
				!strings.Contains(strings.ToLower(anchor), "update") &&
				!strings.Contains(strings.ToLower(anchor), "confirm") &&
				!strings.Contains(strings.ToLower(anchor), "modify") {
				return r
			}
		}
	}
	return ""
}

// alreadyHasCompanion reports whether the removal site's own scope (param
// bag / builder chain / function for builder statements) already sets the
// companion — scoped per call site, so a half-migrated file still gets the
// insert on its remaining calls.
func alreadyHasCompanion(sp span, param string) bool {
	switch sp.site {
	case "statement":
		return strings.Contains(sp.scope, sp.receiver+".set"+pascal(param))
	case "chain-link":
		return strings.Contains(sp.scope, ".set"+pascal(param))
	}
	return strings.Contains(sp.scope, param) || strings.Contains(sp.scope, pascal(param))
}

// companionOpts selects the companion variant for a site. Server-side
// confirmation (confirm:true) with automatic_payment_methods and no
// return_url is a 400 at runtime
// (payment_intent_automatic_payment_method_confirmation_allow_redirects_
// without_return_url): such sites must either gain a merchant-provided
// return_url or pin allow_redirects to "never".
type companionOpts struct {
	allowRedirectsNever bool
	returnURL           string
}

// confirmTrueRe matches confirm being set truthy across the pair-shaped
// SDKs (confirm: true / confirm=True / 'confirm' => true / Confirm =
// true / Confirm: stripe.Bool(true)). Both boundaries are guarded: a word
// character (or $) may not precede OR follow "confirm", so auditConfirm,
// confirmation_method and reconfirm never match.
var confirmTrueRe = regexp.MustCompile(`(?i)(^|[^\w$])["']?confirm["']?\s*(=>|=|:)\s*(stripe\.Bool\()?\s*true`)

// javaConfirmRe matches the builder spelling.
var javaConfirmRe = regexp.MustCompile(`setConfirm\(\s*true`)

// siteConfirmsVia is scoped to THIS call site — the param bag for pair
// languages, the chain for chain links, the receiver's own statements for
// Java builders — PLUS, when the bag was resolved through a variable
// (via "var:<name>"), mutations of that variable in the enclosing function
// (params.confirm = true / params["confirm"] = True / params.Confirm =
// stripe.Bool(true)). It must NOT scan unrelated text: one confirming
// create must never pin allow_redirects onto a sibling create.
func siteConfirmsVia(sp span, via string) bool {
	switch sp.site {
	case "statement":
		return sp.receiver != "" && javaConfirmRe.MatchString(receiverStatements(sp))
	case "chain-link":
		return javaConfirmRe.MatchString(sp.scope)
	}
	if confirmTrueRe.MatchString(sp.scope) {
		return true
	}
	if name := varBagName(via); name != "" {
		return varMutationRe(name, "confirm").MatchString(sp.funcText)
	}
	return false
}

// receiverStatements extracts the enclosing function's statements that
// mention THIS builder receiver as a whole token (declaration or call), so
// sibling builders never contaminate the check — including builders whose
// name is a suffix of another (`b` vs `sb`), and calls whose arguments wrap
// across lines (statements are split on ';', not newlines).
func receiverStatements(sp span) string {
	re := regexp.MustCompile(`(^|[^\w$])` + regexp.QuoteMeta(sp.receiver) + `\s*[.=]`)
	var out []string
	for _, stmt := range strings.Split(sp.funcText, ";") {
		if re.MatchString(stmt) {
			out = append(out, stmt)
		}
	}
	return strings.Join(out, ";")
}

// varBagName extracts the variable name from a "var:<name>" resolution.
func varBagName(via string) string {
	if strings.HasPrefix(via, "var:") {
		return strings.TrimPrefix(via, "var:")
	}
	return ""
}

// varMutationRe matches `<name>.<param> = true`-shaped mutations across the
// pair SDK spellings: params.confirm = true, params["confirm"] = True,
// params.Confirm = stripe.Bool(true), params[:confirm] = true.
func varMutationRe(name, param string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[^\w$])` + regexp.QuoteMeta(name) +
		`(\.|\[)["':]?` + param + `["']?\]?\s*(=|:)\s*(stripe\.Bool\()?\s*true`)
}

// varAssignRe matches any assignment of <param> onto the variable (used for
// return_url presence, where the assigned value is a string, not a bool).
func varAssignRe(name, param string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[^\w$])` + regexp.QuoteMeta(name) +
		`(\.|\[)["':]?` + param + `["']?\]?\s*=`)
}

// siteHasReturnURLVia is deliberately NARROW — the bag/chain, this
// receiver's own statements for Java, or this bag-variable's own mutations:
// claiming a return_url exists when it doesn't yields the runtime 400 this
// logic prevents; the reverse merely pins allow_redirects conservatively.
func siteHasReturnURLVia(sp span, via string) bool {
	text := sp.scope
	if sp.site == "statement" {
		text = receiverStatements(sp)
	}
	for _, tok := range []string{"return_url", "ReturnURL", "ReturnUrl", "setReturnUrl"} {
		if strings.Contains(text, tok) {
			return true
		}
	}
	if name := varBagName(via); name != "" {
		if varAssignRe(name, "return_url").MatchString(sp.funcText) ||
			varAssignRe(name, "ReturnURL").MatchString(sp.funcText) {
			return true
		}
	}
	return false
}

// nonRedirectMethods is a SAFELIST of methods documented to complete
// without leaving the page (cards, bank debits, vouchers, balance, Link's
// inline flow). Anything NOT listed — including methods added to the API
// after this list was written — is treated as redirect-capable, so unknown
// methods land on the conservative branch (gate, not pin).
var nonRedirectMethods = map[string]bool{
	"card": true, "card_present": true, "interac_present": true,
	"us_bank_account": true, "sepa_debit": true, "bacs_debit": true,
	"au_becs_debit": true, "acss_debit": true, "nz_bank_account": true,
	"boleto": true, "oxxo": true, "konbini": true, "multibanco": true,
	"customer_balance": true, "link": true,
}

// methodsNeedingRedirect returns the methods in a removed value that a
// confirm-site pin would silently disable.
func methodsNeedingRedirect(value string) []string {
	var hits []string
	for _, m := range quotedMethods(value) {
		if !nonRedirectMethods[m] {
			hits = append(hits, m)
		}
	}
	return hits
}

// companionText renders the language-correct insertion of
// automatic_payment_methods[enabled]=true for a removal site. site is the
// pair node kind (or statement/chain-link for Java builders); resource picks
// the typed SDKs' params-class prefix.
func companionText(spec langSpec, site, resource, receiver string, opts companionOpts) (string, bool) {
	prefix := pascalSingular(resource) // PaymentIntent | SetupIntent
	switch spec.name {
	case "ruby", "javascript", "typescript", "tsx":
		apm := "automatic_payment_methods: {enabled: true}"
		if opts.allowRedirectsNever {
			apm = "automatic_payment_methods: {enabled: true, allow_redirects: 'never'}"
		}
		if opts.returnURL != "" {
			apm += ", return_url: '" + opts.returnURL + "'"
		}
		return apm, true
	case "python":
		if site == "keyword_argument" {
			apm := `automatic_payment_methods={"enabled": True}`
			if opts.allowRedirectsNever {
				apm = `automatic_payment_methods={"enabled": True, "allow_redirects": "never"}`
			}
			if opts.returnURL != "" {
				apm += `, return_url="` + opts.returnURL + `"`
			}
			return apm, true
		}
		apm := `"automatic_payment_methods": {"enabled": True}`
		if opts.allowRedirectsNever {
			apm = `"automatic_payment_methods": {"enabled": True, "allow_redirects": "never"}`
		}
		if opts.returnURL != "" {
			apm += `, "return_url": "` + opts.returnURL + `"`
		}
		return apm, true
	case "php":
		apm := "'automatic_payment_methods' => ['enabled' => true]"
		if opts.allowRedirectsNever {
			apm = "'automatic_payment_methods' => ['enabled' => true, 'allow_redirects' => 'never']"
		}
		if opts.returnURL != "" {
			apm += ", 'return_url' => '" + opts.returnURL + "'"
		}
		return apm, true
	case "go":
		inner := "Enabled: stripe.Bool(true)"
		if opts.allowRedirectsNever {
			inner += `, AllowRedirects: stripe.String("never")`
		}
		apm := "AutomaticPaymentMethods: &stripe." + prefix + "AutomaticPaymentMethodsParams{" + inner + "}"
		if opts.returnURL != "" {
			apm += `, ReturnURL: stripe.String("` + opts.returnURL + `")`
		}
		return apm, true
	case "csharp":
		inner := "Enabled = true"
		if opts.allowRedirectsNever {
			inner += `, AllowRedirects = "never"`
		}
		apm := "AutomaticPaymentMethods = new " + prefix + "AutomaticPaymentMethodsOptions { " + inner + " }"
		if opts.returnURL != "" {
			apm += `, ReturnUrl = "` + opts.returnURL + `"`
		}
		return apm, true
	case "java":
		builder := prefix + "CreateParams.AutomaticPaymentMethods.builder().setEnabled(true)"
		if opts.allowRedirectsNever {
			builder += ".setAllowRedirects(" + prefix + "CreateParams.AutomaticPaymentMethods.AllowRedirects.NEVER)"
		}
		call := ".setAutomaticPaymentMethods(" + builder + ".build())"
		if opts.returnURL != "" {
			call += `.setReturnUrl("` + opts.returnURL + `")`
		}
		switch site {
		case "statement":
			if receiver == "" {
				return "", false
			}
			return receiver + call + ";", true
		case "chain-link":
			return call, true
		}
	}
	return "", false
}

// spliceFor wraps the companion text with the separator the span swallowed,
// so the replacement drops into the exact bytes the removal vacated.
func spliceFor(label, text string) string {
	switch label {
	case "pair+trailing-comma":
		return text + ","
	case "pair+leading-comma":
		return "," + text
	}
	return text
}
