// Package scan walks a directory, parses source with tree-sitter, and
// matches rule parameters against resolved Stripe call sites.
package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// parseTimeoutMicros, when non-zero, caps every parse's time budget —
// a test-only seam for deterministically forcing ParseStoppedEarly()
// (real usage never sets this; the default parser has no timeout).
var parseTimeoutMicros uint64

// Scan walks root and returns every finding matching rule, plus how many
// files were scanned (seen), parsed (passed the prefilter), skipped
// (rejected by the prefilter, counted directly rather than derived —
// a partially-parsed file counts toward both parsed and incomplete,
// so scanned-parsed-incomplete would double-subtract it and
// undercount true prefilter skips), and incomplete — supported but
// not fully inspected, either unreadable or only partially parsed
// (the parser's budget ran out on pathological input) — never
// silently folded into "skipped by prefilter".
func Scan(root string, rule catalog.Rule) (findings []catalog.Finding, scanned, parsed, skipped, incomplete int, err error) {
	// filepath.WalkDir lstats the root itself rather than following
	// it — a symlinked root would report IsDir() false and get
	// skipped entirely, scanning nothing and returning no error.
	// Resolve it upfront so a symlinked root is walked like any
	// other directory.
	if resolved, evalErr := filepath.EvalSymlinks(root); evalErr == nil {
		root = resolved
	}
	info, statErr := os.Stat(root)
	if statErr != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("cannot scan %q: %w", root, statErr)
	}
	if !info.IsDir() {
		return nil, 0, 0, 0, 0, errorcategory.Errorf(errorcategory.UserInput, "cannot scan %q: not a directory", root)
	}
	resources := map[string]bool{}
	for _, m := range rule.Match {
		for _, op := range m.Operations {
			if r := resourceFromOperation(op); r != "" {
				resources[r] = true
			}
		}
	}

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "vendor", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		spec, ok := specs[strings.ToLower(filepath.Ext(path))]
		if !ok {
			return nil
		}
		// Keep paths portable to Windows: findings' file:line keys and
		// test fixtures assume forward slashes.
		path = filepath.ToSlash(path)
		scanned++

		src, err := os.ReadFile(path)
		if err != nil {
			incomplete++
			return nil
		}

		if !prefilter(src, rule, spec) {
			skipped++
			return nil
		}
		parsed++

		lang := spec.lang()
		if lang == nil {
			return nil
		}
		parser := ts.NewParser(lang)
		if parseTimeoutMicros != 0 {
			parser.SetTimeoutMicros(parseTimeoutMicros) // test-only seam, see parseTimeoutMicros
		}
		tree, err := parser.Parse(src)
		if err != nil {
			return nil
		}
		// A parser that exhausts its budget on pathological input returns
		// a partial tree with a nil error — still scan what it did parse
		// (better than nothing), but don't let that read as full coverage:
		// a Stripe call beyond the parsed portion would otherwise be
		// silently missed.
		if tree.ParseStoppedEarly() {
			incomplete++
		}
		q, err := ts.NewQuery(spec.query, lang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "query error (%s): %v\n", spec.name, err)
			return nil
		}

		var tokens []string
		for r := range resources {
			tokens = append(tokens, spec.resourceTokens(r)...)
		}
		sort.Strings(tokens)
		idx := buildIndex(tree.RootNode(), spec, lang, src, tokens)

		// Scope tokens (and methods) per ParamMatch so a rule nested under
		// subscriptions can't resolve against a PaymentIntent call, and
		// vice versa.
		pmTokens, pmMethods, pmUnambiguous := perMatchTokensAndMethods(rule, spec)

		seen := map[string]bool{}
		for _, m := range q.Execute(tree) {
			for _, c := range m.Captures {
				if c.Name != "key" {
					continue
				}
				// None of these queries are field-anchored (PHP's
				// array_element_initializer has no key/value field names
				// at all to anchor to), so each would also match a bare
				// string/identifier sitting in the *value* position —
				// e.g. create({"description": "payment_method_types"}).
				// Reject anything that isn't structurally in the key slot.
				if !keyInKeyPosition(c.Node, spec, lang) {
					continue
				}
				// Ruby's simple_symbol (hash-rocket ":key => value") keeps
				// its leading ":" as part of its own text; strip it too.
				keyText := strings.Trim(c.Text(src), `"':`)

				for pmIdx, pm := range rule.Match {
					if !containsStr(spec.spell(leafParam(pm.Param)), keyText) {
						continue
					}
					// Enforce the rule's nesting path exactly: a rule for
					// payment_settings.payment_method_types must not match a
					// top-level payment_method_types, and vice versa.
					if !pathSatisfied(c.Node, pm.Param, spec, lang, src) {
						continue
					}
					res := resolve(c.Node, spec, lang, src, pmTokens[pmIdx], pmMethods[pmIdx], pmUnambiguous[pmIdx], idx)
					if !res.confirmed {
						continue
					}
					p := c.Node.StartPoint()
					key := fmt.Sprintf("%s:%d:%d", path, p.Row+1, p.Column+1)
					if seen[key] {
						continue
					}
					seen[key] = true
					findings = append(findings, catalog.Finding{
						Value:    extractValue(c.Node, spec, lang, src),
						File:     path,
						Line:     int(p.Row) + 1,
						Col:      int(p.Column) + 1,
						RuleID:   rule.ID,
						Severity: rule.Severity,
						Param:    pm.Param,
						Anchor:   res.anchor,
						Via:      res.via,
						Message:  rule.Message,
						Docs:     rule.Docs,
					})
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, scanned, parsed, skipped, incomplete, walkErr
	}
	return findings, scanned, parsed, skipped, incomplete, nil
}

// perMatchTokensAndMethods returns, per rule.Match entry: its resource
// tokens; the SDK method name(s) consistent with its operations (see
// expectedMethodNames) — e.g. a create-only ParamMatch doesn't also
// match PaymentIntent.modify — with no constraint at all when its
// operations mix modeled and unmodeled shapes, rather than guessing
// from only the modeled ones; and its unambiguous tokens (see
// langSpec.unambiguousTokens).
func perMatchTokensAndMethods(rule catalog.Rule, spec langSpec) (tokens, methods, unambiguous [][]string) {
	tokens = make([][]string, len(rule.Match))
	methods = make([][]string, len(rule.Match))
	unambiguous = make([][]string, len(rule.Match))
	for i, pm := range rule.Match {
		set := map[string]bool{}
		methodSet := map[string]bool{}
		unambiguousSet := map[string]bool{}
		mixedShapes := false
		for _, op := range pm.Operations {
			if r := resourceFromOperation(op); r != "" {
				for _, t := range spec.resourceTokens(r) {
					set[t] = true
				}
				if spec.unambiguousTokens != nil {
					for _, t := range spec.unambiguousTokens(r) {
						unambiguousSet[t] = true
					}
				}
			}
			opMethods := expectedMethodNames(op)
			if opMethods == nil {
				mixedShapes = true // an unmodeled operation shape - don't guess
				continue
			}
			for _, mname := range opMethods {
				methodSet[mname] = true
			}
		}
		for t := range set {
			tokens[i] = append(tokens[i], t)
		}
		sort.Strings(tokens[i])
		if spec.unambiguousTokens != nil {
			for t := range unambiguousSet {
				unambiguous[i] = append(unambiguous[i], t)
			}
			sort.Strings(unambiguous[i])
		}
		if !mixedShapes {
			for mname := range methodSet {
				methods[i] = append(methods[i], mname)
			}
			sort.Strings(methods[i])
		}
	}
	return tokens, methods, unambiguous
}

// prefilter is a cheap substring check run on every file before parsing.
func prefilter(src []byte, rule catalog.Rule, spec langSpec) bool {
	s := string(src)
	for _, pm := range rule.Match {
		for _, sp := range spec.spell(leafParam(pm.Param)) {
			if strings.Contains(s, sp) {
				return true
			}
		}
	}
	return false
}

// pathSatisfied checks that the matched key's enclosing param-bag keys match
// the rule's dotted path. For "a.b" the key must sit inside a bag keyed "a";
// for a bare "b" it must sit at the top level of the call's param bag.
func pathSatisfied(key *ts.Node, param string, spec langSpec, lang *ts.Language, src []byte) bool {
	want := strings.Split(param, ".")
	want = want[:len(want)-1] // drop the leaf; already matched
	got := ancestorKeys(key, spec, lang, src)
	if len(want) != len(got) {
		return false
	}
	for i := range want {
		if !containsStr(spec.spell(want[i]), got[i]) {
			return false
		}
	}
	return true
}

// resourceFromOperation: "POST /v1/payment_intents/{intent}" -> "payment_intents"
func resourceFromOperation(op string) string {
	parts := strings.Fields(op)
	if len(parts) != 2 {
		return ""
	}
	segs := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	if len(segs) < 2 {
		return ""
	}
	last := "" // segs[0] is the version (v1); skip it
	for _, s := range segs[1:] {
		if strings.HasPrefix(s, "{") {
			continue
		}
		last = s
	}
	return last
}

// expectedMethodNames maps op's HTTP method + path shape to the SDK
// method name(s) real Stripe client libraries use for it — modeled for
// create only (a plain POST with no path parameter: "create" is
// spelled identically across Python/Ruby/PHP/JS/TS), since that's the
// only shape any current rule's operations use. Any other shape
// returns nil (no constraint), so an unmodeled verb falls back to
// resource-only matching rather than guessing a wrong name.
// create_async is Python's asyncio spelling of the same create call.
func expectedMethodNames(op string) []string {
	parts := strings.Fields(op)
	if len(parts) != 2 || parts[0] != "POST" {
		return nil
	}
	segs := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	for _, s := range segs[1:] {
		if strings.HasPrefix(s, "{") {
			return nil // POST with a path parameter (update/action) - not modeled
		}
	}
	return []string{"create", "create_async"}
}

// leafParam: "payment_settings.payment_method_types" -> "payment_method_types"
func leafParam(p string) string {
	if i := strings.LastIndex(p, "."); i >= 0 {
		return p[i+1:]
	}
	return p
}

func nodeText(n *ts.Node, src []byte) string {
	if n == nil {
		return ""
	}
	s, e := n.StartByte(), n.EndByte()
	if int(e) > len(src) {
		return ""
	}
	return string(src[s:e])
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i]) + " …"
	}
	if len(s) > 90 {
		return s[:90] + " …"
	}
	return s
}

