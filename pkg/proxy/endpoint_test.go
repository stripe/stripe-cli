package proxy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientHandler(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))

		reqBody, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "TestAgent/v1", r.UserAgent())
		require.Equal(t, "t=123,v1=hunter2", r.Header.Get("Stripe-Signature"))

		require.Equal(t, "hostname", r.Host)
		require.Equal(t, "customHeaderValue", r.Header.Get("customHeader"))
		require.Equal(t, "customHeaderValue 2", r.Header.Get("customHeader2"))
		require.Equal(t, "", r.Header.Get("emptyHeader"))
		require.Equal(t, "tab", r.Header.Get("removeControlCharacters"))

		require.Equal(t, "{}", string(reqBody))
	}))
	defer ts.Close()

	rcvCtx := eventContext{}
	rcvBody := ""
	rcvForwardURL := ""
	client := NewEndpointClient(
		ts.URL,
		[]string{" Host:       hostname", "customHeader:customHeaderValue", "customHeader2:       customHeaderValue 2",
			"emptyHeader:", ":", "::", "removeControlCharacters:	tab"}, // custom headers
		false,
		[]string{"*"},
		&EndpointConfig{
			ResponseHandler: EndpointResponseHandlerFunc(func(evtCtx eventContext, forwardURL string, resp *http.Response) {
				buf, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)

				rcvCtx = evtCtx
				rcvBody = string(buf)
				rcvForwardURL = forwardURL

				wg.Done()
			}),
		},
	)

	evt := &StripeEvent{
		ID: "evt_123",
	}
	evtCtx := eventContext{
		webhookID:             "wh_123",
		webhookConversationID: "wc_123",
		event:                 evt,
	}
	payload := "{}"
	headers := map[string]string{
		"User-Agent":       "TestAgent/v1",
		"Stripe-Signature": "t=123,v1=hunter2",
	}

	err := client.Post(evtCtx, payload, headers)
	require.NoError(t, err)

	wg.Wait()

	require.Equal(t, "OK!", rcvBody)
	require.Equal(t, ts.URL, rcvForwardURL)
	require.Equal(t, "wh_123", rcvCtx.webhookID)
	require.Equal(t, "wc_123", rcvCtx.webhookConversationID)
	require.Equal(t, "evt_123", rcvCtx.event.ID)
}

func TestClientHandler_Redirects(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	n := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)

		n++
		if n == 1 {
			http.Redirect(w, r, "/foo", http.StatusMovedPermanently)
		} else {
			require.FailNow(t, "Received more than one request")
		}
	}))

	defer ts.Close()

	client := NewEndpointClient(
		ts.URL,
		[]string{},
		false,
		[]string{"*"},
		&EndpointConfig{
			ResponseHandler: EndpointResponseHandlerFunc(func(evtCtx eventContext, forwardURL string, resp *http.Response) {
				require.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
				wg.Done()
			}),
		},
	)

	evt := &StripeEvent{
		ID: "evt_123",
	}
	evtCtx := eventContext{
		webhookID:             "wh_123",
		webhookConversationID: "wc_123",
		event:                 evt,
	}
	payload := "{}"
	headers := map[string]string{
		"User-Agent":       "TestAgent/v1",
		"Stripe-Signature": "t=123,v1=hunter2",
	}

	err := client.Post(evtCtx, payload, headers)
	require.NoError(t, err)

	wg.Wait()
}
