//+build ignore

package main

import (
	"log"

	"github.com/shurcooL/vfsgen"

	"github.com/stripe/stripe-cli/pkg/spec"
)

func main() {
	err := vfsgen.Generate(spec.FS, vfsgen.Options{
		PackageName:  "spec",
		BuildTags:    "!dev",
		VariableName: "FS",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
