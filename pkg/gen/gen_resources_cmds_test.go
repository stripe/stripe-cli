//go:build gen_resources
// +build gen_resources

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/spec"
)

func TestRequestMediaType(t *testing.T) {
	var op spec.Operation
	require.NoError(t, json.Unmarshal([]byte(`{
		"requestBody": {"content": {
			"application/x-www-form-urlencoded": {"schema": {"type": "object", "properties": {"description": {"type": "string"}}}},
			"application/json": {"schema": {"type": "object", "properties": {"enabled": {"type": "boolean"}}}},
			"multipart/form-data": {"schema": {
				"type": "object", "required": ["file", "purpose"],
				"properties": {"file": {"type": "string", "format": "binary"}, "purpose": {"type": "string"}}
			}}
		}}
	}`), &op))
	for _, tc := range []struct {
		namespace ApiNamespace
		want      string
		param     string
	}{
		{V1Namespace, "application/x-www-form-urlencoded", "description"},
		{V1PreviewNamespace, "application/x-www-form-urlencoded", "description"},
		{V2Namespace, "application/json", "enabled"},
		{V2PreviewNamespace, "application/json", "enabled"},
	} {
		t.Run(tc.namespace, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				selected := getMediaType(tc.namespace, &op)
				require.Equal(t, tc.want, selected)
				params := getOperationParams(selected, &op, spec.StripeOperation{Operation: "post"})
				require.Contains(t, params, tc.param)
				require.NotContains(t, params, "file")
			}
		})
	}
	delete(op.RequestBody.Content, "application/x-www-form-urlencoded")
	require.Equal(t, "multipart/form-data", getMediaType(V1Namespace, &op))
	require.Equal(t, "application/json", getMediaType(V2Namespace, &op))
	delete(op.RequestBody.Content, "application/json")
	for _, namespace := range []ApiNamespace{V1Namespace, V2Namespace} {
		require.Equal(t, "multipart/form-data", getMediaType(namespace, &op))
	}

	stripeOp := spec.StripeOperation{MethodName: "create", Operation: "post", Path: "/v1/example"}
	op.Servers = []spec.Server{{URL: "https://files.stripe.com"}}
	api := &spec.Spec{Paths: map[spec.Path]map[spec.HTTPVerb]*spec.Operation{
		"/v1/example": {"post": &op},
	}}
	data := &TemplateData{ApiNamespaces: make(map[ApiNamespace]*ApiNamespaceData)}
	require.NoError(t, addToTemplateData(data, V1Namespace, "", "example", "", api, stripeOp))
	generated := data.ApiNamespaces[V1Namespace].Namespaces[""].Resources["examples"].Operations["create"]
	require.Equal(t, "multipart/form-data", generated.RequestMediaType)
	require.Equal(t, "https://files.stripe.com", generated.ServerURL)
	require.Equal(t, "string", generated.Params["file"].Type)
	require.Equal(t, "binary", generated.Params["file"].Format)
	require.True(t, generated.Params["file"].Required)
	require.Contains(t, generated.Params, "purpose")

	source, err := os.ReadFile("resource_specs.go.tpl")
	require.NoError(t, err)
	tmpl, err := template.New(pathSpecName).Funcs(templateFuncs).Parse(string(source))
	require.NoError(t, err)
	var rendered bytes.Buffer
	require.NoError(t, tmpl.Execute(&rendered, []varEntry{{VarName: generated.VarName, Op: generated}}))
	require.Contains(t, rendered.String(), `RequestMediaType: "multipart/form-data"`)
	require.Contains(t, rendered.String(), `Format: "binary"`)
	require.Contains(t, rendered.String(), `ServerURL: "https://files.stripe.com"`)

	for _, mediaType := range []string{"", "application/x-www-form-urlencoded", "application/json"} {
		t.Run("render default "+mediaType, func(t *testing.T) {
			ordinary := *generated
			ordinary.RequestMediaType = mediaType
			var output bytes.Buffer
			require.NoError(t, tmpl.Execute(&output, []varEntry{{VarName: ordinary.VarName, Op: &ordinary}}))
			require.NotContains(t, output.String(), "RequestMediaType:")
		})
	}
}

func TestRequestMediaTypeAbsentAndFallback(t *testing.T) {
	require.Empty(t, getMediaType(V1Namespace, &spec.Operation{}))
	require.Empty(t, getMediaType(V1Namespace, &spec.Operation{RequestBody: &spec.RequestBody{}}))
	op := &spec.Operation{RequestBody: &spec.RequestBody{Content: map[string]spec.MediaType{
		"text/plain": {}, "application/json": {},
	}}}
	for i := 0; i < 20; i++ {
		require.Empty(t, getMediaType(V1Namespace, op))
	}
}
