package coop

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

var snippetEndpoint = "https://docs.stripe.com/_endpoint/generate-example-snippet"

var httpClient = &http.Client{Timeout: 5 * time.Second}

// FetchSDKSnippet fetches a generated SDK code snippet from the docs endpoint.
// path is the API path (e.g. "/v1/customers"), method is "post"/"get"/etc,
// params is the JSON-encoded request params, and language is "node"/"python"/etc.
func FetchSDKSnippet(path, method string, params interface{}, language string) (string, error) {
	argsJSON := "{}"
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return "", fmt.Errorf("marshaling snippet params: %w", err)
		}
		argsJSON = string(data)
	}

	u, _ := url.Parse(snippetEndpoint)
	q := u.Query()
	q.Set("path", path)
	q.Set("verb", method)
	q.Set("args", argsJSON)
	u.RawQuery = q.Encode()

	resp, err := httpClient.Get(u.String())
	if err != nil {
		return "", fmt.Errorf("fetching snippet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("snippet API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading snippet response: %w", err)
	}

	var snippets map[string]string
	if err := json.Unmarshal(body, &snippets); err != nil {
		return "", fmt.Errorf("parsing snippet response: %w", err)
	}

	snippet, ok := snippets[language]
	if !ok {
		return "", fmt.Errorf("language %q not available (have: curl, node, python, ruby, php, java, go, dotnet)", language)
	}

	return snippet, nil
}

// ShouldFetchSDKSnippet reports whether the docs snippet endpoint has enough
// blueprint data to produce a useful example. Mutating calls without params tend
// to generate misleading empty SDK calls, such as checkout.sessions.create().
func ShouldFetchSDKSnippet(req *APIRequest) bool {
	if req == nil || strings.TrimSpace(req.Path) == "" || strings.TrimSpace(req.Method) == "" {
		return false
	}
	if hasBlueprintReferences(req.Path) {
		return false
	}
	if !isMutatingMethod(req.Method) {
		return true
	}
	return requestHasParams(req.Params)
}

// SDKSnippetGuidance returns a code-comment fallback when the blueprint only
// specifies an endpoint and method.
func SDKSnippetGuidance(req *APIRequest, language string) string {
	if req == nil {
		return ""
	}
	prefix := commentPrefix(language)
	lines := []string{
		fmt.Sprintf("%s Blueprint request: %s %s", prefix, strings.ToUpper(req.Method), req.Path),
	}
	if refs := BlueprintReferences(req.Path, req.Params); len(refs) > 0 {
		lines = append(lines, fmt.Sprintf("%s Resolve blueprint references from prior step outputs at runtime: %s.", prefix, strings.Join(refs, ", ")))
	}
	if requestHasParams(req.Params) {
		lines = append(lines, fmt.Sprintf("%s Use api_request.params as the canonical request shape from the blueprint.", prefix))
	} else {
		lines = append(lines, fmt.Sprintf("%s This blueprint node does not include canonical request params yet.", prefix))
		lines = append(lines, fmt.Sprintf("%s Do not treat an empty SDK call as complete; wire the app to this endpoint and use the step intent, earlier IDs, and Stripe docs to fill the params.", prefix))
	}
	return strings.Join(lines, "\n")
}

func requestHasParams(params interface{}) bool {
	if params == nil {
		return false
	}
	value := reflect.ValueOf(params)
	switch value.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		return value.Len() > 0
	default:
		return true
	}
}

func isMutatingMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}

func commentPrefix(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "python", "ruby":
		return "#"
	default:
		return "//"
	}
}
