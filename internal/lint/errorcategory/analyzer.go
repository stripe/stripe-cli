// Package errorcategory defines an analyzer that requires categories on newly
// created errors while allowing context-only error wrapping.
package errorcategory

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	analyzerName      = "errorcategory"
	errorcategoryPath = "github.com/stripe/stripe-cli/pkg/errorcategory"
	suppression       = "nolint:" + analyzerName
)

// Analyzer requires repository-authored error origins to have a category.
var Analyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  "requires errors.New and non-wrapping fmt.Errorf calls to use categorized constructors",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Path() == errorcategoryPath {
		return nil, nil
	}

	for _, file := range pass.Files {
		filename := pass.Fset.File(file.Pos()).Name()
		if strings.HasSuffix(filename, "_test.go") || ast.IsGenerated(file) {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			packagePath, function := calledFunction(pass, call)
			switch {
			case packagePath == "errors" && function == "New":
				report(pass, file, call, "errors.New creates an uncategorized error; use errorcategory.New")
			case packagePath == "fmt" && function == "Errorf":
				if len(call.Args) == 0 {
					return true
				}
				format, ok := constantString(pass, call.Args[0])
				if !ok {
					report(pass, file, call, "fmt.Errorf with a non-constant format cannot be verified as wrapping; use errorcategory.Errorf")
				} else if !containsWrapDirective(format) {
					report(pass, file, call, "fmt.Errorf creates an uncategorized error; use errorcategory.Errorf")
				}
			}

			return true
		})
	}

	return nil, nil
}

func calledFunction(pass *analysis.Pass, call *ast.CallExpr) (string, string) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}

	identifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", ""
	}

	packageName, ok := pass.TypesInfo.Uses[identifier].(*types.PkgName)
	if !ok {
		return "", ""
	}

	return packageName.Imported().Path(), selector.Sel.Name
}

func constantString(pass *analysis.Pass, expression ast.Expr) (string, bool) {
	value := pass.TypesInfo.Types[expression].Value
	if value == nil || value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(value), true
}

func containsWrapDirective(format string) bool {
	for index := 0; index < len(format); index++ {
		if format[index] != '%' {
			continue
		}
		index++
		if index >= len(format) {
			return false
		}
		if format[index] == '%' {
			continue
		}

		for ; index < len(format); index++ {
			character := format[index]
			if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') {
				if character == 'w' {
					return true
				}
				break
			}
		}
	}
	return false
}

func report(pass *analysis.Pass, file *ast.File, call *ast.CallExpr, message string) {
	comment, found := suppressionComment(pass.Fset, file, call.Pos())
	if found {
		reason := strings.TrimSpace(strings.TrimPrefix(comment, suppression))
		reason = strings.TrimLeft(reason, " -:")
		if reason != "" {
			return
		}
		message = "//" + suppression + " requires a reason"
	}
	pass.Reportf(call.Pos(), "%s", message)
}

func suppressionComment(files *token.FileSet, file *ast.File, position token.Pos) (string, bool) {
	callLine := files.Position(position).Line
	for _, group := range file.Comments {
		for _, comment := range group.List {
			commentLine := files.Position(comment.Pos()).Line
			if commentLine != callLine && commentLine != callLine-1 {
				continue
			}
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if index := strings.Index(text, suppression); index >= 0 {
				return text[index:], true
			}
		}
	}
	return "", false
}
