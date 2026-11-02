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
	// assigns are every variable assignment in the file (not just ones
	// a rule resource cares about), for detecting reassignment — see
	// reassignedBetween.
	assigns []assignment
}

type typedDecl struct {
	typeText string // the declared type's own text, e.g. "PaymentIntentCreateOptions"
	name     string // the bound identifier's own text, e.g. "options"
	text     string // the whole declaration, for display only — see resolution.anchor
	start    uint32
	node     *ts.Node // the declaration itself, for scoping a lookup to its actual binding
}

type callSite struct {
	callee   string
	args     string
	argsNode *ts.Node // for structural argument checks, see argsBindVariable
	node     *ts.Node // the anchor itself, for shadow detection, see shadowed
	start    uint32   // for function-scope checks
}

type assignment struct {
	name  string
	start uint32
	end   uint32
	node  *ts.Node // for scoping a reassignment to the same binding, see reassignedBetween
}

func buildIndex(root *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens []string) fileIndex {
	var idx fileIndex
	walk(root, func(n *ts.Node) {
		kind := n.Type(lang)
		if containsStr(spec.anchorKinds, kind) {
			callee, args, argsNode := splitCall(n, lang, src)
			if calleeNamesResource(spec, callee, tokens) {
				idx.callSites = append(idx.callSites, callSite{callee: callee, args: args, argsNode: argsNode, node: n, start: n.StartByte()})
			}
		}
		if containsStr(spec.declKinds, kind) {
			if typeText, name, ok := declaredType(n, lang, src); ok && containsAny(typeText, tokens) {
				idx.typedDecls = append(idx.typedDecls, typedDecl{
					typeText: typeText,
					name:     name,
					text:     nodeText(n, src),
					start:    n.StartByte(),
					node:     n,
				})
			}
		}
		if containsStr(spec.assignKinds, kind) {
			if lhs := n.NamedChild(0); lhs != nil {
				idx.assigns = append(idx.assigns, assignment{
					name:  strings.TrimSpace(nodeText(lhs, src)),
					start: n.StartByte(),
					end:   n.EndByte(),
					node:  n,
				})
			}
		}
	})
	return idx
}

// declaredType extracts a declaration node's own declared type and
// bound identifier name by field position, rather than treating its
// whole text as fair game — so an untyped declaration whose
// initializer just happens to mention a resource token (e.g. TS's
// untyped `const intent = stripe.paymentIntents.create(...)`) isn't
// mistaken for a typed one. ok is false when n has no type field at
// all (TS's untyped declarators). TS's variable_declarator has both
// fields directly; C#'s variable_declaration has "type" directly but
// nests its bound identifier's own "name" field inside an unfielded
// variable_declarator child.
func declaredType(n *ts.Node, lang *ts.Language, src []byte) (typeText, name string, ok bool) {
	typeNode := fieldChild(n, lang, "type")
	if typeNode == nil {
		return "", "", false
	}
	nameNode := fieldChild(n, lang, "name")
	if nameNode == nil {
		for i := 0; i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c != nil {
				if nn := fieldChild(c, lang, "name"); nn != nil {
					nameNode = nn
					break
				}
			}
		}
	}
	if nameNode == nil {
		return "", "", false
	}
	return nodeText(typeNode, src), strings.TrimSpace(nodeText(nameNode, src)), true
}

// fieldChild returns n's own child with the given field name, if any.
func fieldChild(n *ts.Node, lang *ts.Language, field string) *ts.Node {
	for i := 0; i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i, lang) == field {
			return n.Child(i)
		}
	}
	return nil
}

