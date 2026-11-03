package scan

import (
	"sort"
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
	// Built from every rule resource's tokens at once, so lookups must
	// still check typeText against their own match's tokens.
	typedDecls []typedDecl
	// assigns are every variable assignment in the file (not just ones
	// a rule resource cares about), for detecting reassignment — see
	// reassignedBetween.
	assigns []assignment
	// deletions are field removals from a variable, for deletedBefore.
	deletions []deletion
}

// deletion is a field removal from a variable's bag: JS's
// `delete params.key` / `delete params['key']`, Python's
// `del params['key']`. node is the statement-level node (the
// unary_expression or delete_statement).
type deletion struct {
	name, key string
	node      *ts.Node
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
		idx.deletions = append(idx.deletions, deletionsAt(n, lang, src)...)
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
// A Java builder chain names its type only at its root receiver —
// `PaymentIntentCreateParams.builder()` or `new X.Builder()` — so the
// chain is followed through its "object" fields, never its
// arguments. An anchor with no type of its own (C#'s target-typed
// `new()`) returns "": its type lives on the declaration, which the
// type: path resolves.
func constructedTypeText(anchor *ts.Node, spec langSpec, lang *ts.Language, src []byte) string {
	cur := anchor
	for containsStr(spec.anchorKinds, cur.Type(lang)) {
		if t := fieldChild(cur, lang, "type"); t != nil {
			return nodeText(t, src)
		}
		obj := fieldChild(cur, lang, "object")
		if obj == nil {
			return ""
		}
		cur = obj
	}
	if t := fieldChild(cur, lang, "type"); t != nil {
		return nodeText(t, src)
	}
	return nodeText(cur, src)
}

// maxMergeHops bounds how many intermediate merge assignments
// (`params = {...base}`) resolveVar follows from one bag.
const maxMergeHops = 4

// resolveVar resolves a bag bound to name by assign: a Stripe call it
// reaches (var:), a typed declaration of the same binding (type:), or —
// following assignments that merge it into another variable, e.g.
// `const params = {...base, amount: 1000}` — either of those for that
// variable. A call is skipped when the matched key is unconditionally
// deleted from the bag before it (see deletedBefore). binding, when
// non-nil, is the variable assign writes; nil derives it (outerBinding).
// For an operation-ambiguous type (unambiguous non-empty: Go's legacy
// PaymentIntentParams), only a call site naming the operation-specific
// type counts here — a composite literal like
// []*stripe.PaymentIntentParams{params} is not an operation, and the
// create call itself is proven by goCreateReached instead.
func resolveVar(key *ts.Node, name string, assign, binding *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens, methods, unambiguous []string, idx fileIndex, hops int) resolution {
	// A write into a captured outer variable (a Ruby block's
	// assignment, a JS/Go closure's plain write) reaches calls
	// throughout that variable's own scope, not just the write's.
	if binding == nil {
		binding = outerBinding(name, assign, spec, lang, idx.assigns)
	}
	assignFn := enclosingFuncNode(binding, spec, lang)
	varInScope := funcScope(assignFn)
	keyText := strings.Trim(nodeText(key, src), `"':`)
	reaches := func(use *ts.Node, pos uint32) bool {
		return reachedByAssignment(use, pos, name, assign, binding, assignFn, spec, lang, src, idx.assigns) &&
			!deletedBefore(idx.deletions, name, keyText, assign, binding, assignFn, use, spec, lang, src, idx.assigns)
	}
	for _, cs := range idx.callSites {
		if !varInScope(cs.start) || !operationMatches(spec, cs.callee, tokens, methods) || !argsBindVariable(cs.argsNode, lang, src, name) {
			continue
		}
		if len(unambiguous) > 0 && !containsAny(cs.callee, unambiguous) {
			continue
		}
		if reaches(cs.node, cs.start) {
			return resolution{true, "var:" + name, firstLine(cs.callee + "(" + cs.args + ")")}
		}
	}
	for _, d := range idx.typedDecls {
		if varInScope(d.start) && d.name == name && containsAny(d.typeText, tokens) && sameFuncScope(enclosingFuncNode(d.node, spec, lang), assignFn) && sameLexicalBinding(binding, d.node, spec, lang) {
			return resolution{true, "type:" + name, firstLine(d.text)}
		}
	}
	if hops >= maxMergeHops {
		return resolution{}
	}
	for _, a := range idx.assigns {
		if a.name == name || a.start <= assign.EndByte() || !varInScope(a.start) {
			continue
		}
		rhs := assignRHS(a.node, lang)
		if rhs == nil || !mergesVariable(rhs, name, lang, src) || !reaches(a.node, a.start) {
			continue
		}
		if res := resolveVar(key, a.name, a.node, nil, spec, lang, src, tokens, methods, unambiguous, idx, hops+1); res.confirmed {
			res.via = "var:" + name + ">" + strings.TrimPrefix(res.via, "var:")
			return res
		}
	}
	return resolution{}
}

// deletedBefore reports whether the matched key is removed from name's
// bag — `delete params.key` / `del params['key']`, see deletion —
// after assign and unconditionally before use: a sibling of assign (so
// every path through assign runs it) or dominating use, referring to
// the same binding (not shadowed).
func deletedBefore(dels []deletion, name, key string, assign, binding, assignFn, use *ts.Node, spec langSpec, lang *ts.Language, src []byte, assigns []assignment) bool {
	for _, d := range dels {
		if d.name != name || d.key != key || d.node.StartByte() <= assign.EndByte() || d.node.StartByte() >= use.StartByte() {
			continue
		}
		if !unconditionalSibling(d.node, assign, spec, lang) && !overwritesBefore(d.node, use, spec, lang) {
			continue
		}
		if !shadowed(d.node, assignFn, spec, lang, src, name, binding, assigns) {
			return true
		}
	}
	return false
}

// chainRoot follows a call chain's "object" fields from anchor down to
// its base receiver — the identifier or constructor the chain starts from.
func chainRoot(anchor *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	cur := anchor
	for containsStr(spec.anchorKinds, cur.Type(lang)) {
		obj := fieldChild(cur, lang, "object")
		if obj == nil {
			break
		}
		cur = obj
	}
	return cur
}

// resolve decides whether a matched parameter key belongs to one of the
// rule's operations, trying direct containment, then a variable or
// typed declaration reaching a call, then a builder receiver. methods
// are the SDK method names (e.g. "create") consistent with the rule's
// operations, for anchorCall SDKs — see operationMatches. unambiguous
// is the subset of tokens that alone proves the operation — see
// langSpec.unambiguousTokens.
func resolve(key *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens, methods, unambiguous []string, idx fileIndex) resolution {
	if recv, stmt, _, ok := fieldWrite(key, lang, src); ok {
		return resolveFieldWrite(recv, stmt, spec, lang, src, tokens, unambiguous, idx)
	}
	anchor := climbAnchor(key, spec, lang, src)
	if anchor != nil {
		scope := constructedTypeText(anchor, spec, lang, src)
		if spec.anchorStyle == anchorCall {
			scope, _, _ = splitCall(anchor, lang, src) // only the callee counts
		}
		switch {
		case len(unambiguous) > 0 && !containsAny(scope, unambiguous) && containsAny(scope, tokens):
			// Only an ambiguous token matched (e.g. Go's legacy
			// PaymentIntentParams, shared by create and update) — its
			// type name alone doesn't establish the operation; confirm
			// by finding the call that actually invokes it.
			if invokesCreateMethod(key, anchor, spec, lang, src, unambiguous, idx) {
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
		if res := resolveVar(key, name, assign, nil, spec, lang, src, tokens, methods, unambiguous, idx, 0); res.confirmed {
			return res
		}
	}
	// An in-place merge — Object.assign(params, {...}), params.merge!(...),
	// params.update(...) — writes the key into params itself.
	if name, call := inPlaceMerge(key, spec, lang, src); name != "" {
		var binding *ts.Node // nil when there's no prior assignment (e.g. a parameter): derived from call
		if prior := priorBindings(name, call, spec, lang, idx.assigns); len(prior) > 0 {
			binding = outerBinding(name, prior[0], spec, lang, idx.assigns)
		}
		if res := resolveVar(key, name, call, binding, spec, lang, src, tokens, methods, unambiguous, idx, 0); res.confirmed {
			return res
		}
	}

	// Receiver indirection: paramsBuilder.addPaymentMethodType(...) where
	// paramsBuilder was declared with a params type earlier in the file.
	if anchor := climbAnchor(key, spec, lang, src); anchor != nil {
		recv := anchor.NamedChild(0)
		if spec.climbOutermost {
			// A builder chain's receiver is its base, not the inner chain:
			// b in b.addPaymentMethodType("card").setAmount(1000L).
			recv = chainRoot(anchor, spec, lang)
		}
		if recv != nil {
			rname := strings.TrimSpace(nodeText(recv, src))
			keyFn := enclosingFuncNode(key, spec, lang)
			for _, d := range idx.typedDecls {
				if inScope(d.start) && d.name == rname && containsAny(d.typeText, tokens) && sameFuncScope(enclosingFuncNode(d.node, spec, lang), keyFn) && declVisibleAt(d.node, anchor.StartByte(), spec, lang) {
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
func invokesCreateMethod(key, anchor *ts.Node, spec langSpec, lang *ts.Language, src []byte, unambiguous []string, idx fileIndex) bool {
	for cur := parentOf(anchor); cur != nil; cur = parentOf(cur) {
		if containsStr(spec.anchorKinds, cur.Type(lang)) {
			break // don't climb past an enclosing, unrelated literal
		}
		if containsStr(spec.funcKinds, cur.Type(lang)) {
			break // don't climb out of the enclosing function
		}
		if cur.Type(lang) == "call_expression" {
			callee, _, _ := splitCall(cur, lang, src)
			return goCreatesResource(callee, unambiguous)
		}
	}

	name, assign := boundVariable(key, spec, lang, src)
	if name == "" {
		return false
	}
	return goCreateReached(name, assign, outerBinding(name, assign, spec, lang, idx.assigns), spec, lang, src, unambiguous, idx)
}

// goCreateReached reports whether a stripe-go create call for one of
// unambiguous's resources (see goCreatesResource) is passed name and
// reached by assign's value of it, binding being the variable assign
// writes. idx.callSites can't answer this — it only holds Go's
// composite literals — so this walks every call_expression instead.
func goCreateReached(name string, assign, binding *ts.Node, spec langSpec, lang *ts.Language, src []byte, unambiguous []string, idx fileIndex) bool {
	assignFn := enclosingFuncNode(binding, spec, lang)
	inScope := funcScope(assignFn)
	found := false
	walk(treeRoot(assign), func(n *ts.Node) {
		if found || n.Type(lang) != "call_expression" || !inScope(n.StartByte()) {
			return
		}
		callee, _, argsNode := splitCall(n, lang, src)
		if !goCreatesResource(callee, unambiguous) || !argsBindVariable(argsNode, lang, src, name) {
			return
		}
		if reachedByAssignment(n, n.StartByte(), name, assign, binding, assignFn, spec, lang, src, idx.assigns) {
			found = true
		}
	})
	return found
}

// fieldWrite recognizes key as the field name of a field assignment's
// left-hand side — Go's `params.PaymentMethodTypes = ...` — returning
// the receiver variable's node, the assignment statement, and the
// intermediate field path (["PaymentSettings"] for
// `params.PaymentSettings.PaymentMethodTypes = ...`).
func fieldWrite(key *ts.Node, lang *ts.Language, src []byte) (recv, stmt *ts.Node, path []string, ok bool) {
	sel := parentOf(key)
	if sel == nil || sel.Type(lang) != "selector_expression" {
		return nil, nil, nil, false
	}
	if f := fieldChild(sel, lang, "field"); f == nil || !sameSpan(f, key) {
		return nil, nil, nil, false
	}
	left := parentOf(sel)
	if left == nil || left.Type(lang) != "expression_list" {
		return nil, nil, nil, false
	}
	stmt = parentOf(left)
	if stmt == nil || stmt.Type(lang) != "assignment_statement" {
		return nil, nil, nil, false
	}
	if l := fieldChild(stmt, lang, "left"); l == nil || !sameSpan(l, left) {
		return nil, nil, nil, false
	}
	cur := fieldChild(sel, lang, "operand")
	for cur != nil && cur.Type(lang) == "selector_expression" {
		if f := fieldChild(cur, lang, "field"); f != nil {
			path = append([]string{nodeText(f, src)}, path...)
		}
		cur = fieldChild(cur, lang, "operand")
	}
	if cur == nil || cur.Type(lang) != "identifier" {
		return nil, nil, nil, false
	}
	return cur, stmt, path, true
}

// resolveFieldWrite resolves a field assignment (see fieldWrite) through
// its receiver's declaration: the latest assignment of the receiver
// still in scope before the write. Its declared or constructed type
// names the operation — and, for an operation-ambiguous type (Go's
// legacy PaymentIntentParams), a create call reached after the write
// must confirm it.
func resolveFieldWrite(recv, stmt *ts.Node, spec langSpec, lang *ts.Language, src []byte, tokens, unambiguous []string, idx fileIndex) resolution {
	name := nodeText(recv, src)
	// The binding is the variable's declaration; its type is the latest
	// one stated from there on — a later `params = defaults()` doesn't
	// change the declared `var params *stripe.PaymentIntentParams`.
	var decl *ts.Node
	typeText := ""
	for _, a := range priorBindings(name, stmt, spec, lang, idx.assigns) {
		if typeText == "" {
			typeText = goValueType(a, spec, lang, src)
		}
		if !containsStr(spec.writeKinds, a.Type(lang)) {
			decl = a
			break
		}
	}
	if decl == nil {
		return resolution{}
	}
	if !containsAny(typeText, tokens) {
		return resolution{}
	}
	if len(unambiguous) > 0 && !containsAny(typeText, unambiguous) && !goCreateReached(name, stmt, decl, spec, lang, src, unambiguous, idx) {
		return resolution{}
	}
	return resolution{true, "field:" + name, firstLine(nodeText(decl, src))}
}

// goValueType returns the type a Go declaration gives its variable: a
// `var params T` type, or the composite literal it's initialized from
// (`params := &stripe.PaymentIntentParams{}`).
func goValueType(decl *ts.Node, spec langSpec, lang *ts.Language, src []byte) string {
	if t := fieldChild(decl, lang, "type"); t != nil {
		return nodeText(t, src)
	}
	v := assignRHS(decl, lang)
	if v != nil && v.Type(lang) == "expression_list" {
		v = v.NamedChild(0)
	}
	if v != nil && v.Type(lang) == "unary_expression" {
		v = fieldChild(v, lang, "operand")
	}
	if v == nil || !containsStr(spec.anchorKinds, v.Type(lang)) {
		return ""
	}
	return constructedTypeText(v, spec, lang, src)
}

// goCreatesResource reports whether callee is stripe-go's create call
// for one of the resources named by unambiguous (e.g.
// PaymentIntentCreateParams -> PaymentIntent): goCreateMethod on the
// resource's own package (paymentintent.New) or on a client service
// (sc.V1PaymentIntents.New) — not an unrelated constructor that
// happens to be named New, like request.New.
func goCreatesResource(callee string, unambiguous []string) bool {
	parts := calleeComponents(callee)
	if len(parts) < 2 || parts[len(parts)-1] != goCreateMethod {
		return false
	}
	recv := parts[len(parts)-2]
	for _, tok := range unambiguous {
		base := strings.TrimSuffix(tok, "CreateParams")
		if recv == strings.ToLower(base) || strings.HasSuffix(recv, base+"s") {
			return true
		}
	}
	return false
}

// reachedByAssignment reports whether a candidate use site (node, at
// pos) is actually reached by assign's value of name. binding is the
// variable assign writes (assign itself unless it writes a captured
// outer variable — see outerBinding), and assignFn is binding's own
// function. In the same execution scope as assign (its own function,
// or binding's), the use must run after it — a later assignment can't
// retroactively change what an earlier use already saw; this ordering
// check doesn't apply across a function boundary, since a function's
// body executes when it's *called*, not at its textually fixed
// definition site. The binding mustn't be shadowed by a closer
// declaration in between (shadowed); assign's value mustn't be
// overwritten by a clean reassignment before the use
// (reassignedBetween); and the binding must still be in scope at pos
// (declVisibleAt) — e.g. a block-scoped declaration doesn't reach a
// use outside its block, independent of the other checks.
func reachedByAssignment(node *ts.Node, pos uint32, name string, assign, binding, assignFn *ts.Node, spec langSpec, lang *ts.Language, src []byte, assigns []assignment) bool {
	useFn := enclosingFuncNode(node, spec, lang)
	if pos < assign.EndByte() && (sameFuncScope(useFn, assignFn) || sameFuncScope(useFn, enclosingFuncNode(assign, spec, lang))) {
		return false
	}
	if shadowed(node, assignFn, spec, lang, src, name, binding, assigns) {
		return false
	}
	if reassignedBetween(assigns, name, assign, binding, node, pos, assignFn, spec, lang, src) {
		return false
	}
	return declVisibleAt(binding, pos, spec, lang)
}

// funcScope returns a membership test for fn's byte span — the whole
// file when fn is nil (module level).
func funcScope(fn *ts.Node) func(uint32) bool {
	if fn == nil {
		return func(uint32) bool { return true }
	}
	start, end := fn.StartByte(), fn.EndByte()
	return func(pos uint32) bool { return pos >= start && pos < end }
}

// outerBinding returns the assignment node whose variable assign
// actually writes: assign itself, unless assign — from inside a nested
// function — writes into a variable already bound by an enclosing scope
// (see writesOuterBinding: a Ruby block's assignment, a JS/Go plain
// write), in which case that enclosing binding, followed outward
// through further captures. A plain write that follows its own
// function's local declaration of the name stays local.
func outerBinding(name string, assign *ts.Node, spec langSpec, lang *ts.Language, assigns []assignment) *ts.Node {
	cur := assign
	for range assigns {
		curFn := enclosingFuncNode(cur, spec, lang)
		if d := localDeclFor(name, cur, curFn, spec, lang, assigns); d != nil {
			return d
		}
		if curFn == nil {
			return cur
		}
		var best *ts.Node
		for _, o := range assigns {
			if o.name != name || o.start >= cur.StartByte() || (o.start >= curFn.StartByte() && o.start < curFn.EndByte()) {
				continue
			}
			oFn := enclosingFuncNode(o.node, spec, lang)
			if oFn != nil && (curFn.StartByte() < oFn.StartByte() || curFn.EndByte() > oFn.EndByte()) {
				continue // o's function doesn't enclose cur
			}
			if !declVisibleAt(o.node, cur.StartByte(), spec, lang) {
				continue
			}
			w := assignment{name: name, start: cur.StartByte(), end: cur.EndByte(), node: cur}
			if !writesOuterBinding(w, o.node, oFn, spec, lang) {
				continue
			}
			if best == nil || o.start > best.StartByte() {
				best = o.node
			}
		}
		if best == nil {
			return cur
		}
		cur = best
	}
	return cur
}

// localDeclFor returns, for a plain write (see writeKinds), the
// declaration in its own function (fn, nil for module level) that it
// writes — the latest one still in scope at it — or nil when there is
// none (the write targets something captured from outside fn, or isn't
// a plain write at all).
func localDeclFor(name string, write, fn *ts.Node, spec langSpec, lang *ts.Language, assigns []assignment) *ts.Node {
	if !containsStr(spec.writeKinds, write.Type(lang)) {
		return nil
	}
	var best *ts.Node
	for _, a := range assigns {
		if a.name == name && a.start < write.StartByte() && !containsStr(spec.writeKinds, a.node.Type(lang)) &&
			sameFuncScope(enclosingFuncNode(a.node, spec, lang), fn) && declVisibleAt(a.node, write.StartByte(), spec, lang) &&
			(best == nil || a.start > best.StartByte()) {
			best = a.node
		}
	}
	return best
}

// priorBindings returns the assignments of name before at that are still
// in scope there — in at's own function or an enclosing one — latest
// first.
func priorBindings(name string, at *ts.Node, spec langSpec, lang *ts.Language, assigns []assignment) []*ts.Node {
	var out []*ts.Node
	for _, a := range assigns {
		if a.name != name || a.start >= at.StartByte() || !declVisibleAt(a.node, at.StartByte(), spec, lang) {
			continue
		}
		if fn := enclosingFuncNode(a.node, spec, lang); fn != nil && (at.StartByte() < fn.StartByte() || at.EndByte() > fn.EndByte()) {
			continue
		}
		out = append(out, a.node)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartByte() > out[j].StartByte() })
	return out
}

// inPlaceMergeMethods mutate their receiver: Ruby's Hash#merge!/update,
// Python's dict.update.
var inPlaceMergeMethods = map[string]bool{"merge!": true, "update": true, "deep_merge!": true, "reverse_merge!": true}

// inPlaceMerge recognizes key as an argument of a merge that mutates a
// variable in place — Object.assign(params, {...}) (its first argument),
// params.merge!(...) / params.update(...) (its receiver) — returning that
// variable's name and the merge call. Non-mutating merges around the key
// (dict(...), {...}.merge(...)) are climbed past, as in climbAnchor.
func inPlaceMerge(key *ts.Node, spec langSpec, lang *ts.Language, src []byte) (string, *ts.Node) {
	if spec.anchorStyle != anchorCall {
		return "", nil
	}
	for cur := parentOf(key); cur != nil; cur = parentOf(cur) {
		if containsStr(spec.funcKinds, cur.Type(lang)) {
			return "", nil
		}
		if !containsStr(spec.anchorKinds, cur.Type(lang)) {
			continue
		}
		callee, _, args := splitCall(cur, lang, src)
		if args == nil || key.StartByte() < args.StartByte() {
			return "", nil // the key is in the receiver, not the merged-in arguments
		}
		method, parts := calleeMethodName(callee), calleeComponents(callee)
		switch {
		case inPlaceMergeMethods[method] && len(parts) == 2:
			return parts[0], cur
		case strings.TrimSpace(callee) == "Object.assign":
			if first := args.NamedChild(0); first != nil && first.Type(lang) == "identifier" && key.StartByte() >= first.EndByte() {
				return nodeText(first, src), cur
			}
			return "", nil
		case mergeFuncs[method]:
			continue
		}
		return "", nil
	}
	return "", nil
}

// treeRoot climbs n up to the root of its syntax tree.
func treeRoot(n *ts.Node) *ts.Node {
	for parentOf(n) != nil {
		n = parentOf(n)
	}
	return n
}

// enclosingFunc returns the byte span of the key's enclosing function body,
// or the whole file when the key is at top level.
func enclosingFunc(key *ts.Node, spec langSpec, lang *ts.Language) (uint32, uint32) {
	for cur := parentOf(key); cur != nil; cur = parentOf(cur) {
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
	for cur := parentOf(n); cur != nil; cur = parentOf(cur) {
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
	for cur := parentOf(node); cur != nil; cur = parentOf(cur) {
		if boundary != nil && sameSpan(cur, boundary) {
			return false
		}
		kind := cur.Type(lang)
		switch {
		case containsStr(spec.funcKinds, kind):
			if declaresParam(cur, lang, src, name) {
				return true
			}
			if localAssignmentWithin(assigns, name, cur, node.StartByte(), assign, boundary, spec, lang) {
				return true
			}
		case containsStr(spec.blockKinds, kind), containsStr(spec.controlKinds, kind):
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
func localAssignmentWithin(assigns []assignment, name string, fn *ts.Node, pos uint32, assign, boundary *ts.Node, spec langSpec, lang *ts.Language) bool {
	for _, a := range assigns {
		if a.name != name || a.start < fn.StartByte() || a.start >= fn.EndByte() || sameSpan(a.node, assign) {
			continue
		}
		if writesOuterBinding(a, assign, boundary, spec, lang) {
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

// visibleAt reports whether assignment a's binding is visible at pos:
// declared at or before it and, for a block-confined declaration
// (JS/TS let/const, Go := and var — see bindingBlock), still inside
// its block.
func visibleAt(a assignment, pos uint32, spec langSpec, lang *ts.Language) bool {
	if a.start > pos {
		return false
	}
	return declVisibleAt(a.node, pos, spec, lang)
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
		if declaresTargetOf(a.node, assign, spec, lang) {
			continue
		}
		return true
	}
	return false
}

// declaresTargetOf reports whether decl is the very binding a plain
// write (see writeKinds) assigns into — e.g. `let params = {}` for a
// later `params = {...}` — rather than a different, shadowing one:
// decl precedes the write and is still in scope at it.
func declaresTargetOf(decl, write *ts.Node, spec langSpec, lang *ts.Language) bool {
	return containsStr(spec.writeKinds, write.Type(lang)) &&
		decl.StartByte() < write.StartByte() &&
		declVisibleAt(decl, write.StartByte(), spec, lang)
}

// declaresParam reports whether one of fn's own parameters binds name
// — via its "parameters" (or single-parameter "parameter") field where
// the grammar has one; otherwise the children before its body, the last
// named child. Comments are never part of it, so a body opening with
// a comment that mentions name doesn't read as a parameter.
func declaresParam(fn *ts.Node, lang *ts.Language, src []byte, name string) bool {
	if p := fieldChild(fn, lang, "parameter"); p != nil {
		return paramBinds(p, lang, src, name)
	}
	if ps := fieldChild(fn, lang, "parameters"); ps != nil {
		for i := 0; i < ps.NamedChildCount(); i++ {
			if c := ps.NamedChild(i); c != nil && !strings.Contains(c.Type(lang), "comment") && paramBinds(c, lang, src, name) {
				return true
			}
		}
		return false
	}
	n := fn.NamedChildCount()
	if n == 0 {
		return false
	}
	body := fn.NamedChild(n - 1)
	for i := 0; i < fn.ChildCount(); i++ {
		c := fn.Child(i)
		if c == nil || body == nil || c.StartByte() >= body.StartByte() {
			break
		}
		if !strings.Contains(c.Type(lang), "comment") && hasWord(nodeText(c, src), name) {
			return true
		}
	}
	return false
}

// paramBinds reports whether parameter node p binds name: p itself
// when it's a bare identifier, else its binding part — the "name",
// "pattern" or "left" fields (Python/Ruby default and keyword params,
// Go/Java/C#/PHP typed params, TS's required_parameter, JS's
// assignment_pattern), or its first named child (Python typed and
// splat params). Defaults and type annotations are never consulted,
// so `def f(options=params)` doesn't bind params.
func paramBinds(p *ts.Node, lang *ts.Language, src []byte, name string) bool {
	if p.NamedChildCount() == 0 {
		return stripSpread(nodeText(p, src)) == name
	}
	found := false
	for i := 0; i < p.ChildCount(); i++ {
		switch p.FieldNameForChild(i, lang) {
		case "name", "pattern", "left":
			found = true
			if hasWord(nodeText(p.Child(i), src), name) {
				return true
			}
		}
	}
	if found {
		return false
	}
	first := p.NamedChild(0)
	return first != nil && hasWord(nodeText(first, src), name)
}

// climbAnchor finds the enclosing param-bag node — nearest by default,
// or (with climbOutermost) the root of a builder chain, stopping as
// soon as the current node isn't the *receiver* of its parent so it
// doesn't swallow an enclosing route handler. A merge call (see
// mergeFuncs: {...}.merge(...), Object.assign(...), Python's dict(...))
// carries its keys into its result, so it's climbed past to the request
// it builds, not returned. The climb also stops at
// any funcKinds boundary before reaching an anchor: a bag built
// inside a callback or do...end block — Ruby's
// `Stripe::PaymentIntent.create(...) do |x| options = {...} end`, or
// JS's `create(params, function(err, pi) { var local = {...}; })` —
// sits in its own separate function scope, not as one of the call's
// own request arguments, even though the call node still technically
// encloses it (Ruby's do...end attaches to the call via its own
// "block" field; JS's callback is itself one of the call's
// arguments, but its *body* is not).
func climbAnchor(n *ts.Node, spec langSpec, lang *ts.Language, src []byte) *ts.Node {
	var nearest *ts.Node
	for cur := n; cur != nil; cur = parentOf(cur) {
		if containsStr(spec.anchorKinds, cur.Type(lang)) {
			if spec.anchorStyle == anchorCall {
				if callee, _, _ := splitCall(cur, lang, src); mergeFuncs[calleeMethodName(callee)] {
					continue // a merge carries the key into its result; keep climbing to the request
				}
			}
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
		p := parentOf(cur)
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
	for cur := key; cur != nil; cur = parentOf(cur) {
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
// passed through a merge (create(params.merge(amount: 1000)),
// create(**dict(params, amount=1000)); see mergesVariable) — rather
// than merely appearing as the value of some unrelated argument, e.g.
// create(metadata=params), where name ends up nested under metadata,
// not merged into the call's own parameters.
func argsBindVariable(argsNode *ts.Node, lang *ts.Language, src []byte, name string) bool {
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
		// The envelope's value may itself merge name in, e.g.
		// create(params={**params, 'amount': 1000}).
		if arg.NamedChildCount() == 2 {
			k := strings.TrimSpace(nodeText(arg.NamedChild(0), src))
			vNode := arg.NamedChild(1)
			if k == "params" && (strings.TrimSpace(nodeText(vNode, src)) == name || mergesVariable(vNode, name, lang, src)) {
				return true
			}
		}
		// A top-level argument that's itself an object/array/hash
		// literal containing a spread of name, e.g.
		// create({ ...params, amount: 1000 }) — object merging, not
		// argument spreading, so it isn't caught by the bare check
		// above; a nested property whose value happens to equal name
		// (not itself a spread) still doesn't count. A splatted
		// argument is judged by what it splats: **dict(params, ...).
		expr := arg
		if t := nodeText(arg, src); stripSpread(t) != strings.TrimSpace(t) && arg.NamedChildCount() == 1 {
			expr = arg.NamedChild(0)
		}
		if mergesVariable(expr, name, lang, src) {
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
// "*"), or Go's address-of "&" (New(&params) passes the same bag as
// New(params) with a pointer literal), from s, if present, along with
// surrounding whitespace.
func stripSpread(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"**", "...", "*", "&"} {
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
func reassignedBetween(assigns []assignment, name string, assign, binding, use *ts.Node, to uint32, assignFn *ts.Node, spec langSpec, lang *ts.Language, src []byte) bool {
	from := assign.EndByte()
	for _, a := range assigns {
		if a.name != name || a.start <= from || a.start >= to {
			continue
		}
		if selfMerge(a, lang, src) {
			continue
		}
		// From a nested scope, only a write into assign's own variable
		// (see writesOuterBinding) can overwrite it — and only when it
		// definitely runs before the use, in the same statement list.
		if !sameFuncScope(enclosingFuncNode(a.node, spec, lang), assignFn) {
			if writesOuterBinding(a, binding, assignFn, spec, lang) && overwritesBefore(a.node, use, spec, lang) {
				return true
			}
			continue
		}
		// Only an *unconditional* reassignment counts as definitely
		// overwriting assign — not one nested inside an if/loop/try/etc.
		// that might not execute (e.g. `if use_defaults: params = {}`
		// doesn't overwrite params when use_defaults is false, so the
		// original bag can still reach the call). Either it's a sibling
		// of assign, so every path through assign also runs it, or it
		// dominates the use, so every path to the use runs it — e.g. an
		// unconditional `params = {}` after an `if` that set the bag.
		if unconditionalSibling(a.node, assign, spec, lang) || overwritesBefore(a.node, use, spec, lang) {
			return true
		}
	}
	return false
}

// writesOuterBinding reports whether a, inside a scope nested within
// boundary (assign's own function, nil for module level), writes into
// assign's variable rather than declaring its own: a plain write (see
// writeKinds), or — where captureKinds applies — an assignment made
// after assign already bound the name, with only capture scopes (Ruby
// blocks, not `def`) in between.
func writesOuterBinding(a assignment, assign, boundary *ts.Node, spec langSpec, lang *ts.Language) bool {
	if containsStr(spec.writeKinds, a.node.Type(lang)) {
		return true
	}
	if len(spec.captureKinds) == 0 || a.start <= assign.StartByte() {
		return false
	}
	for cur := parentOf(a.node); cur != nil; cur = parentOf(cur) {
		if boundary != nil && sameSpan(cur, boundary) {
			break
		}
		kind := cur.Type(lang)
		if containsStr(spec.funcKinds, kind) && !containsStr(spec.captureKinds, kind) {
			return false
		}
	}
	return true
}

// overwritesBefore reports whether assignment node a unconditionally
// runs before use: a is a whole statement (see isWholeStatement) in a
// statement list that also contains, as a later statement, the one
// holding use.
func overwritesBefore(a, use *ts.Node, spec langSpec, lang *ts.Language) bool {
	container := parentOf(a)
	if len(spec.blockKinds) > 0 {
		container = nearestBlock(a, spec, lang)
	}
	if container == nil || !isWholeStatement(a, container) {
		return false
	}
	for cur := use; cur != nil && parentOf(cur) != nil; cur = parentOf(cur) {
		if sameSpan(parentOf(cur), container) {
			return cur.StartByte() > a.StartByte()
		}
	}
	return false
}

// selfMerge reports whether a's right-hand side structurally merges
// its own target variable back in — `params = {...params, amount: 1}`,
// `params = dict(params, amount=1)`, `params = params.merge(...)`,
// `params = params | {...}` — so the earlier bag's keys survive the
// reassignment. Any other right-hand side, even one that mentions the
// name (`params.except(:payment_method_types)`, `{description:
// 'params'}`), replaces the bag.
func selfMerge(a assignment, lang *ts.Language, src []byte) bool {
	rhs := assignRHS(a.node, lang)
	return rhs != nil && mergesVariable(rhs, a.name, lang, src)
}

// assignRHS returns an assignment node's right-hand side.
func assignRHS(n *ts.Node, lang *ts.Language) *ts.Node {
	if rhs := fieldChild(n, lang, "right"); rhs != nil {
		return rhs
	}
	return fieldChild(n, lang, "value")
}

// staticKey reports whether a deletion's key node names a field by its
// literal spelling — a dot property (x.key) or a plain string (x['key']) —
// rather than a computed expression (x[k], x[f"{k}"]) whose runtime value
// is unknown.
func staticKey(k *ts.Node, lang *ts.Language) bool {
	switch k.Type(lang) {
	case "property_identifier":
		return true
	case "string":
		for i := 0; i < k.NamedChildCount(); i++ {
			if c := k.NamedChild(i); c != nil && c.Type(lang) == "interpolation" {
				return false
			}
		}
		return true
	}
	return false
}

// deletionsAt returns the field removals n itself performs: JS/TS's
// `delete x.key` / `delete x['key']` (a unary_expression), Python's
// `del x['key']` (a delete_statement, possibly listing several).
func deletionsAt(n *ts.Node, lang *ts.Language, src []byte) []deletion {
	var targets []*ts.Node
	switch n.Type(lang) {
	case "unary_expression":
		if n.ChildCount() > 0 && nodeText(n.Child(0), src) == "delete" {
			if arg := fieldChild(n, lang, "argument"); arg != nil {
				targets = append(targets, arg)
			}
		}
	case "delete_statement":
		for i := 0; i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c != nil {
				targets = append(targets, c)
			}
		}
	}
	var out []deletion
	for _, t := range targets {
		obj, k := fieldChild(t, lang, "object"), fieldChild(t, lang, "property")
		if k == nil {
			k = fieldChild(t, lang, "index") // JS subscript_expression
		}
		if obj == nil {
			obj, k = fieldChild(t, lang, "value"), fieldChild(t, lang, "subscript") // Python subscript
		}
		if obj == nil || k == nil || !staticKey(k, lang) {
			continue
		}
		out = append(out, deletion{
			name: strings.TrimSpace(nodeText(obj, src)),
			key:  strings.Trim(strings.TrimSpace(nodeText(k, src)), `"'`),
			node: n,
		})
	}
	return out
}

// mergeFuncs are SDK-agnostic call names whose result carries forward
// every key of the receiver and arguments.
var mergeFuncs = map[string]bool{
	"merge": true, "merge!": true, "deep_merge": true, "reverse_merge": true, "with_defaults": true, // Ruby
	"dict":        true,                                                       // Python
	"assign":      true,                                                       // JS Object.assign
	"array_merge": true, "array_replace": true, "array_merge_recursive": true, // PHP
}

// mergesVariable reports whether expression n is a merge that includes
// name: a literal spreading it, a merge call on or over it, or a
// dict-union/array-union operator with it as an operand.
func mergesVariable(n *ts.Node, name string, lang *ts.Language, src []byte) bool {
	if spreadsVariable(n, src, name) {
		return true
	}
	if args := fieldChild(n, lang, "arguments"); args != nil {
		callee, _, _ := splitCall(n, lang, src)
		if !mergeFuncs[calleeMethodName(callee)] {
			return false
		}
		if parts := calleeComponents(callee); len(parts) >= 2 && parts[len(parts)-2] == name {
			return true
		}
		for i := 0; i < args.NamedChildCount(); i++ {
			if c := args.NamedChild(i); c != nil && stripSpread(nodeText(c, src)) == name {
				return true
			}
		}
		return false
	}
	if op := fieldChild(n, lang, "operator"); op != nil {
		if o := strings.TrimSpace(nodeText(op, src)); o == "|" || o == "+" {
			for _, side := range []string{"left", "right"} {
				if c := fieldChild(n, lang, side); c != nil && strings.TrimSpace(nodeText(c, src)) == name {
					return true
				}
			}
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
		ap, bp := parentOf(a), parentOf(b)
		return ap != nil && bp != nil && sameSpan(ap, bp)
	}
	ba, bb := nearestBlock(a, spec, lang), nearestBlock(b, spec, lang)
	if ba == nil || bb == nil || !sameSpan(ba, bb) {
		return false
	}
	return isWholeStatement(a, ba) && !nestedInControlFlow(b, bb, spec, lang)
}

// isWholeStatement reports whether assignment node a is an entire
// statement of container — a direct child, or the sole content of a
// per-statement wrapper (JS/PHP's expression_statement) that is — rather
// than part of a larger expression that may not evaluate it, like
// `useDefaults && (params = {})`, or the body of a brace-less if.
func isWholeStatement(a, container *ts.Node) bool {
	p := parentOf(a)
	if p == nil {
		return false
	}
	if sameSpan(p, container) {
		return true
	}
	gp := parentOf(p)
	return gp != nil && sameSpan(gp, container) && p.NamedChildCount() == 1
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
	for cur := parentOf(node); cur != nil && !sameSpan(cur, block); cur = parentOf(cur) {
		if containsStr(spec.controlKinds, cur.Type(lang)) {
			return true
		}
	}
	return false
}

// nearestBlock returns n's nearest ancestor whose type is a
// statement-list container (see langSpec.blockKinds).
func nearestBlock(n *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	for cur := parentOf(n); cur != nil; cur = parentOf(cur) {
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
// var, Go's plain reassignment, Python/Ruby's assignment). A
// declaration that is a control statement's initializer — Go's
// `if params := ...; cond`, JS's `for (let i = ...)` — is confined to
// that statement, not the block around it.
func bindingBlock(node *ts.Node, spec langSpec, lang *ts.Language) *ts.Node {
	confined := false
	switch node.Type(lang) {
	case "var_spec", "short_var_declaration":
		confined = true
	}
	if p := parentOf(node); p != nil && p.Type(lang) == "lexical_declaration" {
		confined = true
	}
	if !confined {
		return nil
	}
	for cur := parentOf(node); cur != nil; cur = parentOf(cur) {
		if t := cur.Type(lang); containsStr(spec.blockKinds, t) || containsStr(spec.controlKinds, t) {
			return cur
		}
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
