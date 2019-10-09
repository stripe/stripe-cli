package stripe

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerboseTransport_Verbose(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req_123")
		w.Header().Set("Non-Whitelisted-Header", "foo")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	var b bytes.Buffer

	httpTransport := &http.Transport{}
	tr := &verboseTransport{
		Transport: httpTransport,
		Verbose:   true,
		Out:       &b,
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("POST", ts.URL+"/test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	out := b.String()
	require.Regexp(t, regexp.MustCompile("> POST http://(.+)/test\n"), out)
	require.Contains(t, out, "> Authorization: Bearer [REDACTED]\n")
	require.Contains(t, out, "> Content-Type: application/x-www-form-urlencoded\n")
	require.Contains(t, out, "< HTTP 200\n")
	require.Contains(t, out, "< Request-Id: req_123\n")
	require.NotContains(t, out, "Non-Whitelisted-Header")
}
