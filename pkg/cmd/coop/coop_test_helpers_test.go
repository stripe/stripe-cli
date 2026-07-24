package coopcmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type commandTestBlueprintRepository struct{}

func (commandTestBlueprintRepository) List(context.Context) ([]coop.WorkbenchBlueprintSummary, error) {
	return []coop.WorkbenchBlueprintSummary{
		{
			ID:               "blpt_one_time",
			Key:              "one-time-payment",
			BlueprintType:    "learning",
			BlueprintVersion: 6,
			TemplateVersion:  1,
			Title:            coop.MessageDescriptor{DefaultMessage: "Accept a one-time payment"},
			Description:      coop.MessageDescriptor{DefaultMessage: "Create and verify a one-time payment."},
			StepRefs:         []coop.WorkbenchStepRef{{StepKey: "one-time-payment--setup", StepVersion: 2}},
			Metadata:         coop.BlueprintMetadata{Products: []string{"Payments"}},
		},
		{ID: "blpt_flat_fee", Key: "flat-fee", BlueprintType: "learning"},
		{ID: "blpt_flat_subscription", Key: "flat-subscription", BlueprintType: "learning"},
		{
			ID:               "blpt_testing",
			Key:              "testing-only",
			BlueprintType:    "testing",
			BlueprintVersion: 1,
			Title:            coop.MessageDescriptor{DefaultMessage: "Testing only"},
		},
	}, nil
}

func (commandTestBlueprintRepository) Retrieve(_ context.Context, key string) (*coop.WorkbenchBlueprint, error) {
	return &coop.WorkbenchBlueprint{
		WorkbenchBlueprintSummary: coop.WorkbenchBlueprintSummary{
			ID:               "blpt_one_time",
			Key:              key,
			BlueprintType:    "learning",
			BlueprintVersion: 6,
			TemplateVersion:  1,
			Title:            coop.MessageDescriptor{DefaultMessage: "Accept a one-time payment"},
			Description:      coop.MessageDescriptor{DefaultMessage: "Create and verify a one-time payment."},
			Metadata:         coop.BlueprintMetadata{Products: []string{"Payments"}},
		},
		Steps: []coop.WorkbenchStep{{
			Key:             key + "--setup",
			StepVersion:     2,
			TemplateVersion: 1,
			Title:           coop.MessageDescriptor{DefaultMessage: "Set up payment"},
			Required:        true,
			Nodes: []coop.WorkbenchBlueprintNode{{
				NodeType:    coop.NodeAPIRequest,
				Key:         "create-payment",
				Title:       coop.MessageDescriptor{DefaultMessage: "Create payment"},
				Description: coop.MessageDescriptor{DefaultMessage: "Create a PaymentIntent and save its identifier."},
				APIRequestDetails: &coop.WorkbenchAPIRequestDetails{
					Fixture: coop.WorkbenchRequestFixture{
						Method: "POST",
						Path:   "/v1/payment_intents",
						Params: map[string]any{"amount": float64(2000), "currency": "usd"},
					},
				},
			}},
		}},
	}, nil
}

func init() {
	options.BlueprintRepository = commandTestBlueprintRepository{}
}

func commandTestCompiledBlueprint(t *testing.T) *coop.Blueprint {
	t.Helper()
	source, err := (commandTestBlueprintRepository{}).Retrieve(t.Context(), "one-time-payment")
	require.NoError(t, err)
	blueprint, err := coop.CompileBlueprint(source, nil)
	require.NoError(t, err)
	return blueprint
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return captureOutput(t, &os.Stdout, fn)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return captureOutput(t, &os.Stderr, fn)
}

func captureOutput(t *testing.T, target **os.File, fn func()) string {
	t.Helper()

	orig := *target
	r, w, err := os.Pipe()
	require.NoError(t, err)
	*target = w

	var buf bytes.Buffer
	readErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(&buf, r)
		readErr <- err
	}()

	closed := false
	defer func() {
		*target = orig
		if !closed {
			_ = w.Close()
		}
		_ = r.Close()
	}()

	fn()

	closed = true
	require.NoError(t, w.Close())
	*target = orig

	require.NoError(t, <-readErr)
	require.NoError(t, r.Close())
	return strings.TrimSpace(buf.String())
}

func TestCaptureStdoutDrainsLargeOutput(t *testing.T) {
	output := captureStdout(t, func() {
		_, err := os.Stdout.WriteString(strings.Repeat("x", 128*1024))
		require.NoError(t, err)
	})

	assert.Len(t, output, 128*1024)
}
