package doctor

// The scanner: walk, prefilter, parse, and match rule parameters against
// resolved Stripe call sites.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

func scan(root string, rule Rule) (findings []Finding, scanned, parsed int, err error) {
	info, statErr := os.Stat(root)
	if statErr != nil {
		return nil, 0, 0, fmt.Errorf("cannot scan %q: %w", root, statErr)
	}
	if !info.IsDir() {
		return nil, 0, 0, fmt.Errorf("cannot scan %q: not a directory", root)
	}
	// Resource names the rule's operations imply, e.g. "payment_intents".
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
			// Never scan (or edit) vendored/generated trees.
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
		scanned++

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Prefilter: no spelling of any rule param present -> never parse.
		// This is what keeps a large repo fast.
		if !prefilter(src, rule, spec) {
			return nil
		}
		parsed++

		lang := spec.lang()
		if lang == nil {
			return nil
		}
		tree, err := ts.NewParser(lang).Parse(src)
		if err != nil {
			return nil
		}
		q, err := ts.NewQuery(spec.query, lang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "query error (%s): %v\n", spec.name, err)
			return nil
		}

		// Union of every SDK token the rule's resources can appear as (for
		// the file index); resolution re-filters per ParamMatch below.
		var tokens []string
		for r := range resources {
			tokens = append(tokens, spec.resourceTokens(r)...)
		}
		sort.Strings(tokens)
		idx := buildIndex(tree.RootNode(), spec, lang, src, tokens)

		// Per-ParamMatch tokens: a nested rule scoped to subscriptions must
		// not resolve against a PaymentIntent call, and vice versa.
		pmTokens := make([][]string, len(rule.Match))
		for i, pm := range rule.Match {
			set := map[string]bool{}
			for _, op := range pm.Operations {
				if r := resourceFromOperation(op); r != "" {
					for _, t := range spec.resourceTokens(r) {
						set[t] = true
					}
				}
			}
			for t := range set {
				pmTokens[i] = append(pmTokens[i], t)
			}
			sort.Strings(pmTokens[i])
		}

		seen := map[string]bool{}
		for _, m := range q.Execute(tree) {
			for _, c := range m.Captures {
				if c.Name != "key" {
					continue
				}
				keyText := strings.Trim(c.Text(src), `"'`)

				for pmIdx, pm := range rule.Match {
					if keyText != spec.spell(leafParam(pm.Param)) {
						continue
					}
					// Enforce the rule's nesting path exactly: a rule for
					// payment_settings.payment_method_types must not match a
					// top-level payment_method_types, and vice versa.
					if !pathSatisfied(c.Node, pm.Param, spec, lang, src) {
						continue
					}
					res := resolve(c.Node, spec, lang, src, pmTokens[pmIdx], idx)
					if !res.confirmed {
						continue
					}
					p := c.Node.StartPoint()
					key := fmt.Sprintf("%s:%d:%d", path, p.Row+1, p.Column+1)
					if seen[key] {
						continue
					}
					seen[key] = true
					findings = append(findings, Finding{
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
		return nil, scanned, parsed, walkErr
	}
	return findings, scanned, parsed, nil
}

// extractValue returns the value expression paired with a matched key: the
// last named child of the enclosing pair-like node, or for Java's flat builder
// plain substring test in this SDK's spelling. Cheap; runs on every file.
func prefilter(src []byte, rule Rule, spec langSpec) bool {
	s := string(src)
	for _, pm := range rule.Match {
		if strings.Contains(s, spec.spell(leafParam(pm.Param))) {
			return true
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
		if spec.spell(want[i]) != got[i] {
			return false
		}
	}
	return true
}

// ancestorKeys returns the keys of enclosing param-bag entries, outermost
// first, excluding the matched key's own entry.
//
// The climb stops at the enclosing param bag. Without that bound, a binding
// like C#'s `options = new(){...}` — an assignment_expression, the same node
// kind as a parameter entry — would be miscounted as a nesting level and every
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
	// Skip the version segment; take the last non-templated segment.
	last := ""
	for _, s := range segs[1:] {
		if strings.HasPrefix(s, "{") {
			continue
		}
		last = s
	}
	return last
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

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		return findings[i].Line < findings[j].Line
	})
}

// style the argument list of the add* invocation itself.
func extractValue(key *ts.Node, spec langSpec, lang *ts.Language, src []byte) string {
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

// prefilter reports whether src could possibly contain a rule param, using a
// top-level parameter in that bag would fail its path check.
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
	// Skip the key's own enclosing pair before collecting.
	cur := key.Parent()
	for cur != nil && !isPair(cur.Type(lang)) {
		cur = cur.Parent()
	}
	if cur != nil {
		cur = cur.Parent()
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
			keys = append([]string{strings.Trim(nodeText(kn, src), `"'`)}, keys...)
		}
	}
	return keys
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
