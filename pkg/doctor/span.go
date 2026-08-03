package doctor

// Removal spans: the byte ranges that delete a matched parameter cleanly
// per language shape, with whole-line expansion for pure removals.

import (
	"fmt"

	ts "github.com/odvcencio/gotreesitter"
)

type span struct {
	start, end uint32
	label      string
	// site describes the removal-site shape for companion insertion: the pair
	// node kind for pair languages, "statement"/"chain-link" for Java.
	site string
	// receiver is the builder variable text for Java statement sites.
	receiver string
	// group identifies the builder INSTANCE for Java dedupe: same-named
	// builders in different functions, and distinct chains in one file, must
	// not collapse onto each other.
	group string
	// scope is the text checked for an already-present companion: the param
	// bag for pair languages, the whole chain for chain links, the enclosing
	// function for builder statements.
	scope string
	// funcText is the enclosing function's text, for create-evidence checks
	// that the anchor alone cannot answer (Go's shared params struct).
	funcText string
	// replace, when non-empty, turns the deletion into a splice; variant
	// says which companion shape it is (plain | never-pin | return_url).
	replace string
	variant string
}

// climbLast walks ancestors of n and returns the OUTERMOST node whose kind is
// in kinds (nil when none).
func climbLast(n *ts.Node, kinds []string, lang *ts.Language) *ts.Node {
	var last *ts.Node
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		if containsStr(kinds, cur.Type(lang)) {
			last = cur
		}
	}
	return last
}

// climbFirst returns the NEAREST ancestor of n whose kind is in kinds.
func climbFirst(n *ts.Node, kinds []string, lang *ts.Language) *ts.Node {
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		if containsStr(kinds, cur.Type(lang)) {
			return cur
		}
	}
	return nil
}

// removalSpan computes the byte range that deletes a matched parameter
// entirely, per language shape.
func removalSpan(key *ts.Node, spec langSpec, lang *ts.Language, src []byte) (span, bool) {
	if spec.pairKinds == nil {
		// Java builder: the key is the method name of an invocation link.
		var inv *ts.Node
		for cur := key.Parent(); cur != nil; cur = cur.Parent() {
			if containsStr(spec.anchorKinds, cur.Type(lang)) {
				inv = cur
				break
			}
		}
		if inv == nil {
			return span{}, false
		}
		// Standalone statement (paramsBuilder.addX("card");) → remove the
		// whole statement; chain link (.addX("card")) → remove from the end
		// of the receiver through the end of this link.
		recv := inv.NamedChild(0)
		fn := climbFirst(inv, spec.funcKinds, lang)
		fnStart := uint32(0)
		fnText := nodeText(fn, src)
		if fn != nil {
			fnStart = fn.StartByte()
		} else {
			// Top-level code has no enclosing function; the whole file is
			// the scope (attribution checks are name-scoped, so this is
			// safe, and missing it hides top-level confirm mutations).
			fnText = string(src)
		}
		if p := inv.Parent(); p != nil && p.Type(lang) == "expression_statement" {
			receiver := nodeText(recv, src)
			return span{start: p.StartByte(), end: p.EndByte(), label: "statement", site: "statement",
				receiver: receiver,
				// Builder instance = receiver name scoped to its function.
				group:    fmt.Sprintf("stmt|%d|%s", fnStart, receiver),
				scope:    fnText,
				funcText: fnText}, true
		}
		if recv == nil {
			return span{}, false
		}
		// Every link of one chain shares the same OUTERMOST invocation node,
		// which is exactly the builder-instance identity we need.
		outer := climbLast(key, spec.anchorKinds, lang)
		outerStart := inv.StartByte()
		if outer != nil {
			outerStart = outer.StartByte()
		}
		return span{start: recv.EndByte(), end: inv.EndByte(), label: "chain-link", site: "chain-link",
			group:    fmt.Sprintf("chain|%d", outerStart),
			scope:    nodeText(outer, src),
			funcText: fnText}, true
	}

	// Pair-shaped languages: the enclosing pair node plus one separator.
	var pair *ts.Node
	for cur := key.Parent(); cur != nil; cur = cur.Parent() {
		if containsStr(spec.pairKinds, cur.Type(lang)) {
			pair = cur
			break
		}
	}
	if pair == nil {
		return span{}, false
	}
	site := pair.Type(lang)
	// The companion checks are scoped to THIS param bag (the pair's parent
	// literal) and this call's enclosing function; top-level code falls
	// back to the whole file (var-attribution checks are name-scoped).
	pairFunc := nodeText(climbFirst(pair, spec.funcKinds, lang), src)
	if pairFunc == "" {
		pairFunc = string(src)
	}
	base := span{site: site,
		scope:    nodeText(pair.Parent(), src),
		funcText: pairFunc}
	s, e := pair.StartByte(), pair.EndByte()
	// Prefer swallowing the trailing comma; else the leading one.
	if i := skipWS(src, int(e), +1); i < len(src) && src[i] == ',' {
		base.start, base.end, base.label = s, uint32(i+1), "pair+trailing-comma"
		return base, true
	}
	if i := skipWS(src, int(s)-1, -1); i >= 0 && src[i] == ',' {
		base.start, base.end, base.label = uint32(i), e, "pair+leading-comma"
		return base, true
	}
	base.start, base.end, base.label = s, e, "pair"
	return base, true
}

// expandToLine widens a pure removal to whole lines when only whitespace
// would remain around it, so deletions never leave blank or whitespace-only
// lines behind (multi-line spans expand to their outer line boundaries).
func expandToLine(sp span, src []byte) span {
	ls := int(sp.start)
	for ls > 0 && src[ls-1] != '\n' {
		ls--
	}
	le := int(sp.end)
	for le < len(src) && src[le] != '\n' {
		le++
	}
	if !allWS(src[ls:sp.start]) || !allWS(src[sp.end:le]) {
		return sp
	}
	if le < len(src) {
		le++ // swallow the newline
	}
	out := sp
	out.start, out.end, out.label = uint32(ls), uint32(le), sp.label+"+line"
	return out
}

func allWS(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' {
			return false
		}
	}
	return true
}

// ---------- companion (version-forked replace) ----------

// companionDecision is the resolved version fork: insert the companion, or

func skipWS(src []byte, i, dir int) int {
	for i >= 0 && i < len(src) && (src[i] == ' ' || src[i] == '\t' || (dir > 0 && src[i] == '\n')) {
		i += dir
	}
	return i
}