// SortFindings orders findings by file, then by line, for stable output.
func SortFindings(findings []catalog.Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		return findings[i].Line < findings[j].Line
	})
}

// extractValue returns the value expression paired with a matched key.
func extractValue(key *ts.Node, spec langSpec, lang *ts.Language, src []byte) string {
	if key.Type(lang) == "shorthand_property_identifier" {
		// {payment_method_types} is sugar for {payment_method_types:
		// payment_method_types} — the identifier doubles as its own
		// value, and isn't nested in a pair to climb to.
		return firstLine(nodeText(key, src))
	}
	kinds := spec.pairKinds
	if kinds == nil { // java: the key IS the method name of the invocation
		kinds = spec.anchorKinds
	}
	for cur := key.Parent(); cur != nil; cur = cur.Parent() {
		if !containsStr(kinds, cur.Type(lang)) {
			continue
		}
		if n := cur.NamedChildCount(); n > 1 {
			if v := cur.NamedChild(n - 1); v != nil && !sameSpan(v, key) {
				return firstLine(nodeText(v, src))
			}
		}
		return ""
	}
	return ""
}

// ancestorKeys returns the keys of enclosing param-bag entries, outermost
// first, excluding the matched key's own entry. The climb stops at the
// anchor call, since e.g. C#'s `options = new(){...}` would otherwise be
// miscounted as a nesting level.
func ancestorKeys(key *ts.Node, spec langSpec, lang *ts.Language, src []byte) []string {
	bound := climbAnchor(key, spec, lang)
	isPair := func(k string) bool {
		for _, want := range spec.pairKinds {
			if k == want {
				return true
			}
		}
		return false
	}
	// Skip the key's own enclosing pair before collecting — except a
	// shorthand property identifier, which has no enclosing pair of
	// its own to skip (it's a direct child of object, not nested in
	// one), so collection starts climbing from there immediately.
	cur := key.Parent()
	if key.Type(lang) != "shorthand_property_identifier" {
		for cur != nil && !isPair(cur.Type(lang)) {
			cur = cur.Parent()
		}
		if cur != nil {
			cur = cur.Parent()
		}
	}

	var keys []string
	for ; cur != nil; cur = cur.Parent() {
		if bound != nil && sameSpan(cur, bound) {
			break
		}
		if !isPair(cur.Type(lang)) {
			continue
		}
		if kn := firstNamedChild(cur); kn != nil {
			k := strings.Trim(nodeText(kn, src), `"':`)
			if isRequestEnvelope(cur, bound, k) {
				continue // the SDK's reserved request-body keyword, not a nested API field
			}
			keys = append([]string{k}, keys...)
		}
	}
	return keys
}

