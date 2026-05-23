package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"
	"github.com/stripe/stripe-cli-docs-plugin/cmd"
)

func main() {
	out := flag.String("out", "./docs/cli", "output directory")
	format := flag.String("format", "markdown", "markdown|man|rest")
	front := flag.Bool("frontmatter", false, "prepend YAML front matter to markdown files")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cmd.New().Root()
	root.DisableAutoGenTag = true

	switch *format {
	case "markdown":
		if *front {
			prepender := func(filename string) string {
				base := filepath.Base(filename)
				name := strings.TrimSuffix(base, filepath.Ext(base))
				title := strings.ReplaceAll(name, "_", " ")
				return fmt.Sprintf("---\ntitle: %q\nslug: %q\ndescription: \"CLI reference for %s\"\n---\n\n", title, name, title)
			}
			link := func(name string) string { return strings.ToLower(name) }
			if err := doc.GenMarkdownTreeCustom(root, *out, prepender, link); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := doc.GenMarkdownTree(root, *out); err != nil {
				log.Fatal(err)
			}
		}
	case "man":
		hdr := &doc.GenManHeader{Title: strings.ToUpper(root.Name()), Section: "1"}
		if err := doc.GenManTree(root, hdr, *out); err != nil {
			log.Fatal(err)
		}
	case "rest":
		if err := doc.GenReSTTree(root, *out); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown format: %s", *format)
	}
}
