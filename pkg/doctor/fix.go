package doctor

// fix.go — span-verified remediation. For each finding, compute the span that
// removes the whole parameter entry (pair + separator, or builder link, or
// whole statement), apply the edits in memory, REPARSE the result, and only
// write files whose reparse is clean.
//
// Rules with a Companion fork by account API version: at/after the rule's
// cutoff the parameter is simply removed; below it (or when the version is
// unknowable) the removal becomes a REPLACEMENT that inserts the companion
// parameter in the same spot, because bare removal would silently change
// behavior there (verified live: a bare pre-cutoff PaymentIntent resolves to
// card-only).

import (
	"fmt"
	"os"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// siteRec is one located finding: its span plus language context.
type siteRec struct {
	f    Finding
	spec langSpec
	sp   span
}

// callSite is the decision unit: a pair-language finding alone, or every
// finding of one Java builder together.
type decisionSite struct{ recs []siteRec }

type fileEdit struct {
	spec  langSpec
	spans []span
}

// locateSpans re-parses each finding's file (cached) and computes its
// removal span.
func locateSpans(findings []Finding) []siteRec {
	srcCache := map[string][]byte{}
	var recs []siteRec
	for _, f := range findings {
		spec := specs[strings.ToLower(ext(f.File))]
		src, ok := srcCache[f.File]
		if !ok {
			b, rerr := os.ReadFile(f.File)
			if rerr != nil {
				continue
			}
			src = b
			srcCache[f.File] = b
		}
		lang := spec.lang()
		tree, terr := ts.NewParser(lang).Parse(src)
		if terr != nil {
			continue
		}
		key := tree.RootNode().NamedNodeAtByte(byteAt(src, f.Line, f.Col))
		if key == nil {
			continue
		}
		sp, ok := removalSpan(key, spec, lang, src)
		if !ok {
			continue
		}
		recs = append(recs, siteRec{f: f, spec: spec, sp: sp})
	}
	return recs
}

// groupCallSites groups records into call sites. A pair-language finding is
// its own site; Java splits one builder's method list across N findings (one
// per addPaymentMethodType), which MUST be judged as a unit — a gate that
// skips one link but removes another would leave half a parameter next to an
// inserted companion, code the API rejects outright.
func groupCallSites(recs []siteRec) []*decisionSite {
	var sites []*decisionSite
	byGroup := map[string]*decisionSite{}
	for _, r := range recs {
		if r.sp.group == "" {
			sites = append(sites, &decisionSite{recs: []siteRec{r}})
			continue
		}
		k := r.f.File + "|" + r.sp.group
		if byGroup[k] == nil {
			byGroup[k] = &decisionSite{}
			sites = append(sites, byGroup[k])
		}
		byGroup[k].recs = append(byGroup[k].recs, r)
	}
	return sites
}

// siteIntent classifies at SITE granularity: a builder adding card+ideal is
// a static multi-method list, not a deliberate single restriction.
func siteIntent(st *decisionSite) string {
	if len(st.recs) == 1 {
		return classifyIntent(st.recs[0].f.Value)
	}
	for _, r := range st.recs {
		if classifyIntent(r.f.Value) == "dynamic" {
			return "dynamic"
		}
	}
	return "static"
}

// siteCombinedValue joins the site's values for union-level checks.
func siteCombinedValue(st *decisionSite) string {
	if len(st.recs) == 1 {
		return st.recs[0].f.Value
	}
	var vs []string
	for _, r := range st.recs {
		vs = append(vs, r.f.Value)
	}
	return strings.Join(vs, ", ")
}

// siteCompanion decides the companion outcome for one site. gated=true means
// the site must not be edited at all (a confirm-redirect skip was recorded).
// Server-side-confirmation sites (confirm:true, no return_url) are evaluated
// on BOTH branches of the version fork: after removal,
// automatic_payment_methods is in effect either way (inserted below the
// cutoff, default-on above it), so the runtime 400 exists on both. Per
// confirm site:
//
//	--return-url given         → companion + return_url
//	list had redirect-capable
//	methods, no return_url     → GATE (pinning drops methods the merchant
//	                             used; removal alone 400s)

// otherwise                  → companion + allow_redirects:"never"
func siteCompanion(st *decisionSite, rule Rule, dec *companionDecision, report *FixReport) (replaceText, variant string, gated bool) {
	lead := st.recs[0]
	if dec == nil || rule.Companion == nil || lead.f.Param != rule.Companion.ForParam ||
		alreadyHasCompanion(lead.sp, rule.Companion.Param) {
		return "", "", false
	}
	res := companionResourceFor(lead.spec, lead.f.Anchor, lead.sp.funcText, rule.Companion.Resources)
	if res == "" {
		return "", "", false
	}
	confirmSite := siteConfirmsVia(lead.sp, lead.f.Via) && !siteHasReturnURLVia(lead.sp, lead.f.Via)
	if confirmSite && dec.returnURL == "" {
		if rm := methodsNeedingRedirect(siteCombinedValue(st)); len(rm) > 0 {
			report.Skipped = append(report.Skipped, SkippedFinding{File: lead.f.File, Line: lead.f.Line, Intent: "confirm-redirect",
				Reason: "server-side confirmation uses redirect-capable method(s) " + strings.Join(rm, ", ") +
					" — removal would either fail at runtime (no return_url) or drop them (allow_redirects:\"never\"); rerun with --return-url <url>"})
			return "", "", true
		}
	}
	// Confirm sites force the explicit insert even on the omit branch —
	// that is where the pin/return_url must live.
	if !dec.insert && !confirmSite {
		return "", "", false
	}
	var opts companionOpts
	if confirmSite {
		if dec.returnURL != "" {
			opts.returnURL = dec.returnURL
			variant = "return_url"
		} else {
			opts.allowRedirectsNever = true
			variant = "never-pin"
		}
	}
	text, ok := companionText(lead.spec, lead.sp.site, res, lead.sp.receiver, opts)
	if !ok {
		return "", "", false
	}
	if variant == "" {
		variant = "plain"
	}
	return spliceFor(lead.sp.label, text), variant, false
}

// applyFileEdits splices every span in memory, reparses, and (when apply is
// set) writes only reparse-clean files.
func applyFileEdits(edits map[string]*fileEdit, apply bool, report *FixReport) {
	var files []string
	for f := range edits {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, file := range files {
		fe := edits[file]
		src, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		// Apply spans back-to-front so offsets stay valid.
		sort.Slice(fe.spans, func(i, j int) bool { return fe.spans[i].start > fe.spans[j].start })
		out := src
		ff := FixFile{Path: file}
		for _, sp := range fe.spans {
			if sp.replace == "" {
				// Pure removal: widen to the whole line when only whitespace
				// would remain, so no blank line marks the spot.
				sp = expandToLine(sp, src)
			}
			if int(sp.end) > len(out) || sp.start >= sp.end {
				continue
			}
			spliced := append(append([]byte{}, out[:sp.start]...), sp.replace...)
			out = append(spliced, out[sp.end:]...)
			ff.BytesRemoved += int(sp.end - sp.start)
			ff.BytesAdded += len(sp.replace)
			if sp.replace != "" && report.Companion != nil {
				report.Companion.Inserts++
			}
			ff.Edits = append(ff.Edits, FixEdit{Start: sp.start, End: sp.end, Label: sp.label, Variant: sp.variant})
		}
		writeFixedFile(fe, &ff, out, apply, report)
		report.Files = append(report.Files, ff)
	}
}

// writeFixedFile reparses the edited buffer and writes it when clean.
func writeFixedFile(fe *fileEdit, ff *FixFile, out []byte, apply bool, report *FixReport) {
	lang := fe.spec.lang()
	tree, err := ts.NewParser(lang).Parse(out)
	if err != nil || tree.RootNode().HasError() {
		ff.Reparse = "error"
		report.AllClean = false
		return
	}
	ff.Reparse = "clean"
	if !apply {
		return
	}
	info, statErr := os.Stat(ff.Path)
	mode := os.FileMode(0o644)
	if statErr == nil {
		mode = info.Mode()
	}
	if werr := os.WriteFile(ff.Path, out, mode); werr == nil {
		ff.Written = true
	} else {
		// Record and continue: a half-applied tree must still produce a
		// complete report of what happened.
		ff.Error = werr.Error()
		report.AllClean = false
	}
}

// fixRun computes removal spans per file, applies them in memory, reparses,
// and — only when apply is true AND the reparse is clean — writes the file.
// A file that fails reparse is never written. dec carries the resolved
// companion fork (nil when the rule has no companion).
func fixRun(root string, rule Rule, apply, includeAll bool, dec *companionDecision) (*FixReport, error) {
	if rule.Action != "remove" {
		return nil, fmt.Errorf("rule %s is action=%q: it detects and advises but has no automatic fix — run `doctor` and follow %s", rule.ID, rule.Action, rule.Docs)
	}
	findings, _, _, err := scan(root, rule)
	if err != nil {
		return nil, err
	}
	report := &FixReport{Command: "fix", Applied: apply, AllClean: true}
	if rule.Companion != nil && dec != nil {
		mode := "omit"
		if dec.insert {
			mode = "insert"
		}
		report.Companion = &CompanionReport{Param: rule.Companion.Param, Mode: mode, Reason: dec.reason, OldestVersion: dec.oldest, Account: dec.account}
	}

	edits := map[string]*fileEdit{}
	confirmNever, confirmWithURL := 0, 0
	for _, st := range groupCallSites(locateSpans(findings)) {
		lead := st.recs[0]
		if intent := siteIntent(st); !includeAll && (intent == "dynamic" || intent == "deliberate") {
			reason := "value is computed at runtime — review the routing logic (use --all to override)"
			if intent == "deliberate" {
				reason = "single-method restriction looks intentional — consider excluded_payment_method_types (use --all to override; note wallet-class methods use the wallets hash instead)"
			}
			report.Skipped = append(report.Skipped, SkippedFinding{File: lead.f.File, Line: lead.f.Line, Intent: intent, Reason: reason})
			continue
		}
		replaceText, variant, gated := siteCompanion(st, rule, dec, report)
		if gated {
			continue
		}
		for i, r := range st.recs {
			sp := r.sp
			if i == 0 && replaceText != "" {
				sp.replace = replaceText
				sp.variant = variant
				label := "→+" + rule.Companion.Param
				switch variant {
				case "never-pin":
					label += `(allow_redirects:"never")`
					confirmNever++
					if report.Companion != nil {
						report.Companion.PinnedSites = append(report.Companion.PinnedSites, SiteRef{File: r.f.File, Line: r.f.Line})
					}
				case "return_url":
					label += "+return_url"
					confirmWithURL++
				}
				sp.label += label
			}
			if edits[r.f.File] == nil {
				edits[r.f.File] = &fileEdit{spec: r.spec}
			}
			edits[r.f.File].spans = append(edits[r.f.File].spans, sp)
		}
	}

	applyFileEdits(edits, apply, report)
	companionNotes(report, confirmNever, confirmWithURL)
	return report, nil
}

// companionNotes summarizes the confirm-site outcomes for the report.
func companionNotes(report *FixReport, confirmNever, confirmWithURL int) {
	if report.Companion == nil {
		return
	}
	if report.Companion.Mode == "omit" && (confirmNever > 0 || confirmWithURL > 0) {
		report.Companion.Notes = append(report.Companion.Notes,
			"confirm site(s) received an explicit companion despite the omit verdict: automatic_payment_methods is default-on at this account's version, so server-side confirmation still needs the pin or a return_url to avoid a runtime 400")
	}
	if confirmNever > 0 {
		report.Companion.Notes = append(report.Companion.Notes, fmt.Sprintf(
			"%d server-side-confirmation site(s) got allow_redirects:\"never\" (no return_url present): APM+confirm without a return_url is a runtime 400; rerun with --return-url <url> to enable redirect-based payment methods instead", confirmNever))
	}
	if confirmWithURL > 0 {
		report.Companion.Notes = append(report.Companion.Notes, fmt.Sprintf(
			"%d server-side-confirmation site(s) gained the provided return_url alongside the companion", confirmWithURL))
	}
}

func ext(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i:]
	}
	return ""
}

// byteAt converts a 1-based line:col back to a byte offset.
func byteAt(src []byte, line, col int) uint32 {
	l := 1
	for i := 0; i < len(src); i++ {
		if l == line {
			return uint32(i + col - 1)
		}
		if src[i] == '\n' {
			l++
		}
	}
	return 0
}
