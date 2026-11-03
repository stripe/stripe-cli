package scan

// Hidden grammar-debugging helper behind `stripe parse-dump`.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// DumpTrees prints the parsed S-expression tree for every file under root
// (recursively, e.g. testdata/<language>/) whose extension this package
// supports.
func DumpTrees(root string) {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	for _, f := range files {
		spec, ok := specs[strings.ToLower(filepath.Ext(f))]
		if !ok {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		lang := spec.lang()
		tree, err := ts.NewParser(lang).Parse(src)
		if err != nil {
			fmt.Printf("%s: parse error: %v\n", f, err)
			continue
		}
		fmt.Printf("######## %s (%s) ########\n%s\n", filepath.Base(f), spec.name,
			tree.RootNode().SExpr(lang))
	}
}