// constructedTypeText returns anchor's own constructed-type text for
// anchorType SDKs (Go's composite_literal, C#'s
// object_creation_expression) — just the type name, not the whole
// initializer, so an unrelated property's VALUE (a string, a
// comment) that merely happens to contain a resource token can't
// falsely establish the operation — e.g. a C# `new UnrelatedOptions {
// Description = "PaymentIntentCreateOptions", ... }` must be judged
// on "UnrelatedOptions", not on text appearing anywhere inside it.
// Anchors with no "type" field (C#'s target-typed new(), which
// relies on its own declaration's type instead — see resolve's
// type: path; Java's builder-chain method_invocation, which has no
// single type-naming field at all) fall back to their own full text,
// unchanged.
func constructedTypeText(anchor *ts.Node, lang *ts.Language, src []byte) string {
	if t := fieldChild(anchor, lang, "type"); t != nil {
		return nodeText(t, src)
	}
	return nodeText(anchor, src)
}

// resolve decides whether a matched parameter key belongs to one of the
// rule's operations, trying direct containment, then a variable or
// typed declaration reaching a call, then a builder receiver. methods
// are the SDK method names (e.g. "create") consistent with the rule's
// operations, for anchorCall SDKs — see operationMatches. unambiguous
// is the subset of tokens that alone proves the operation — see
// langSpec.unambiguousTokens.
func resolve(key *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens, methods, unambiguous []string, idx fileIndex) resolution {
	anchor := climbAnchor(key, spec, lang)
	if anchor != nil {
		scope := constructedTypeText(anchor, lang, src)
		if spec.anchorStyle == anchorCall {
			scope, _, _ = splitCall(anchor, lang, src) // only the callee counts
		}
		switch {
		case len(unambiguous) > 0 && !containsAny(scope, unambiguous) && containsAny(scope, tokens):
			// Only an ambiguous token matched (e.g. Go's legacy
			// PaymentIntentParams, shared by create and update) — its
			// type name alone doesn't establish the operation; confirm
			// by finding the call that actually invokes it.
			if invokesCreateMethod(key, anchor, spec, lang, src, idx) {
				return resolution{true, "direct", firstLine(nodeText(anchor, src))}
			}
		case operationMatches(spec, scope, tokens, methods):
			return resolution{true, "direct", firstLine(nodeText(anchor, src))}
		}
	}

	// Scoped to the enclosing function: a `params` here must not resolve
	// against a Stripe call in an unrelated function.
	scopeStart, scopeEnd := enclosingFunc(key, spec, lang)
	inScope := func(pos uint32) bool { return pos >= scopeStart && pos < scopeEnd }
	if name, assign := boundVariable(key, spec, lang, src); name != "" {
		assignFn := enclosingFuncNode(assign, spec, lang)
		for _, cs := range idx.callSites {
			if !inScope(cs.start) || !operationMatches(spec, cs.callee, tokens, methods) || !argsBindVariable(cs.argsNode, src, name) {
				continue
			}
			if !reachedByAssignment(cs.node, cs.start, name, assign, assignFn, spec, lang, src, idx.assigns) {
				continue
			}
			return resolution{true, "var:" + name, firstLine(cs.callee + "(" + cs.args + ")")}
		}
		for _, d := range idx.typedDecls {
			if inScope(d.start) && d.name == name && sameFuncScope(enclosingFuncNode(d.node, spec, lang), assignFn) && sameLexicalBinding(assign, d.node, spec, lang) {
				return resolution{true, "type:" + name, firstLine(d.text)}
			}
		}
	}

	// Receiver indirection: paramsBuilder.addPaymentMethodType(...) where
	// paramsBuilder was declared with a params type earlier in the file.
	if anchor := climbAnchor(key, spec, lang); anchor != nil {
		if recv := anchor.NamedChild(0); recv != nil {
			rname := strings.TrimSpace(nodeText(recv, src))
			keyFn := enclosingFuncNode(key, spec, lang)
			for _, d := range idx.typedDecls {
				if inScope(d.start) && d.name == rname && sameFuncScope(enclosingFuncNode(d.node, spec, lang), keyFn) && declVisibleAt(d.node, anchor.StartByte(), spec, lang) {
					return resolution{true, "recv:" + rname, firstLine(d.text)}
				}
			}
		}
	}
	return resolution{}
}

