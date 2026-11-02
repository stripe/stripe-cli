package scan

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// anchorStyle is how an SDK names the operation a param bag belongs to:
// at the call site (dynamic SDKs) or via a dedicated params type (typed
// SDKs). Both are derivable from the OpenAPI resource name.
type anchorStyle int

const (
	anchorCall anchorStyle = iota // ruby/python/php/js: Stripe::PaymentIntent.create(...)
	anchorType                    // go/java/csharp: PaymentIntentParams{...}
)

// langSpec is everything language-specific in the engine. Adding a language
// means adding one of these; adding a *rule* means touching none of them.
type langSpec struct {
	name string
	lang func() *ts.Language

	// query captures @key: the node holding the parameter's name.
	query string

	// spell maps a canonical snake_case OpenAPI param name to every
	// spelling this SDK accepts for it. SDK-wide convention, not
	// per-parameter knowledge. Usually one spelling; Java's builder
	// exposes both a singular adder and a bulk "addAll" adder for the
	// same field.
	spell func(canonical string) []string

	// anchorKinds are node kinds that can hold a param bag: a call for dynamic
	// SDKs, a typed literal for typed ones.
	anchorKinds []string
	anchorStyle anchorStyle

	// climbOutermost picks the outermost anchor rather than the nearest. Only
	// Java needs this: a builder chain nests left-to-right, so the type token
	// sits at the innermost receiver and only the outermost node spans it.
	climbOutermost bool

	// pairKinds are node kinds representing one "key: value" entry in a param
	// bag, used to reconstruct a matched parameter's nesting path.
	pairKinds []string

	// assignKinds are node kinds that bind a param bag to a variable, so the
	// engine can follow `params = {...}` to a later `create(params)`.
	assignKinds []string

	// declKinds are node kinds that declare a variable with an explicit type,
	// which is how C# target-typed `new()` and TS annotations name the
	// operation somewhere other than the literal.
	declKinds []string

	// resourceTokens lists every token that identifies this OpenAPI resource in
	// this SDK — a call path, a params type, or both.
	resourceTokens func(resource string) []string

	// unambiguousTokens, when non-nil, is the subset of resourceTokens
	// that alone proves the operation (e.g. Go's PaymentIntentCreateParams).
	// Nil means every resourceTokens entry does (true for every SDK
	// except Go, where the legacy PaymentIntentParams type is shared by
	// create *and* update — matching it alone doesn't establish which).
	unambiguousTokens func(resource string) []string

	// funcKinds are node kinds that delimit a function/method body. Variable
	// resolution is scoped to the enclosing function so a `params` in one
	// function cannot resolve against a Stripe call in another.
	funcKinds []string

	// blockKinds are node kinds that hold a list of sibling statements —
	// a function/method body, an if/for/while body, or a bare {} block.
	// Used two ways: (1) finding the shared container two statements
	// are both direct children of, since some grammars (JS, PHP) wrap
	// every statement in its own per-kind node (lexical_declaration,
	// expression_statement, ...), so comparing Parent() directly never
	// finds true siblings; (2) JS/TS's block-scoped let/const, whose
	// binding only shadows within its own nearest blockKinds ancestor,
	// not the whole enclosing function. Nil (Python/Ruby/Go/Java/C#)
	// falls back to comparing Parent() directly, which already works
	// where statements aren't wrapped this way.
	blockKinds []string

	// controlKinds are conditional/loop statement kinds whose body can
	// be a single statement with no block wrapper of its own — e.g.
	// PHP/JS's `if (cond) $x = 1;` with no braces. Such a body sits
	// directly inside the if/for/while node, which itself still sits
	// directly inside the same blockKinds container as a true sibling
	// statement, so nearestBlock alone can't tell the two apart;
	// climbing through controlKinds (see unconditionalSibling) can.
	// Nil where it doesn't apply: Python/Ruby's grammars still wrap
	// even a one-line conditional body in its own block node, and
	// Go/Java/C# require braces, so blockKinds already catches both.
	controlKinds []string

	// writeKinds are assignKinds entries that write to whatever
	// binding name resolution already finds, rather than introducing
	// a new one — e.g. JS's bare `params = {}` (assignment_expression)
	// inside a nested closure writes through to an outer, captured
	// params instead of shadowing it the way an actual declaration
	// (`let`/`const`/`var`, i.e. variable_declarator) would. Nil where
	// every assignKinds entry already means "introduces a binding":
	// Python/Ruby/PHP treat any bare assignment inside a function as
	// local to it, so even a nested function's own plain assignment
	// really is a distinct, shadowing binding there.
	writeKinds []string
}

