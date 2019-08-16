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
		require.Nil(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "TestAgent/v1", r.UserAgent())
		require.Equal(t, "t=123,v1=hunter2", r.Header.Get("Stripe-Signature"))
		require.Equal(t, "{}", string(reqBody))
	}))
	defer ts.Close()

	rcvBody := ""
	rcvWebhookID := ""
	client := NewEndpointClient(
		ts.URL,
		false,
		[]string{"*"},
		&EndpointConfig{
			ResponseHandler: EndpointResponseHandlerFunc(func(webhookID string, resp *http.Response) {
				buf, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)

				rcvBody = string(buf)
				rcvWebhookID = webhookID

				wg.Done()
			}),
		},
	)

	webhookID := "wh_123"
	payload := "{}"
	headers := map[string]string{
		"User-Agent":       "TestAgent/v1",
		"Stripe-Signature": "t=123,v1=hunter2",
	}

	err := client.Post(webhookID, payload, headers)

	wg.Wait()

	require.Nil(t, err)
	require.Equal(t, "OK!", rcvBody)
	require.Equal(t, "wh_123", rcvWebhookID)
}
