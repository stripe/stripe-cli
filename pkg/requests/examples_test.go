package requests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Nil(t, err)
	assert.Equal(t, "test-id", resp["id"])
}

func TestBuildRequest(t *testing.T) {
	ex := Examples{
		APIVersion: "v1",
		SecretKey:  "secret-key",
	}

	req, params := ex.buildRequest("POST", []string{"foo=bar"})

	assert.Equal(t, []string{"foo=bar"}, params.data)
	assert.Equal(t, "POST", req.Method)
}

func TestChargeCaptured(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		SecretKey:  "secret-key",
	}

	err := ex.ChargeCaptured()
	assert.Nil(t, err)
}

func TestChargeFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		SecretKey:  "secret-key",
	}

	err := ex.ChargeFailed()
	assert.Nil(t, err)
}

func TestChargeSucceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		SecretKey:  "secret-key",
	}

	err := ex.ChargeSucceeded()
	assert.Nil(t, err)
}

func TestCustomerCreated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := jsonBytes()
		w.Write(data)
	}))
	defer ts.Close()

	ex := Examples{
		APIBaseURL: ts.URL,
		APIVersion: "v1",
		SecretKey:  "secret-key",
	}

	err := ex.CustomerCreated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.CustomerUpdated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.CustomerSourceCreated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.CustomerSourceUpdated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.CustomerSubscriptionUpdated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.InvoiceCreated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.InvoiceFinalized()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.InvoicePaymentSucceeded()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.InvoiceUpdated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.PaymentIntentCreated()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.PaymentIntentSucceeded()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.PaymentIntentFailed()
	assert.Nil(t, err)
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
		SecretKey:  "secret-key",
	}

	err := ex.PaymentMethodAttached()
	assert.Nil(t, err)
}