var specs = map[string]langSpec{
	".rb": {
		name: "ruby",
		lang: grammars.RubyLanguage,
		// hash_key_symbol: Ruby 1.9+ `key: value`; simple_symbol: hash-rocket
		// `:key => value` (its own text keeps the leading ":", stripped in
		// scan.go); string: hash-rocket `"key" => value`.
		query: `(pair (hash_key_symbol) @key)
				 (pair (simple_symbol) @key)
				 (pair (string (string_content) @key))`,
		spell:       identity,
		anchorKinds: []string{"call"},
		anchorStyle: anchorCall,
		pairKinds:   []string{"pair"},
		assignKinds: []string{"assignment"},
		funcKinds:   []string{"method", "singleton_method", "lambda", "do_block", "block"},
		// pascalSingular(r) is the legacy static style (Stripe::PaymentIntent.create);
		// r itself is the resource-style StripeClient spelling
		// (client.v1.payment_intents.create).
		resourceTokens: func(r string) []string { return []string{pascalSingular(r), r} },
	},
	".py": {
		name: "python",
		lang: grammars.PythonLanguage,
		query: `(keyword_argument (identifier) @key)
				 (pair (string (string_content) @key))`,
		spell:       identity,
		anchorKinds: []string{"call"},
		anchorStyle: anchorCall,
		pairKinds:   []string{"keyword_argument", "pair"},
		assignKinds: []string{"assignment"},
		// lambda is its own scope boundary too — its own parameter
		// shadows an outer binding of the same name just like a def's.
		funcKinds: []string{"function_definition", "lambda"},
		// pascalSingular(r) is the legacy static style (stripe.PaymentIntent.create);
		// r itself is the resource-style StripeClient spelling
		// (client.v1.payment_intents.create).
		resourceTokens: func(r string) []string { return []string{pascalSingular(r), r} },
	},
	".php": {
		name:  "php",
		lang:  grammars.PhpLanguage,
		query: `(array_creation_expression (array_element_initializer (string (string_content) @key)))`,
		spell: identity,
		// Two real SDK styles: client calls ($stripe->setupIntents->create)
		// are member_call_expression; legacy static calls
		// (\Stripe\SetupIntent::create) are scoped_call_expression.
		anchorKinds: []string{"member_call_expression", "scoped_call_expression"},
		anchorStyle: anchorCall,
		pairKinds:   []string{"array_element_initializer"},
		assignKinds: []string{"assignment_expression"},
		funcKinds:   []string{"function_definition", "method_declaration", "anonymous_function_creation_expression", "arrow_function"},
		// PHP wraps every statement in its own node (expression_statement,
		// etc.), so program/compound_statement is the shared container
		// two sibling statements are both direct children of.
		blockKinds: []string{"program", "compound_statement"},
		// PHP's brace-less single-statement bodies (if/for/foreach/
		// while/do/switch/try) have no block wrapper of their own.
		controlKinds: []string{"if_statement", "for_statement", "foreach_statement", "while_statement", "do_statement", "switch_statement", "try_statement"},
		resourceTokens: func(r string) []string {
			return []string{lowerCamelPlural(r), pascalSingular(r)}
		},
	},
	".js": {
		name: "javascript",
		lang: grammars.JavascriptLanguage,
		// property_identifier: {payment_method_types: [...]}; string:
		// {"payment_method_types": [...]}; shorthand_property_identifier:
		// {payment_method_types} (sugar for {payment_method_types:
		// payment_method_types}), a direct child of object rather than
		// nested in a pair — see keyInKeyPosition/extractValue/
		// ancestorKeys for the special-casing it needs.
		query: `(pair (property_identifier) @key)
				 (pair (string (string_fragment) @key))
				 (object (shorthand_property_identifier) @key)`,
		spell:       identity,
		anchorKinds: []string{"call_expression"},
		anchorStyle: anchorCall,
		pairKinds:   []string{"pair"},
		assignKinds: []string{"assignment_expression", "variable_declarator"},
		funcKinds:   []string{"function_declaration", "function_expression", "method_definition", "arrow_function", "generator_function_declaration"},
		// JS wraps every statement in its own node (lexical_declaration,
		// expression_statement, etc.), so program/statement_block is the
		// shared container two sibling statements are both direct
		// children of. Also the scope a let/const binding is limited to.
		blockKinds: []string{"program", "statement_block"},
		// JS's brace-less single-statement bodies have no block
		// wrapper of their own, same issue as PHP's.
		controlKinds: []string{"if_statement", "for_statement", "for_in_statement", "while_statement", "do_statement", "switch_statement", "try_statement"},
		// A bare `params = {}` (no let/const/var) inside a nested
		// function writes through to whatever params resolution
		// already finds — an outer, captured variable, not a new
		// local — so it must not be treated as shadowing on its own.
		writeKinds:     []string{"assignment_expression"},
		resourceTokens: func(r string) []string { return []string{lowerCamelPlural(r)} },
	},
	".ts": {
		name: "typescript",
		lang: grammars.TypescriptLanguage,
		query: `(pair (property_identifier) @key)
				 (pair (string (string_fragment) @key))
				 (object (shorthand_property_identifier) @key)`,
		spell:       identity,
		anchorKinds: []string{"call_expression"},
		anchorStyle: anchorCall,
		pairKinds:   []string{"pair"},
		assignKinds: []string{"assignment_expression", "variable_declarator"},
		declKinds:   []string{"variable_declarator"},
		// TS names the operation either at the call or in a type annotation:
		// let params: Stripe.PaymentIntentCreateParams
		funcKinds:    []string{"function_declaration", "function_expression", "method_definition", "arrow_function", "generator_function_declaration"},
		blockKinds:   []string{"program", "statement_block"},
		controlKinds: []string{"if_statement", "for_statement", "for_in_statement", "while_statement", "do_statement", "switch_statement", "try_statement"},
		writeKinds:   []string{"assignment_expression"},
		resourceTokens: func(r string) []string {
			return []string{lowerCamelPlural(r), pascalSingular(r) + "CreateParams"}
		},
	},
	".go": {
		name:        "go",
		lang:        grammars.GoLanguage,
		query:       `(literal_value (keyed_element (literal_element (identifier) @key)))`,
		spell:       pascalSpell,
		anchorKinds: []string{"composite_literal"},
		anchorStyle: anchorType,
		pairKinds:   []string{"keyed_element"},
		// var_spec: `var params = &stripe.PaymentIntentParams{...}` —
		// its own first named child is the bound identifier directly,
		// same as short_var_declaration/assignment_statement, but it's
		// one level deeper (wrapped in its own var_declaration, unlike
		// the other two, which sit directly in the statement list) —
		// see blockKinds below for why that depth difference matters.
		assignKinds: []string{"short_var_declaration", "assignment_statement", "var_spec"},
		funcKinds:   []string{"function_declaration", "method_declaration", "func_literal"},
		// var_spec's extra var_declaration wrapper means comparing
		// Parent() directly no longer finds true statement-list
		// siblings between a var_spec-based assignment and a
		// short_var_declaration/assignment_statement-based one (same
		// problem JS/PHP have for other reasons) — climb to the
		// nearest enclosing block instead. declaresParam/
		// blockScopedShadow's lexical_declaration check is JS/TS-only
		// and never matches a Go node, so this is otherwise a no-op
		// for Go's existing shadow detection.
		blockKinds: []string{"source_file", "block"},
		// <Resource>Params is the legacy type, shared by create and
		// update; <Resource>CreateParams is the operation-specific type
		// the current client-based stripe-go API generates.
		resourceTokens: func(r string) []string {
			base := pascalSingular(r)
			return []string{base + "Params", base + "CreateParams"}
		},
		unambiguousTokens: func(r string) []string {
			return []string{pascalSingular(r) + "CreateParams"}
		},
	},
	".java": {
		name:           "java",
		lang:           grammars.JavaLanguage,
		query:          `(method_invocation (identifier) @key (argument_list))`,
		spell:          javaBuilderAdders, // confirmed against the real SDK
		anchorKinds:    []string{"method_invocation"},
		anchorStyle:    anchorType,
		climbOutermost: true,
		pairKinds:      nil,
		assignKinds:    []string{"variable_declarator"},
		declKinds:      []string{"local_variable_declaration"},
		funcKinds:      []string{"method_declaration", "constructor_declaration", "lambda_expression"},
		resourceTokens: func(r string) []string { return []string{pascalSingular(r) + "CreateParams"} },
	},
	".cs": {
		name:        "csharp",
		lang:        grammars.CSharpLanguage,
		query:       `(initializer_expression (assignment_expression (identifier) @key))`,
		spell:       pascalSpell,
		anchorKinds: []string{"object_creation_expression", "implicit_object_creation_expression"},
		anchorStyle: anchorType,
		pairKinds:   []string{"assignment_expression"},
		assignKinds: []string{"assignment_expression", "variable_declarator"},
		// Target-typed `new()` puts the type on the declaration, not the
		// literal: `PaymentIntentCreateOptions options;`
		declKinds:      []string{"variable_declaration"},
		funcKinds:      []string{"method_declaration", "constructor_declaration", "local_function_statement", "lambda_expression"},
		resourceTokens: func(r string) []string { return []string{pascalSingular(r) + "CreateOptions"} },
	},
}

