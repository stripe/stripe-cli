package p400

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Metadata belongs to the Stripe P400 reader object
// we don't currently make use of it directly for quickstart
type Metadata struct{}

// Reader represents the Stripe P400 reader object shape
type Reader struct {
	ID              string   `json:"id"`
	Object          string   `json:"object"`
	DeviceSwVersion string   `json:"device_software_version"`
	DeviceType      string   `json:"device_type"`
	IPAddress       string   `json:"ip_address"`
	Label           string   `json:"label"`
	Livemode        bool     `json:"livemode"`
	Location        string   `json:"location"`
	SerialNumber    string   `json:"serial_number"`
	Status          string   `json:"status"`
	Metadata        Metadata `json:"metadata"`
	BaseURL         string   `json:"base_url"`
}

type readersResponse struct {
	Error   string   `json:"error"`
	Object  string   `json:"object"`
	URL     string   `json:"url"`
	HasMore bool     `json:"has_more"`
	Data    []Reader `json:"data"`
}

type createPaymentIntentResponse struct {
	ID string `json:"id"`
}

type startNewRPCSessionResponse struct {
	SDKRPCSessionToken string `json:"sdk_rpc_session_token"`
}

type getConnectionTokenResponse struct {
	Secret string `json:"secret"`
}

// DiscoverReaders calls the Stripe API to get a list of currently registered P400 readers on the account
// it returns a map of Reader types
func DiscoverReaders(ctx context.Context, tsCtx TerminalSessionContext) ([]Reader, error) {
	parsedBaseURL, err := url.Parse(stripe.DefaultAPIBaseURL)

	if err != nil {
		return nil, err
	}

	var readersList []Reader

	if err != nil {
		return readersList, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  tsCtx.PstToken,
		Verbose: false,
	}

	res, err := client.PerformRequest(ctx, http.MethodGet, stripeTerminalReadersPath, "", nil)

	if err != nil {
		return readersList, err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 400 && res.StatusCode < 500 {
			err = ErrStripeForbiddenResponse
		} else {
			fmt.Println(res)
			fmt.Println(res.StatusCode)
			err = ErrStripeGenericResponse
		}

		return readersList, err
	}

	var result readersResponse

	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&result)

	readersList = result.Data

	return readersList, nil
}

// StartNewRPCSession calls the Stripe API for a new RPC session token for interacting with a P400 reader
// returns a session token when successful
func StartNewRPCSession(tsCtx TerminalSessionContext) (string, error) {
	httpclient := http.Client{}
	parsedBaseURL, err := url.Parse(stripe.DefaultAPIBaseURL)

	if err != nil {
		return "", err
	}

	stripeTerminalRPCSessionURL := fmt.Sprintf("%s%s", parsedBaseURL, rpcSessionPath)

	data := url.Values{}
	data.Set("pos_device_info[device_class]", tsCtx.DeviceInfo.DeviceClass)
	data.Set("pos_device_info[device_uuid]", tsCtx.DeviceInfo.DeviceUUID)
	data.Set("pos_device_info[host_os_version]", tsCtx.DeviceInfo.HostOSVersion)
	data.Set("pos_device_info[hardware_model][pos_info][description]", tsCtx.DeviceInfo.HardwareModel.POSInfo.Description)
	data.Set("pos_device_info[app_model][app_id]", tsCtx.DeviceInfo.AppModel.AppID)
	data.Set("pos_device_info[app_model][app_version]", tsCtx.DeviceInfo.AppModel.AppVersion)

	encodedURLData := data.Encode()
	urlDataBuffer := bytes.NewBuffer([]byte(encodedURLData))

	request, err := http.NewRequest("POST", stripeTerminalRPCSessionURL, urlDataBuffer)

	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tsCtx.PstToken))

	res, err := httpclient.Do(request)

	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 400 && res.StatusCode < 500 {
			err = ErrStripeForbiddenResponse
		} else {
			err = ErrStripeGenericResponse
		}

		return "", err
	}

	var result startNewRPCSessionResponse

	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&result)

	sessionToken := result.SDKRPCSessionToken

	return sessionToken, nil
}

