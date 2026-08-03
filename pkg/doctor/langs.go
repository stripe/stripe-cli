package doctor

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// anchorStyle describes how an SDK identifies which API operation a param bag
// belongs to. Dynamic SDKs name it at the call site (receiver + method); typed
// SDKs name it with a dedicated params type. Both are derivable from the
// OpenAPI resource name, so a rule stays language-agnostic.
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

	// spell maps a canonical snake_case OpenAPI param name to this SDK's
	// spelling. SDK-wide convention, not per-parameter knowledge.
	spell func(canonical string) string

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

	// funcKinds are node kinds that delimit a function/method body. Variable
	// resolution is scoped to the enclosing function so a `params` in one
	// function cannot resolve against a Stripe call in another.
	funcKinds []string
}

var specs = map[string]langSpec{
	".rb": {
		name:           "ruby",
		lang:           grammars.RubyLanguage,
		query:          `(pair (hash_key_symbol) @key)`,
		spell:          identity,
		anchorKinds:    []string{"call"},
		anchorStyle:    anchorCall,
		pairKinds:      []string{"pair"},
		assignKinds:    []string{"assignment"},
		funcKinds:      []string{"method", "singleton_method", "lambda", "do_block", "block"},
		resourceTokens: func(r string) []string { return []string{pascalSingular(r)} },
	},
	".py": {
		name: "python",
		lang: grammars.PythonLanguage,
		query: `(keyword_argument (identifier) @key)
				 (pair (string (string_content) @key))`,
		spell:          identity,
		anchorKinds:    []string{"call"},
		anchorStyle:    anchorCall,
		pairKinds:      []string{"keyword_argument", "pair"},
		assignKinds:    []string{"assignment"},
		funcKinds:      []string{"function_definition"},
		resourceTokens: func(r string) []string { return []string{pascalSingular(r)} },
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
		resourceTokens: func(r string) []string {
			return []string{lowerCamelPlural(r), pascalSingular(r)}
		},
	},
	".js": {
		name:           "javascript",
		lang:           grammars.JavascriptLanguage,
		query:          `(pair (property_identifier) @key)`,
		spell:          identity,
		anchorKinds:    []string{"call_expression"},
		anchorStyle:    anchorCall,
		pairKinds:      []string{"pair"},
		assignKinds:    []string{"assignment_expression", "variable_declarator"},
		funcKinds:      []string{"function_declaration", "function_expression", "method_definition", "arrow_function", "generator_function_declaration"},
		resourceTokens: func(r string) []string { return []string{lowerCamelPlural(r)} },
	},
	".ts": {
		name:        "typescript",
		lang:        grammars.TypescriptLanguage,
		query:       `(pair (property_identifier) @key)`,
		spell:       identity,
		anchorKinds: []string{"call_expression"},
		anchorStyle: anchorCall,
		pairKinds:   []string{"pair"},
		assignKinds: []string{"assignment_expression", "variable_declarator"},
		declKinds:   []string{"variable_declarator"},
		// TS names the operation either at the call or in a type annotation:
		// let params: Stripe.PaymentIntentCreateParams
		funcKinds: []string{"function_declaration", "function_expression", "method_definition", "arrow_function", "generator_function_declaration"},
		resourceTokens: func(r string) []string {
			return []string{lowerCamelPlural(r), pascalSingular(r) + "CreateParams"}
		},
	},
	".go": {
		name:           "go",
		lang:           grammars.GoLanguage,
		query:          `(literal_value (keyed_element (literal_element (identifier) @key)))`,
		spell:          pascal,
		anchorKinds:    []string{"composite_literal"},
		anchorStyle:    anchorType,
		pairKinds:      []string{"keyed_element"},
		assignKinds:    []string{"short_var_declaration", "assignment_statement"},
		funcKinds:      []string{"function_declaration", "method_declaration", "func_literal"},
		resourceTokens: func(r string) []string { return []string{pascalSingular(r) + "Params"} },
	},
	".java": {
		name:           "java",
		lang:           grammars.JavaLanguage,
		query:          `(method_invocation (identifier) @key (argument_list))`,
		spell:          javaBuilderAdder, // confirmed against the real SDK
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
		spell:       pascal,
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

// init maps extension variants onto existing specs: ESM/CJS are plain
// JavaScript to tree-sitter (the JS grammar includes JSX), and TSX needs the
// dedicated TSX grammar with the TypeScript spec otherwise unchanged.
func init() {
	specs[".mjs"] = specs[".js"]
	specs[".cjs"] = specs[".js"]
	specs[".jsx"] = specs[".js"]
	tsx := specs[".ts"]
	tsx.name = "tsx"
	tsx.lang = grammars.TsxLanguage
	specs[".tsx"] = tsx
}

func identity(s string) string { return s }

// pascal: payment_method_types -> PaymentMethodTypes
func pascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		parts[i] = title(p)
	}
	return strings.Join(parts, "")
}

// javaBuilderAdder: payment_method_types -> addPaymentMethodType
func javaBuilderAdder(s string) string {
	return "add" + strings.TrimSuffix(pascal(s), "s")
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