// isRequestEnvelope reports whether cur is a pair/keyword-argument node
// directly inside anchor's own argument list, spelling the SDK's
// reserved request-envelope keyword ("params") rather than a real
// nested API field name — e.g. Python/Ruby's
// client.v1.payment_intents.create(params: {...}).
func isRequestEnvelope(cur, anchor *ts.Node, key string) bool {
	if key != "params" || anchor == nil {
		return false
	}
	n := anchor.NamedChildCount()
	if n == 0 {
		return false
	}
	argsNode := anchor.NamedChild(n - 1)
	return argsNode != nil && cur.Parent() != nil && sameSpan(cur.Parent(), argsNode)
}

// keyInKeyPosition reports whether key — a query capture — sits in the
// key/name slot of its enclosing pair-shaped node (pairKinds), rather
// than its value slot. Climbs past wrapper nodes (e.g. Python's
// string_content inside string inside pair) to the pairKind's own
// direct child first. For Java (pairKinds is nil, key is the invoked
// method's own name) the equivalent slot is the method_invocation's
// second-to-last named child — the one immediately before its trailing
// argument_list, whether or not it also has an explicit receiver.
func keyInKeyPosition(key *ts.Node, spec langSpec, lang *ts.Language) bool {
	if key.Type(lang) == "shorthand_property_identifier" {
		// Always in key position — it has no separate value slot to
		// be mistaken for.
		return true
	}
	isParent := func(n *ts.Node) bool {
		if spec.pairKinds != nil {
			return containsStr(spec.pairKinds, n.Type(lang))
		}
		return containsStr(spec.anchorKinds, n.Type(lang))
	}
	child := key
	for child.Parent() != nil && !isParent(child.Parent()) {
		child = child.Parent()
	}
	parent := child.Parent()
	if parent == nil {
		return false
	}
	if spec.pairKinds == nil {
		n := parent.NamedChildCount()
		return n >= 2 && sameSpan(parent.NamedChild(n-2), child)
	}
	return parent.NamedChildCount() > 0 && sameSpan(parent.NamedChild(0), child)
}

func firstNamedChild(n *ts.Node) *ts.Node {
	if n.NamedChildCount() == 0 {
		return nil
	}
	c := n.NamedChild(0)
	// Descend through wrapper nodes (e.g. php string -> string_content).
	for c != nil && c.NamedChildCount() == 1 {
		c = c.NamedChild(0)
	}
	return c
}
