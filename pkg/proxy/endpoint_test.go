package proxy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestClientHandler(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))

		reqBody, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)

		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "TestAgent/v1", r.UserAgent())
		assert.Equal(t, "t=123,v1=hunter2", r.Header.Get("Stripe-Signature"))
		assert.Equal(t, "{}", string(reqBody))
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
				assert.Nil(t, err)

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

	assert.Nil(t, err)
	assert.Equal(t, "OK!", rcvBody)
	assert.Equal(t, "wh_123", rcvWebhookID)
}
