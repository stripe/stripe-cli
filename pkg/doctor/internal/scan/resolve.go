package scan

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// This file resolves a matched parameter to an API operation two ways:
// directly (the bag is inside the call) or indirectly (the bag is bound
// to a variable that reaches a call, or declared with a params type) —
// real integrations often build the bag separately and pass it later:
//
//	params = {'payment_method_types': [...], 'amount': 1099}
//	...
//	stripe.PaymentIntent.create(**params)

// resolution records how a match was tied to an operation.
type resolution struct {
	confirmed bool
	via       string // "direct", "var:params", "type:options"
	anchor    string // human-readable site text
}

// fileIndex is the per-file structural summary the resolver needs, built
// once so resolution stays linear rather than quadratic.
type fileIndex struct {
	// callSites are calls whose *callee* names a rule resource — scoped
	// to the callee, not the whole node, so an enclosing route handler
	// like app.post('/create-payment-intent', ...) doesn't match just
	// because a Stripe call appears somewhere in its body.
	callSites []callSite
	// typedDecls are declarations naming a params type, e.g.
	// `PaymentIntentCreateOptions options;` for C# target-typed new().
	typedDecls []typedDecl
}

type typedDecl struct {
	text  string
	start uint32
}

type callSite struct {
	callee string
	args   string
	start  uint32 // for function-scope checks
}

func buildIndex(root *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens []string) fileIndex {
	var idx fileIndex
	walk(root, func(n *ts.Node) {
		kind := n.Type(lang)
		if containsStr(spec.anchorKinds, kind) {
			callee, args := splitCall(n, src)
			if containsAny(callee, tokens) {
				idx.callSites = append(idx.callSites, callSite{callee: callee, args: args, start: n.StartByte()})
			}
		}
		if containsStr(spec.declKinds, kind) {
			if t := nodeText(n, src); containsAny(t, tokens) {
				idx.typedDecls = append(idx.typedDecls, typedDecl{text: t, start: n.StartByte()})
			}
		}
	})
	return idx
}

// resolve decides whether a matched parameter key belongs to one of the
// rule's operations, trying direct containment, then a variable or
// typed declaration reaching a call, then a builder receiver.
func resolve(key *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens []string, idx fileIndex) resolution {
	if anchor := climbAnchor(key, spec, lang); anchor != nil {
		scope := nodeText(anchor, src)
		if spec.anchorStyle == anchorCall {
			scope, _ = splitCall(anchor, src) // only the callee counts
		}
		if containsAny(scope, tokens) {
			return resolution{true, "direct", firstLine(nodeText(anchor, src))}
		}
	}

	// Scoped to the enclosing function: a `params` here must not resolve
	// against a Stripe call in an unrelated function.
	scopeStart, scopeEnd := enclosingFunc(key, spec, lang)
	inScope := func(pos uint32) bool { return pos >= scopeStart && pos < scopeEnd }
	if name, _ := boundVariable(key, spec, lang, src); name != "" {
		for _, cs := range idx.callSites {
			if inScope(cs.start) && containsAny(cs.callee, tokens) && hasWord(cs.args, name) {
				return resolution{true, "var:" + name, firstLine(cs.callee + "(" + cs.args + ")")}
			}
		}
		for _, d := range idx.typedDecls {
			if inScope(d.start) && containsAny(d.text, tokens) && hasWord(d.text, name) {
				return resolution{true, "type:" + name, firstLine(d.text)}
			}
		}
	}

	// Receiver indirection: paramsBuilder.addPaymentMethodType(...) where
	// paramsBuilder was declared with a params type earlier in the file.
	if anchor := climbAnchor(key, spec, lang); anchor != nil {
		if recv := anchor.NamedChild(0); recv != nil {
			rname := strings.TrimSpace(nodeText(recv, src))
			for _, d := range idx.typedDecls {
				if inScope(d.start) && containsAny(d.text, tokens) && hasWord(d.text, rname) {
					return resolution{true, "recv:" + rname, firstLine(d.text)}
				}
			}
		}
	}
	return resolution{}
}

