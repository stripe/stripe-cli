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