// operationMatches reports whether callee both names one of the rule's
// resource tokens and, for anchorCall SDKs, invokes a method consistent
// with the rule's operations — e.g. a rule scoped to
// POST /v1/payment_intents must not match
// PaymentIntent.modify/construct_from, which target a different
// operation or make no request at all. anchorType SDKs (Go/Java/C#)
// already encode the operation in their resourceTokens (e.g.
// PaymentIntentCreateParams), so no separate check applies there. An
// empty methods (an operation shape expectedMethodNames doesn't model)
// also skips the check rather than guessing a wrong name.
func operationMatches(spec langSpec, callee string, tokens, methods []string) bool {
	if !calleeNamesResource(spec, callee, tokens) {
		return false
	}
	if spec.anchorStyle != anchorCall || len(methods) == 0 {
		return true
	}
	return containsStr(methods, calleeMethodName(callee))
}

// calleeNamesResource reports whether callee names one of the rule's
// resource tokens. For anchorCall SDKs, callee is a dotted/scoped
// selector chain (e.g. "Stripe::PaymentIntent.create",
// "client.v1.payment_intents.create") — a resource token must match a
// COMPLETE selector component, not merely appear as a substring, so
// an unrelated model sharing a resource name as a prefix (e.g.
// PaymentIntentAudit.create) doesn't match PaymentIntent. anchorType
// SDKs (Go/Java/C#) match against the whole constructed-type text
// instead (callee here is actually the initializer's own text, not a
// call chain), unchanged — see resolve's typed-anchor matching for
// narrowing that further to the declared type itself.
func calleeNamesResource(spec langSpec, callee string, tokens []string) bool {
	if spec.anchorStyle != anchorCall {
		return containsAny(callee, tokens)
	}
	for _, part := range calleeComponents(callee) {
		if containsStr(tokens, part) {
			return true
		}
	}
	return false
}

