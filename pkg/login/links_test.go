package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLinks(t *testing.T) {
	expectedLinks := Links{
		BrowserURL:       "https://stripe.com/browser",
		PollURL:          "https://stripe.com/poll",
		VerificationCode: "dinosaur-pineapple-polkadot",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		require.NoError(t, r.ParseForm())
		require.Equal(t, "test", r.PostFormValue("device_name"))

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedLinks)
	}))
	defer ts.Close()

	links, err := GetLinks(context.Background(), ts.URL, "test")
	require.NoError(t, err)
	require.Equal(t, expectedLinks, *links)
}

func TestGetLinksHTTPStatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	links, err := GetLinks(context.Background(), ts.URL, "test")
	require.EqualError(t, err, "unexpected http status code: 500 ")
	require.Empty(t, links)
}

func TestGetLinksRequestError(t *testing.T) {
	errorString := ""

	if runtime.GOOS == "windows" {
		errorString = "connectex: No connection could be made because the target machine actively refused it."
	} else {
		errorString = "connect: connection refused"
	}

	// Immediately close the HTTP server so that the request fails.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	links, err := GetLinks(context.Background(), ts.URL, "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), errorString)
	require.Empty(t, links)
}

func TestGetLinksParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		badLinks := make(map[string]int)
		badLinks["browser_url"] = 10
		json.NewEncoder(w).Encode(badLinks)
	}))
	defer ts.Close()

	links, err := GetLinks(context.Background(), ts.URL, "test")
	require.EqualError(t, err, "json: cannot unmarshal number into Go struct field Links.browser_url of type string")
	require.Empty(t, links)
}