// enclosingFunc returns the byte span of the key's enclosing function body,
// or the whole file when the key is at top level.
func enclosingFunc(key *ts.Node, spec langSpec, lang *ts.Language) (uint32, uint32) {
	for cur := key.Parent(); cur != nil; cur = cur.Parent() {
		if containsStr(spec.funcKinds, cur.Type(lang)) {
			return cur.StartByte(), cur.EndByte()
		}
	}
	return 0, ^uint32(0)
}

// climbAnchor finds the enclosing param-bag node — nearest by default,
// or (with climbOutermost) the root of a builder chain, stopping as
// soon as the current node isn't the *receiver* of its parent so it
// doesn't swallow an enclosing route handler.
func climbAnchor(n *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	var nearest *ts.Node
	for cur := n; cur != nil; cur = cur.Parent() {
		if containsStr(spec.anchorKinds, cur.Type(lang)) {
			nearest = cur
			break
		}
	}
	if nearest == nil || !spec.climbOutermost {
		return nearest
	}
	cur := nearest
	for {
		p := cur.Parent()
		if p == nil || !containsStr(spec.anchorKinds, p.Type(lang)) {
			return cur
		}
		if first := p.NamedChild(0); first == nil || !sameSpan(first, cur) {
			return cur
		}
		cur = p
	}
}

func sameSpan(a, b *ts.Node) bool {
	return a.StartByte() == b.StartByte() && a.EndByte() == b.EndByte()
}

// boundVariable returns the name of the variable a param bag is assigned
// to, skipping any node whose left-hand side is the matched key itself —
// in C# the parameter (PaymentMethodTypes = ...) is the same node kind
// as the binding (options = new(){...}).
func boundVariable(key *ts.Node, spec langSpec, lang *ts.Language, src []byte) (string, *ts.Node) {
	for cur := key; cur != nil; cur = cur.Parent() {
		if !containsStr(spec.assignKinds, cur.Type(lang)) {
			continue
		}
		lhs := cur.NamedChild(0)
		if lhs == nil || sameSpan(lhs, key) {
			continue
		}
		return strings.TrimSpace(nodeText(lhs, src)), cur
	}
	return "", nil
}

// splitCall separates a call node into its callee text and argument text,
// treating the last named child as the argument list.
func splitCall(n *ts.Node, src []byte) (callee, args string) {
	cnt := n.NamedChildCount()
	if cnt == 0 {
		return nodeText(n, src), ""
	}
	last := n.NamedChild(cnt - 1)
	if last == nil {
		return nodeText(n, src), ""
	}
	start, argStart, end := n.StartByte(), last.StartByte(), last.EndByte()
	if int(argStart) > len(src) || int(end) > len(src) || argStart < start {
		return nodeText(n, src), ""
	}
	return string(src[start:argStart]), string(src[argStart:end])
}

func walk(n *ts.Node, fn func(*ts.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children() {
		walk(c, fn)
	}
}

func containsStr(hay []string, s string) bool {
	for _, h := range hay {
		if h == s {
			return true
		}
	}
	return false
}

func containsAny(s string, tokens []string) bool {
	for _, t := range tokens {
		if t != "" && strings.Contains(s, t) {
			return true
		}
	}
	return false
}

// hasWord reports whether name appears in s as a whole identifier, so "params"
// does not match "paramsFoo".
func hasWord(s, name string) bool {
	if name == "" {
		return false
	}
	for i := 0; ; {
		j := strings.Index(s[i:], name)
		if j < 0 {
			return false
		}
		j += i
		beforeOK := j == 0 || !isIdentChar(s[j-1])
		after := j + len(name)
		afterOK := after >= len(s) || !isIdentChar(s[after])
		if beforeOK && afterOK {
			return true
		}
		i = j + 1
		if i >= len(s) {
			return false
		}
	}
}

func isIdentChar(b byte) bool {
	return b == '_' || b == '$' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
