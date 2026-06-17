package coop

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
		if err == nil {
			argsJSON = string(data)
		}
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