// ESM/CJS/JSX parse as plain JavaScript; TSX needs its own grammar.
func init() {
	specs[".mjs"] = specs[".js"]
	specs[".cjs"] = specs[".js"]
	specs[".jsx"] = specs[".js"]
	tsx := specs[".ts"]
	tsx.name = "tsx"
	tsx.lang = grammars.TsxLanguage
	specs[".tsx"] = tsx
}

// SupportsExt reports whether ext (lowercased, with its leading dot, e.g.
// ".rb") names a language this package can scan.
func SupportsExt(ext string) bool {
	_, ok := specs[ext]
	return ok
}

func identity(s string) []string { return []string{s} }

// pascal: payment_method_types -> PaymentMethodTypes
func pascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		parts[i] = title(p)
	}
	return strings.Join(parts, "")
}

func pascalSpell(s string) []string { return []string{pascal(s)} }

// javaBuilderAdders: payment_method_types -> [addPaymentMethodType,
// addAllPaymentMethodType] — the builder exposes both a singular adder
// and a bulk "addAll" adder for the same field.
func javaBuilderAdders(s string) []string {
	base := strings.TrimSuffix(pascal(s), "s")
	return []string{"add" + base, "addAll" + base}
}

// pascalSingular: payment_intents -> PaymentIntent
func pascalSingular(resource string) string {
	return strings.TrimSuffix(pascal(resource), "s")
}

// lowerCamelPlural: payment_intents -> paymentIntents
func lowerCamelPlural(resource string) string {
	p := pascal(resource)
	if p == "" {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
