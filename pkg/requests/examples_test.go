package requests

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func jsonBytes() []byte {
	type TestJSON struct {
		ID string `json:"id"`
	}
	data := TestJSON{"test-id"}
	bytes, _ := json.Marshal(data)
	return bytes
}

func TestParseResponse(t *testing.T) {
	bytes := jsonBytes()
	resp, err := parseResponse(bytes)
	require.Nil(t, err)
	require.Equal(t, "test-id", resp["id"])
}

func TestBuildRequest(t *testing.T) {
	ex := Examples{
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	req, params := ex.buildRequest(http.MethodPost, []string{"foo=bar"})

	require.Equal(t, []string{"foo=bar"}, params.data)
	require.Equal(t, http.MethodPost, req.Method)
}

func TestChargeCaptured(t *testing.T) {
	count := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.Nil(t, err)

		// Because it's 2 calls to server for this
		if count == 0 {
			require.NotEmpty(t, body)
			require.EqualValues(t, "amount=2000&currency=usd&capture=false&source=tok_visa", string(body))
			count++
		} else {
			require.Empty(t, body)
		}

		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.ChargeCaptured()
	require.NoError(t, err)
}

func TestChargeFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.Nil(t, err)
		require.NotEmpty(t, body)
		require.EqualValues(t, "amount=2000&currency=usd&source=tok_chargeDeclined", string(body))

		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.ChargeFailed()
	require.Nil(t, err)
}

func TestChargeSucceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.Nil(t, err)
		require.EqualValues(t, "amount=2000&currency=usd&source=tok_visa", string(body))

		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.ChargeSucceeded()
	require.Nil(t, err)
}

func TestCustomerCreated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.Nil(t, err)
		require.Empty(t, body)

		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.CustomerCreated()
	require.Nil(t, err)
}

func TestCustomerUpdated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.CustomerUpdated()
	require.Nil(t, err)
}

func TestCustomerSourceCreated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.CustomerSourceCreated()
	require.Nil(t, err)
}

func TestCustomerSourceUpdated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.CustomerSourceUpdated()
	require.Nil(t, err)
}

func TestCustomerSubscriptionUpdated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.CustomerSubscriptionUpdated()
	require.Nil(t, err)
}

func TestInvoiceCreated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.InvoiceCreated()
	require.Nil(t, err)
}

func TestInvoiceFinalized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.InvoiceFinalized()
	require.Nil(t, err)
}

func TestInvoicePaymentSucceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.InvoicePaymentSucceeded()
	require.Nil(t, err)
}

func TestInvoiceUpdated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.InvoiceUpdated()
	require.Nil(t, err)
}

func TestPaymentIntentCreated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.PaymentIntentCreated()
	require.Nil(t, err)
}

func TestPaymentIntentSucceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.PaymentIntentSucceeded()
	require.Nil(t, err)
}

func TestPaymentIntentFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.PaymentIntentFailed()
	require.Nil(t, err)
}

func TestPaymentMethodAttached(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		APIKey:     "secret-key",
	}

	err := ex.PaymentMethodAttached()
	require.Nil(t, err)
}
