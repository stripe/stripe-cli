package postinstall

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintTips_Directory_PrintsTips(t *testing.T) {
	var output bytes.Buffer
	PrintTips(&output, "directory")

	// Protects that installing Directory gives users at least one clear feedback path.
	assert.Contains(t, output.String(), "directory@stripe.com")
}

func TestPrintTips_Projects_PrintsTips(t *testing.T) {
	var output bytes.Buffer
	PrintTips(&output, "projects")

	out := output.String()
	assert.Contains(t, out, "stripe projects catalog")
	assert.Contains(t, out, "stripe projects init")
	assert.Contains(t, out, "https://docs.stripe.com/projects")
	assert.Contains(t, out, "https://projects.dev")
	assert.Contains(t, out, "projects-feedback@stripe.com")
	assert.Contains(t, out, "npx skills add")
}
