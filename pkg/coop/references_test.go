package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkNodeReferencesVisitsNestedValuesAndObjectKeys(t *testing.T) {
	value := map[string]any{
		"${node.setup.create-product:lookup_key}": []any{
			"/v1/products/${node.setup.create-product:id}",
			map[string]any{
				"clock": "${node.setup.create-clock.create-request:id}",
			},
		},
	}

	var references []nodeReference
	err := walkNodeReferences(value, func(reference nodeReference) error {
		references = append(references, reference)
		return nil
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, []nodeReference{
		{Ref: "setup.create-product", Base: "setup.create-product", Field: "lookup_key"},
		{Ref: "setup.create-product", Base: "setup.create-product", Field: "id"},
		{
			Ref:    "setup.create-clock.create-request",
			Base:   "setup.create-clock",
			Source: "create-request",
			Field:  "id",
		},
	}, references)
}