// calleeComponents splits a dotted/scoped selector chain into its
// individual identifiers — e.g. "Stripe::PaymentIntent.create" ->
// ["Stripe", "PaymentIntent", "create"], "\Stripe\SetupIntent::create"
// -> ["Stripe", "SetupIntent", "create"].
func calleeComponents(callee string) []string {
	normalized := strings.NewReplacer("->", ".", "::", ".", `\`, ".").Replace(strings.TrimSpace(callee))
	var parts []string
	for _, p := range strings.Split(normalized, ".") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// calleeMethodName returns the final method-name segment of a callee's
// text — the part after its last '.', '::', or '->' — e.g.
// "Stripe::PaymentIntent.create" -> "create",
// "client.v1.payment_intents.create" -> "create".
func calleeMethodName(callee string) string {
	normalized := strings.NewReplacer("->", ".", "::", ".").Replace(strings.TrimSpace(callee))
	parts := strings.Split(normalized, ".")
	return strings.TrimSpace(parts[len(parts)-1])
}

// goCreateMethod is stripe-go's create convention — distinct from the
// "create"/"create_async" spelling anchorCall SDKs use (expectedMethodNames).
const goCreateMethod = "New"

// invokesCreateMethod reports whether anchor — a composite literal
// naming an operation-ambiguous params type (e.g. Go's legacy,
// create-and-update-shared PaymentIntentParams) — is actually reached
// by a call to goCreateMethod, either directly as an inline call
// argument, or via the variable it's assigned to. idx.callSites can't
// answer this: it only holds anchorKinds nodes (composite literals,
// for Go), never the real call_expression that invokes them, so this
// does its own walk over every call_expression in the file instead.
func invokesCreateMethod(key, anchor *ts.Node, spec langSpec, lang *ts.Language, src []byte, idx fileIndex) bool {
	for cur := anchor.Parent(); cur != nil; cur = cur.Parent() {
		if containsStr(spec.anchorKinds, cur.Type(lang)) {
			break // don't climb past an enclosing, unrelated literal
		}
		if containsStr(spec.funcKinds, cur.Type(lang)) {
			break // don't climb out of the enclosing function
		}
		if cur.Type(lang) == "call_expression" {
			callee, _, _ := splitCall(cur, lang, src)
			return calleeMethodName(callee) == goCreateMethod
		}
	}

	name, assign := boundVariable(key, spec, lang, src)
	if name == "" {
		return false
	}
	assignFn := enclosingFuncNode(assign, spec, lang)
	scopeStart, scopeEnd := enclosingFunc(key, spec, lang)
	found := false
	walk(treeRoot(key), func(n *ts.Node) {
		if found || n.Type(lang) != "call_expression" {
			return
		}
		if n.StartByte() < scopeStart || n.StartByte() >= scopeEnd {
			return
		}
		callee, _, argsNode := splitCall(n, lang, src)
		if calleeMethodName(callee) != goCreateMethod || !argsBindVariable(argsNode, src, name) {
			return
		}
		if reachedByAssignment(n, n.StartByte(), name, assign, assignFn, spec, lang, src, idx.assigns) {
			found = true
		}
	})
	return found
}

// reachedByAssignment reports whether a candidate use site (node, at
// pos) is actually reached by assign's binding of name: in the same
// execution scope as the assignment, the use must run after it (not
// before — a later assignment can't retroactively change what an
// earlier use already saw; this ordering check doesn't apply across
// a function boundary, since a function's body executes when it's
// *called*, not at its textually fixed definition site); the
// binding mustn't be shadowed by a closer declaration in between
// (shadowed); it mustn't be overwritten by a clean reassignment
// before the use (reassignedBetween); and assign's own binding must
// actually still be in scope at pos at all (declVisibleAt) — e.g. a
// block-scoped declaration doesn't reach a use outside its block,
// independent of any of the other three checks.
func reachedByAssignment(node *ts.Node, pos uint32, name string, assign, assignFn *ts.Node, spec langSpec, lang *ts.Language, src []byte, assigns []assignment) bool {
	if sameFuncScope(enclosingFuncNode(node, spec, lang), assignFn) && pos < assign.EndByte() {
		return false
	}
	if shadowed(node, assignFn, spec, lang, src, name, assign, assigns) {
		return false
	}
	if reassignedBetween(assigns, name, assign, pos, assignFn, spec, lang) {
		return false
	}
	return declVisibleAt(assign, pos, spec, lang)
}

// treeRoot climbs n up to the root of its syntax tree.
func treeRoot(n *ts.Node) *ts.Node {
	for n.Parent() != nil {
		n = n.Parent()
	}
	return n
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

// enclosingFuncNode is enclosingFunc, returning the function node
// itself (or nil at top level) rather than its byte span — for walking
// back out through nested scopes, see shadowed. n may be
// nil (e.g. no assignment node was found), in which case there's no
// enclosing function to report.
func enclosingFuncNode(n *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	if n == nil {
		return nil
	}
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		if containsStr(spec.funcKinds, cur.Type(lang)) {
			return cur
		}
	}
	return nil
}

// shadowed reports whether name's outer binding — assign itself,
// whose own enclosing function is boundary (nil for module level) —
// is hidden from node by a closer declaration in between: a nested
// function's own parameter or local assignment (e.g.
// `function createIntent(params) { stripe.PaymentIntent.create(params) }`
// shadows an outer, module-level params object), or a block-scoped
// let/const (JS/TS) in some enclosing {} block, bare or not — so a
// reference at node doesn't reach the outer binding at all. Climbing
// stops at boundary, since that scope's own declaration, if it names
// name, is exactly what the outer assignment itself binds.
func shadowed(node, boundary *ts.Node, spec langSpec, lang *ts.Language, src []byte, name string, assign *ts.Node, assigns []assignment) bool {
	for cur := node.Parent(); cur != nil; cur = cur.Parent() {
		if boundary != nil && sameSpan(cur, boundary) {
			return false
		}
		kind := cur.Type(lang)
		switch {
		case containsStr(spec.funcKinds, kind):
			if declaresParam(cur, src, name) {
				return true
			}
			if localAssignmentWithin(assigns, name, cur, node.StartByte(), assign, spec, lang) {
				return true
			}
		case containsStr(spec.blockKinds, kind):
			if blockScopedShadow(assigns, name, cur, node.StartByte(), assign, spec, lang) {
				return true
			}
		}
	}
	return false
}

// localAssignmentWithin reports whether name has a visible (see
// visibleAt) assignment directly within fn itself — not some separate
// nested helper function merely defined inside fn's byte range, whose
// own locals never reach fn's own body at all — at or before pos,
// other than assign itself: a local declaration establishing its own
// binding for name within fn (e.g. `let params = {}` in its body),
// distinct from any same-named binding outside it, even though
// declaresParam (formal parameters only) wouldn't catch it. A plain
// write to an existing binding (see writeKinds) doesn't count: it
// writes through to whatever params resolution already finds — an
// outer, captured variable for a closure that never declared its own
// — not a new local shadowing it. Checked before the enclosing-
// function comparison, which costs an ancestor climb this one avoids
// needing at all.
func localAssignmentWithin(assigns []assignment, name string, fn *ts.Node, pos uint32, assign *ts.Node, spec langSpec, lang *ts.Language) bool {
	for _, a := range assigns {
		if a.name != name || a.start < fn.StartByte() || a.start >= fn.EndByte() || sameSpan(a.node, assign) {
			continue
		}
		if containsStr(spec.writeKinds, a.node.Type(lang)) {
			continue
		}
		if !sameFuncScope(enclosingFuncNode(a.node, spec, lang), fn) {
			continue
		}
		if visibleAt(a, pos, spec, lang) {
			return true
		}
	}
	return false
}

// visibleAt reports whether assignment a's binding is visible at pos.
// A block-scoped let/const (JS/TS) is only visible within its own
// nearest enclosing block (see langSpec.blockKinds); any other
// assignment kind — var, bare assignment, Python/Ruby's plain
// assignment — is visible throughout its enclosing function once
// declared.
func visibleAt(a assignment, pos uint32, spec langSpec, lang *ts.Language) bool {
	if a.start > pos {
		return false
	}
	if p := a.node.Parent(); p != nil && p.Type(lang) == "lexical_declaration" {
		block := nearestBlock(a.node, spec, lang)
		return block != nil && pos >= block.StartByte() && pos < block.EndByte()
	}
	return true
}

// blockScopedShadow reports whether name is given its own block-
// confined binding (see bindingBlock — JS/TS's let/const, Go's
// var_spec/short_var_declaration) whose nearest enclosing block is
// exactly block (not some deeper-nested block merely contained
// within its byte range), at or before pos, other than assign itself
// — a new, block-scoped binding distinct from any outer variable of
// the same name (a bare var/plain reassignment inside the same
// block, by contrast, still refers to the outer one, and is handled
// by reassignedBetween instead, not treated as shadowing here).
func blockScopedShadow(assigns []assignment, name string, block *ts.Node, pos uint32, assign *ts.Node, spec langSpec, lang *ts.Language) bool {
	for _, a := range assigns {
		if a.name != name || a.start > pos || sameSpan(a.node, assign) {
			continue
		}
		ab := bindingBlock(a.node, spec, lang)
		if ab == nil || !sameSpan(ab, block) {
			continue
		}
		return true
	}
	return false
}

// declaresParam reports whether fn's own parameter list — approximated
// as its text up to its body, the last named child — names name.
func declaresParam(fn *ts.Node, src []byte, name string) bool {
	n := fn.NamedChildCount()
	if n == 0 {
		return false
	}
	body := fn.NamedChild(n - 1)
	if body == nil || body.StartByte() > uint32(len(src)) {
		return false
	}
	header := string(src[fn.StartByte():body.StartByte()])
	return hasWord(header, name)
}

// climbAnchor finds the enclosing param-bag node — nearest by default,
// or (with climbOutermost) the root of a builder chain, stopping as
// soon as the current node isn't the *receiver* of its parent so it
// doesn't swallow an enclosing route handler. The climb also stops at
// any funcKinds boundary before reaching an anchor: a bag built
// inside a callback or do...end block — Ruby's
// `Stripe::PaymentIntent.create(...) do |x| options = {...} end`, or
// JS's `create(params, function(err, pi) { var local = {...}; })` —
// sits in its own separate function scope, not as one of the call's
// own request arguments, even though the call node still technically
// encloses it (Ruby's do...end attaches to the call via its own
// "block" field; JS's callback is itself one of the call's
// arguments, but its *body* is not).
func climbAnchor(n *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	var nearest *ts.Node
	for cur := n; cur != nil; cur = cur.Parent() {
		if containsStr(spec.anchorKinds, cur.Type(lang)) {
			nearest = cur
			break
		}
		if cur != n && containsStr(spec.funcKinds, cur.Type(lang)) {
			return nil
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

// splitCall separates a call node into its callee text, its argument
// text, and the argument-list node itself (for structural checks, see
// argsBindVariable) — found via the call's own "arguments" field
// (never assumed to be the last named child, which breaks for e.g.
// Ruby's `foo(...) do ... end`, where the trailing do_block — field
// "block" — is the last named child, not the real argument_list) and
// a terminal name field ("function" for Python/JS/TS/Go/PHP's plain
// calls, "method" for Ruby, "name" for PHP/Java's member/scoped
// calls) whose own end byte bounds the callee text — excluding e.g. a
// comment sitting between the name and the argument list, which the
// old "callee is everything before the args" slice would have
// swallowed.
func splitCall(n *ts.Node, lang *ts.Language, src []byte) (callee, args string, argsNode *ts.Node) {
	argsNode = fieldChild(n, lang, "arguments")
	nameNode := fieldChild(n, lang, "function")
	if nameNode == nil {
		nameNode = fieldChild(n, lang, "method")
	}
	if nameNode == nil {
		nameNode = fieldChild(n, lang, "name")
	}
	if argsNode != nil && nameNode != nil {
		start, calleeEnd := n.StartByte(), nameNode.EndByte()
		argStart, argEnd := argsNode.StartByte(), argsNode.EndByte()
		if calleeEnd <= uint32(len(src)) && argEnd <= uint32(len(src)) && calleeEnd >= start && argStart >= calleeEnd {
			return string(src[start:calleeEnd]), string(src[argStart:argEnd]), argsNode
		}
	}
	// Fallback for node shapes without those fields (e.g. C#'s
	// implicit_object_creation_expression) — the same "last named
	// child is the argument/initializer list" heuristic as before.
	cnt := n.NamedChildCount()
	if cnt == 0 {
		return nodeText(n, src), "", nil
	}
	last := n.NamedChild(cnt - 1)
	if last == nil {
		return nodeText(n, src), "", nil
	}
	start, argStart, end := n.StartByte(), last.StartByte(), last.EndByte()
	if argStart > uint32(len(src)) || end > uint32(len(src)) || argStart < start {
		return nodeText(n, src), "", nil
	}
	return string(src[start:argStart]), string(src[argStart:end]), last
}

// argsBindVariable reports whether name is passed to a call as its
// entire top-level param bag — directly (create(params)),
// spread/unpacked (create(**params), create(...params)), via the SDK's
// reserved request-envelope keyword (create(params=params) /
// create(params: params)), or spread into a top-level object/array
// literal argument (create({ ...params, amount: 1000 })) — rather than
// merely appearing as the value of some unrelated argument, e.g.
// create(metadata=params), where name ends up nested under metadata,
// not merged into the call's own parameters.
func argsBindVariable(argsNode *ts.Node, src []byte, name string) bool {
	if argsNode == nil || name == "" {
		return false
	}
	for i := 0; i < argsNode.NamedChildCount(); i++ {
		arg := argsNode.NamedChild(i)
		if arg == nil {
			continue
		}
		if stripSpread(nodeText(arg, src)) == name {
			return true
		}
		// The SDK's reserved request-envelope keyword: params=params /
		// params: params (Python/Ruby's StripeClient style) binds the
		// whole bag the same way a bare positional/spread argument does.
		if arg.NamedChildCount() == 2 {
			k := strings.TrimSpace(nodeText(arg.NamedChild(0), src))
			v := strings.TrimSpace(nodeText(arg.NamedChild(1), src))
			if k == "params" && v == name {
				return true
			}
		}
		// A top-level argument that's itself an object/array/hash
		// literal containing a spread of name, e.g.
		// create({ ...params, amount: 1000 }) — object merging, not
		// argument spreading, so it isn't caught by the bare check
		// above; a nested property whose value happens to equal name
		// (not itself a spread) still doesn't count.
		if spreadsVariable(arg, src, name) {
			return true
		}
	}
	return false
}

// spreadsVariable reports whether container (an object/array/hash
// literal passed as a call argument) spreads name at its own top
// level, e.g. { ...params, amount: 1000 } — not merely containing name
// as some unrelated property's value.
func spreadsVariable(container *ts.Node, src []byte, name string) bool {
	if container == nil {
		return false
	}
	for i := 0; i < container.NamedChildCount(); i++ {
		c := container.NamedChild(i)
		if c == nil {
			continue
		}
		t := nodeText(c, src)
		if stripped := stripSpread(t); stripped != strings.TrimSpace(t) && stripped == name {
			return true
		}
	}
	return false
}

// stripSpread removes a leading spread/splat marker ("**", "...", or
// "*") from s, if present, along with surrounding whitespace.
func stripSpread(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"**", "...", "*"} {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimSpace(s[len(prefix):])
		}
	}
	return s
}

// reassignedBetween reports whether name is assigned again to the same
// binding as assign — i.e. in the same enclosing function (assignFn,
// nil for module level) — strictly after assign's own end and
// strictly before to. E.g. params reassigned to a clean bag between
// the matched bag's own assignment and the call reached by it means
// that bag was never actually sent. A same-named reassignment inside
// some other, unrelated nested function is a different binding
// (shadowing) and doesn't count — mirrors shadowed's reasoning for
// the call side.
func reassignedBetween(assigns []assignment, name string, assign *ts.Node, to uint32, assignFn *ts.Node, spec langSpec, lang *ts.Language) bool {
	from := assign.EndByte()
	for _, a := range assigns {
		if a.name != name || a.start <= from || a.start >= to {
			continue
		}
		if !sameFuncScope(enclosingFuncNode(a.node, spec, lang), assignFn) {
			continue
		}
		// Only an *unconditional* reassignment counts as definitely
		// overwriting assign — one that's a sibling statement at the
		// same nesting level, not nested inside an if/loop/try/etc.
		// that might not execute (e.g. `if use_defaults: params = {}`
		// doesn't overwrite params when use_defaults is false, so the
		// original bag can still reach the call).
		if unconditionalSibling(a.node, assign, spec, lang) {
			return true
		}
	}
	return false
}

// unconditionalSibling reports whether a and b are direct children of
// the same statement list — i.e. a is not nested inside any
// conditional/loop/try construct relative to b's own nesting level.
// Grammars that wrap every statement in its own per-kind node (JS's
// lexical_declaration/expression_statement, PHP's expression_statement)
// need climbing to the shared blockKinds container first — comparing
// Parent() directly would compare two different per-statement wrapper
// nodes, which never match even for true siblings. Grammars without
// blockKinds configured (Python/Ruby/Go/Java/C#) don't wrap statements
// this way, so comparing Parent() directly already works.
func unconditionalSibling(a, b *ts.Node, spec langSpec, lang *ts.Language) bool {
	if len(spec.blockKinds) == 0 {
		ap, bp := a.Parent(), b.Parent()
		return ap != nil && bp != nil && sameSpan(ap, bp)
	}
	ba, bb := nearestBlock(a, spec, lang), nearestBlock(b, spec, lang)
	if ba == nil || bb == nil || !sameSpan(ba, bb) {
		return false
	}
	return !nestedInControlFlow(a, ba, spec, lang) && !nestedInControlFlow(b, bb, spec, lang)
}

// nestedInControlFlow reports whether node sits inside a conditional
// or loop body relative to block (its own nearestBlock) — true even
// when that body has no braces of its own, e.g. PHP/JS's single-
// statement `if (cond) $x = 1;`: the assignment is a direct child of
// the if_statement itself, which is in turn a direct child of the
// very same enclosing block as a true, unconditional sibling
// statement — nearestBlock alone can't distinguish the two, since
// there's no intervening compound_statement/statement_block to catch
// via blockKinds. Climbing through controlKinds between node and
// block can.
func nestedInControlFlow(node, block *ts.Node, spec langSpec, lang *ts.Language) bool {
	for cur := node.Parent(); cur != nil && !sameSpan(cur, block); cur = cur.Parent() {
		if containsStr(spec.controlKinds, cur.Type(lang)) {
			return true
		}
	}
	return false
}

// nearestBlock returns n's nearest ancestor whose type is a
// statement-list container (see langSpec.blockKinds).
func nearestBlock(n *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		if containsStr(spec.blockKinds, cur.Type(lang)) {
			return cur
		}
	}
	return nil
}

// bindingBlock returns the block a declaration node is confined to,
// for a declaration kind that never escapes its own block: Go's
// var_spec/short_var_declaration (Go has no hoisting — every local
// declaration is scoped to the block it's written in, unlike a plain
// `=` reassignment, which binds whatever name resolution already
// finds) or JS/TS's let/const (variable_declarator whose parent is
// lexical_declaration). Nil means node isn't block-confined at all —
// it's visible throughout its enclosing function once declared (JS/TS
// var, Go's plain reassignment, Python/Ruby's assignment).
func bindingBlock(node *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	switch node.Type(lang) {
	case "var_spec", "short_var_declaration":
		return nearestBlock(node, spec, lang)
	}
	if p := node.Parent(); p != nil && p.Type(lang) == "lexical_declaration" {
		return nearestBlock(node, spec, lang)
	}
	return nil
}

// declVisibleAt reports whether node's own binding reaches pos: true
// unless node is block-confined (see bindingBlock) and pos falls
// outside that block — e.g. a `params := ...` declared inside an
// if-branch doesn't reach a call after the branch closes, even when
// nothing else (ordering, shadowing, reassignment) rejects the match.
func declVisibleAt(node *ts.Node, pos uint32, spec langSpec, lang *ts.Language) bool {
	block := bindingBlock(node, spec, lang)
	return block == nil || (pos >= block.StartByte() && pos < block.EndByte())
}

// sameLexicalBinding reports whether a and b could be the very same
// variable binding: each must be visible at the other's own
// position (see declVisibleAt) — checked both ways, since either one
// could be the block-confined one. Two same-named declarations where
// one is confined to a block that doesn't contain the other are
// different bindings even when nothing else distinguishes them —
// e.g. an outer, function-scoped bag and an unrelated inner block's
// own re-declaration of the same name, where being in the same
// enclosing function alone doesn't establish they're the same
// variable.
func sameLexicalBinding(a, b *ts.Node, spec langSpec, lang *ts.Language) bool {
	return declVisibleAt(a, b.StartByte(), spec, lang) && declVisibleAt(b, a.StartByte(), spec, lang)
}

// sameFuncScope reports whether a and b are the same enclosing
// function — including both being nil (module level).
func sameFuncScope(a, b *ts.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	return sameSpan(a, b)
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
