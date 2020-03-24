package resource

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/kr/text"
	"github.com/russross/blackfriday"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/spec"
)

const tab = "    "

type ResourceReference struct {
	object string
	cmd    *cobra.Command
	args   []string
	spec   *spec.Spec
}

func (rh *ResourceReference) helpFunc() {
	var err error

	rh.spec, err = spec.LoadSpec(pathStripeSpec)
	if err != nil {
		panic(err)
	}

	err = pageString(rh.helpString(), rh.cmd.OutOrStdout())
	if err != nil {
		panic(err)
	}
}

func (rh *ResourceReference) helpString() string {
	var sb strings.Builder

	writeLine(&sb, ansi.Bold("OBJECT"))
	writeLine(&sb, rh.object)
	writeLine(&sb, "")

	if rh.cmd.HasSubCommands() {
		writeLine(&sb, ansi.Bold("OPERATIONS"))

		for _, c := range rh.cmd.Commands() {
			writeLine(&sb, text.Indent(c.Name(), tab))
		}

		writeLine(&sb, "")
	}

	writeLine(&sb, ansi.Bold("OBJECT"))
	response := rh.spec.Components.Schemas[rh.object]
	writeParams(&sb, response, rh.spec, tab)

	return sb.String()
}

type OperationReference struct {
	oc   *OperationCmd
	cmd  *cobra.Command
	args []string
	spec *spec.Spec
}

func (oh *OperationReference) helpFunc() {
	var err error

	oh.spec, err = spec.LoadSpec(pathStripeSpec)
	if err != nil {
		panic(err)
	}

	err = pageString(oh.helpString(), oh.cmd.OutOrStdout())
	if err != nil {
		panic(err)
	}
}

func (oh *OperationReference) helpString() string {
	var sb strings.Builder

	opSpec := oh.spec.Paths[spec.Path(oh.oc.Path)][spec.HTTPVerb(strings.ToLower(oh.oc.HTTPVerb))]

	writeLine(&sb, ansi.Bold("USAGE"))
	writeLine(&sb, text.Indent(oh.cmd.CommandPath(), tab))
	writeLine(&sb, "")
	writeLine(&sb, ansi.Bold("PATH"))
	writeLine(&sb, text.Indent(fmt.Sprint(ansi.ColorizeHTTPVerb(oh.oc.HTTPVerb), " ", oh.oc.Path), tab))
	writeLine(&sb, "")
	writeLine(&sb, ansi.Bold("DESCRIPTION"))
	writeLine(&sb, text.Indent(p.Sanitize(text.Wrap(opSpec.Description, 76)), tab))
	writeLine(&sb, "")

	writeLine(&sb, ansi.Bold("PARAMETERS"))
	paramsSchema := opSpec.RequestBody.Content["application/x-www-form-urlencoded"].Schema
	writeParams(&sb, paramsSchema, oh.spec, tab)

	sb.WriteString(fmt.Sprintln("\n" + ansi.Bold("RESPONSE")))
	sb.WriteString(fmt.Sprint(ansi.Italic("SUCCESS")))
	successResponseSchema := opSpec.Responses["200"].Content["application/json"].Schema.Dereference(oh.spec)
	writeParams(&sb, successResponseSchema, oh.spec, tab)

	sb.WriteString(fmt.Sprintln("\n" + ansi.Italic("ERROR")))
	errorResponseSchema := opSpec.Responses["default"].Content["application/json"].Schema.Dereference(oh.spec)
	writeParams(&sb, errorResponseSchema, oh.spec, tab)

	return sb.String()
}

func paramHelpString(name string, schema *spec.Schema, tab string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%s• %s %s\n", tab, ansi.Bold(name), ansi.Italic(ansi.Faint(schema.Type))))

	tab += "  "

	if len(schema.Description) > 0 {
		converted := string(blackfriday.Markdown([]byte(schema.Description), ansi.MarkdownTermRenderer(0), 0))
		wrapped := text.Wrap(converted, 80-len(tab))
		sb.WriteString(text.Indent(wrapped, tab))
	}

	for _, subName := range sortedParamNames(schema) {
		subSchema := schema.Properties[subName]
		sb.WriteString(paramHelpString(subName, subSchema, tab))
	}

	return sb.String()
}

func sortedParamNames(schema *spec.Schema) []string {
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeParams(sb *strings.Builder, paramSchema *spec.Schema, spec *spec.Spec, space string) {
	for _, name := range sortedParamNames(paramSchema) {
		schema := paramSchema.Properties[name]
		sb.WriteString(paramHelpString(name, schema, space))
		if schema.Ref != "" {
			writeParams(sb, schema.Dereference(spec), spec, space+tab)
		}
		sb.WriteString("\n")
	}
}

func pageString(s string, out io.Writer) error {
	pagerExe := ""
	switch {
	case len(os.Getenv("PAGER")) > 0:
		pagerExe = os.Getenv("PAGER")
	case runtime.GOOS == "windows":
		pagerExe = "more"
	default:
		pagerExe = "less"
	}

	pager := exec.Command(pagerExe)

	pager.Stdin = strings.NewReader(s)
	pager.Stdout = out

	err := pager.Run()
	if err != nil {
		return err
	}

	return nil
}

func writeLine(sb *strings.Builder, content string) {
	sb.WriteString(fmt.Sprintln((content)))
}