// GetNewConnectionToken calls the Stripe API and requests a new connection token in order to start a new reader session
// it returns the connection token when successful
func GetNewConnectionToken(ctx context.Context, tsCtx TerminalSessionContext) (string, error) {
	parsedBaseURL, err := url.Parse(stripe.DefaultAPIBaseURL)

	if err != nil {
		return "", err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  tsCtx.APIKey,
		Verbose: false,
	}

	res, err := client.PerformRequest(ctx, http.MethodPost, stripeTerminalConnectionTokensPath, "", nil)

	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 400 && res.StatusCode < 500 {
			err = ErrStripeForbiddenResponse
		} else {
			err = ErrStripeGenericResponse
		}

		return "", err
	}

	var result getConnectionTokenResponse

	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&result)

	pstToken := result.Secret

	return pstToken, nil
}

// CreatePaymentIntent calls the Stripe API to create a new Payment Intent in order to later attach a collected P400 payment to
// it returns the Payment Intent Id
func CreatePaymentIntent(ctx context.Context, tsCtx TerminalSessionContext) (string, error) {
	parsedBaseURL, err := url.Parse(stripe.DefaultAPIBaseURL)

	if err != nil {
		return "", err
	}

	amountStr := strconv.Itoa(tsCtx.Amount)

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  tsCtx.APIKey,
		Verbose: false,
	}

	data := url.Values{}
	data.Set("amount", amountStr)
	data.Set("currency", tsCtx.Currency)
	data.Set("payment_method_types[]", "card_present")
	data.Set("capture_method", "manual")
	data.Set("description", "Stripe CLI Test Payment")

	res, err := client.PerformRequest(ctx, http.MethodPost, stripeCreatePaymentIntentPath, data.Encode(), nil)

	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 400 && res.StatusCode < 500 {
			err = ErrStripeForbiddenResponse
		} else {
			err = ErrStripeGenericResponse
		}

		return "", err
	}

	var result createPaymentIntentResponse

	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&result)

	paymentIntentID := result.ID

	return paymentIntentID, nil
}

// CapturePaymentIntent manually captures the Payment Intent after a Payment Method is attached which is the required flow for collecting payments on the Terminal platform
func CapturePaymentIntent(ctx context.Context, tsCtx TerminalSessionContext) error {
	parsedBaseURL, err := url.Parse(stripe.DefaultAPIBaseURL)

	if err != nil {
		return err
	}

	stripeCapturePaymentIntentURL := fmt.Sprintf(stripeCapturePaymentIntentPath, tsCtx.PaymentIntentID)

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  tsCtx.APIKey,
		Verbose: false,
	}

	res, err := client.PerformRequest(ctx, http.MethodPost, stripeCapturePaymentIntentURL, "", nil)

	if err != nil {
		return ErrCapturePaymentIntentFailed
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 400 && res.StatusCode < 500 {
			err = ErrStripeForbiddenResponse
		} else {
			err = ErrStripeGenericResponse
		}

		return err
	}

	res.Body.Close()

	return nil
}

// RegisterReader calls the Stripe API to register a new P400 reader to an account
// it returns the IP address of the reader if successful
func RegisterReader(ctx context.Context, regcode string, tsCtx TerminalSessionContext) (Reader, error) {
	parsedBaseURL, err := url.Parse(stripe.DefaultAPIBaseURL)
	var result Reader

	if err != nil {
		return result, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  tsCtx.APIKey,
		Verbose: false,
	}

	data := url.Values{}
	data.Set("registration_code", regcode)

	res, err := client.PerformRequest(ctx, http.MethodPost, stripeTerminalRegisterPath, data.Encode(), nil)

	if err != nil {
		return result, err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 400 && res.StatusCode < 500 {
			err = ErrStripeForbiddenResponse
		} else {
			fmt.Println(res)
			fmt.Println(res.StatusCode)
			err = ErrStripeGenericResponse
		}

		return result, err
	}

	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&result)

	return result, nil
}
