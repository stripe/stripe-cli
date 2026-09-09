// Package errorcategory defines an analyzer that requires categories on newly
// created errors while allowing errors wrapped with fmt.Errorf and %w.
package errorcategory

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const categoryPackage = "github.com/stripe/stripe-cli/pkg/errorcategory"

// Analyzer reports error creation sites that do not assign a semantic category.
var Analyzer = &analysis.Analyzer{
	Name: "errorcategory",
	Doc:  "require semantic categories when creating errors",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Path() == categoryPackage {
		return nil, nil
	}

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") || ast.IsGenerated(file) {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			pkgPath, name := calledFunction(pass, call)
			switch {
			case pkgPath == "errors" && name == "New":
				if !isSuppressed(pass, file, call) {
					pass.Reportf(call.Pos(), "errors.New creates an uncategorized error; use errorcategory.New")
				}
			case pkgPath == "fmt" && name == "Errorf":
				if !isWrappingErrorf(pass, call) && !isSuppressed(pass, file, call) {
					pass.Reportf(call.Pos(), "fmt.Errorf creates an uncategorized error; use errorcategory.Errorf (only constant formats containing %%w may use fmt.Errorf)")
				}
			}

			return true
		})
	}

	return nil, nil
}

func calledFunction(pass *analysis.Pass, call *ast.CallExpr) (string, string) {
	var object types.Object
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		object = pass.TypesInfo.Uses[fun]
	case *ast.SelectorExpr:
		object = pass.TypesInfo.Uses[fun.Sel]
	}

	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return "", ""
	}
	return function.Pkg().Path(), function.Name()
}

func isWrappingErrorf(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	value := pass.TypesInfo.Types[call.Args[0]].Value
	if value == nil || value.Kind() != constant.String {
		return false
	}
	return hasWrappingDirective(constant.StringVal(value))
}

func hasWrappingDirective(format string) bool {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i >= len(format) {
			return false
		}
		if format[i] == '%' {
			continue
		}

		for ; i < len(format); i++ {
			if format[i] == 'w' {
				return true
			}
			if (format[i] >= 'a' && format[i] <= 'z') || (format[i] >= 'A' && format[i] <= 'Z') {
				break
			}
		}
	}
	return false
}

func isSuppressed(pass *analysis.Pass, file *ast.File, call *ast.CallExpr) bool {
	callStart := pass.Fset.Position(call.Pos()).Line
	callEnd := pass.Fset.Position(call.End()).Line

	for _, group := range file.Comments {
		commentStart := pass.Fset.Position(group.Pos()).Line
		commentEnd := pass.Fset.Position(group.End()).Line
		adjacentBefore := commentEnd == callStart-1
		overlapsCall := commentStart <= callEnd && commentEnd >= callStart
		if !adjacentBefore && !overlapsCall {
			continue
		}

		for _, comment := range group.List {
			const directive = "nolint:errorcategory"
			index := strings.Index(comment.Text, directive)
			if index < 0 {
				continue
			}

			reason := strings.Trim(comment.Text[index+len(directive):], " \t-:*/")
			if reason == "" {
				pass.Reportf(call.Pos(), "//nolint:errorcategory must include a reason why no semantic category can be assigned")
			}
			return true
		}
	}

	return false
}
