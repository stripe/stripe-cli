package requests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"

	"github.com/spf13/cobra"
)

// RequestParameters captures the structure of the parameters that can be sent to Stripe
type RequestParameters struct {
	data          []string
	expand        []string
	startingAfter string
	endingBefore  string
	idempotency   string
	limit         string
	version       string
	stripeAccount string
}

// AppendData appends data to the request parameters.
func (r *RequestParameters) AppendData(data []string) {
	r.data = append(r.data, data...)
}

// AppendExpand appends fields to the expand parameter.
func (r *RequestParameters) AppendExpand(fields []string) {
	r.expand = append(r.expand, fields...)
}

// SetIdempotency sets the value for the `Idempotency-Key` header.
func (r *RequestParameters) SetIdempotency(value string) {
	r.idempotency = value
}

// SetStripeAccount sets the value for the `Stripe-Account` header.
func (r *RequestParameters) SetStripeAccount(value string) {
	r.stripeAccount = value
}

// SetVersion sets the value for the `Stripe-Version` header.
func (r *RequestParameters) SetVersion(value string) {
	r.version = value
}

// RequestError captures the response of the request that resulted in an error
type RequestError struct {
	msg        string
	StatusCode int
	ErrorType  string
	ErrorCode  string
	Body       interface{} // the raw response body
}

func (e RequestError) Error() string {
	return fmt.Sprintf("%s, status=%d, body=%s", e.msg, e.StatusCode, e.Body)
}

// IsAPIKeyExpiredError returns true if the provided error was caused by a
// request returning an `api_key_expired` error code.
//
// See https://stripe.com/docs/error-codes/api-key-expired.
func IsAPIKeyExpiredError(err error) bool {
	var reqErr RequestError
	if errors.As(err, &reqErr) {
		return reqErr.StatusCode == 401 && reqErr.ErrorCode == "api_key_expired"
	}
	return false
}

// Base encapsulates the required information needed to make requests to the API
type Base struct {
	Cmd *cobra.Command

	Method  string
	Profile *config.Profile

	Parameters RequestParameters

	// SuppressOutput is used by `trigger` to hide output
	SuppressOutput bool

	DarkStyle bool

	APIBaseURL string

	Livemode bool

	autoConfirm bool
	showHeaders bool
}

var confirmationCommands = map[string]bool{http.MethodDelete: true}

// RunRequestsCmd is the interface exposed for the CLI to run network requests through
func (rb *Base) RunRequestsCmd(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("this command only supports one argument. Run with the --help flag to see usage and examples")
	}

	if len(args) == 0 {
		return nil
	}

	confirmed, err := rb.confirmCommand()
	if err != nil {
		return err
	} else if !confirmed {
		fmt.Println("Exiting without execution. User did not confirm the command.")
		return nil
	}

	apiKey, err := rb.Profile.GetAPIKey(rb.Livemode)
	if err != nil {
		return err
	}

	path, err := createOrNormalizePath(args[0])
	if err != nil {
		return err
	}

	_, err = rb.MakeRequest(cmd.Context(), apiKey, path, &rb.Parameters, false)

	return err
}

// InitFlags initialize shared flags for all requests commands
func (rb *Base) InitFlags() {
	if rb.Cmd.Flags().Lookup("confirm") == nil {
		rb.Cmd.Flags().BoolVarP(&rb.autoConfirm, "confirm", "c", false, "Skip the warning prompt and automatically confirm the command being entered")
	}

	rb.Cmd.Flags().StringArrayVarP(&rb.Parameters.data, "data", "d", []string{}, "Data for the API request")
	rb.Cmd.Flags().StringArrayVarP(&rb.Parameters.expand, "expand", "e", []string{}, "Response attributes to expand inline")
	rb.Cmd.Flags().StringVarP(&rb.Parameters.idempotency, "idempotency", "i", "", "Set the idempotency key for the request, prevents replaying the same requests within 24 hours")
	rb.Cmd.Flags().StringVarP(&rb.Parameters.version, "stripe-version", "v", "", "Set the Stripe API version to use for your request")
	rb.Cmd.Flags().StringVar(&rb.Parameters.stripeAccount, "stripe-account", "", "Set a header identifying the connected account")
	rb.Cmd.Flags().BoolVarP(&rb.showHeaders, "show-headers", "s", false, "Show response headers")
	rb.Cmd.Flags().BoolVar(&rb.Livemode, "live", false, "Make a live request (default: test)")
	rb.Cmd.Flags().BoolVar(&rb.DarkStyle, "dark-style", false, "Use a darker color scheme better suited for lighter command-lines")

	// Conditionally add flags for GET requests. I'm doing it here to keep `limit`, `start_after` and `ending_before` unexported
	if rb.Method == http.MethodGet {
		if rb.Cmd.Flags().Lookup("limit") == nil {
			rb.Cmd.Flags().StringVarP(&rb.Parameters.limit, "limit", "l", "", "How many objects to be returned, between 1 and 100 (default is 10)")
		}

		if rb.Cmd.Flags().Lookup("starting-after") == nil {
			rb.Cmd.Flags().StringVarP(&rb.Parameters.startingAfter, "starting-after", "a", "", "Retrieve the next page in the list. This is a cursor for pagination and should be an object ID")
		}

		if rb.Cmd.Flags().Lookup("ending-before") == nil {
			rb.Cmd.Flags().StringVarP(&rb.Parameters.endingBefore, "ending-before", "b", "", "Retrieve the previous page in the list. This is a cursor for pagination and should be an object ID")
		}
	}

	// Hidden configuration flags, useful for dev/debugging
	rb.Cmd.Flags().StringVar(&rb.APIBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	rb.Cmd.Flags().MarkHidden("api-base") // #nosec G104
}

// MakeMultiPartRequest will make a multipart/form-data request to the Stripe API with the specific variables given to it.
// Similar to making a multipart request using curl, add the local filepath to params arg with @ prefix.
// e.g. params.AppendData([]string{"photo=@/path/to/local/file.png"})
func (rb *Base) MakeMultiPartRequest(ctx context.Context, apiKey, path string, params *RequestParameters, errOnStatus bool) ([]byte, error) {
	reqBody, contentType, err := rb.buildMultiPartRequest(params)
	if err != nil {
		return []byte{}, err
	}

	configure := func(req *http.Request) {
		req.Header.Set("Content-Type", contentType)
	}

	return rb.performRequest(ctx, apiKey, path, params, reqBody.String(), errOnStatus, configure)
}

// MakeRequest will make a request to the Stripe API with the specific variables given to it
func (rb *Base) MakeRequest(ctx context.Context, apiKey, path string, params *RequestParameters, errOnStatus bool) ([]byte, error) {
	data, err := rb.buildDataForRequest(params)
	if err != nil {
		return []byte{}, err
	}

	return rb.performRequest(ctx, apiKey, path, params, data, errOnStatus, nil)
}

func (rb *Base) performRequest(ctx context.Context, apiKey, path string, params *RequestParameters, data string, errOnStatus bool, additionalConfigure func(req *http.Request)) ([]byte, error) {
	parsedBaseURL, err := url.Parse(rb.APIBaseURL)
	if err != nil {
		return []byte{}, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  apiKey,
		Verbose: rb.showHeaders,
	}

	configure := func(req *http.Request) {
		rb.setIdempotencyHeader(req, params)
		rb.setStripeAccountHeader(req, params)
		rb.setVersionHeader(req, params)
		if additionalConfigure != nil {
			additionalConfigure(req)
		}
	}

	resp, err := client.PerformRequest(ctx, rb.Method, path, data, configure)

	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 401 || (errOnStatus && resp.StatusCode >= 300) {
		requestError := compileRequestError(body, resp.StatusCode)
		return []byte{}, requestError
	}

	if !rb.SuppressOutput {
		if err != nil {
			return []byte{}, err
		}

		result := ansi.ColorizeJSON(string(body), rb.DarkStyle, os.Stdout)
		fmt.Print(result)
	}

	return body, nil
}

func compileRequestError(body []byte, statusCode int) RequestError {
	type requestErrorContent struct {
		Code string `json:"code"`
		Type string `json:"type"`
	}

	type requestErrorBody struct {
		Content requestErrorContent `json:"error"`
	}

	var errorBody requestErrorBody
	json.Unmarshal(body, &errorBody)
	return RequestError{
		msg:        "Request failed",
		StatusCode: statusCode,
		ErrorType:  errorBody.Content.Type,
		ErrorCode:  errorBody.Content.Code,
		Body:       string(body),
	}
}

// Confirm calls the confirmCommand() function, triggering the confirmation process
func (rb *Base) Confirm() (bool, error) {
	return rb.confirmCommand()
}

// Note: We converted to using two arrays to track keys and values, with our own
// implementation of Go's url.Values Encode function due to our query parameters being
// order sensitive for API requests involving arrays like `items` for `/v1/orders`.
// Go's url.Values uses Go's map, which jumbles the key ordering, and their Encode
// implementation sorts keys by alphabetical order, but this doesn't work for us since
// some API endpoints have required parameter ordering. Yes, this is hacky, but it works.
func (rb *Base) buildDataForRequest(params *RequestParameters) (string, error) {
	keys := []string{}
	values := []string{}

	if len(params.data) > 0 || len(params.expand) > 0 {
		for _, datum := range params.data {
			splitDatum := strings.SplitN(datum, "=", 2)

			if len(splitDatum) < 2 {
				return "", fmt.Errorf("Invalid data argument: %s", datum)
			}

			keys = append(keys, splitDatum[0])
			values = append(values, splitDatum[1])
		}

		for _, datum := range params.expand {
			keys = append(keys, "expand[]")
			values = append(values, datum)
		}
	}

	if rb.Method == http.MethodGet {
		if params.limit != "" {
			keys = append(keys, "limit")
			values = append(values, params.limit)
		}

		if params.startingAfter != "" {
			keys = append(keys, "starting_after")
			values = append(values, params.startingAfter)
		}

		if params.endingBefore != "" {
			keys = append(keys, "ending_before")
			values = append(values, params.endingBefore)
		}
	}

	return encode(keys, values), nil
}

func (rb *Base) buildMultiPartRequest(params *RequestParameters) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	mp := multipart.NewWriter(&body)
	defer mp.Close()
	for _, datum := range params.data {
		splitDatum := strings.SplitN(datum, "=", 2)

		if len(splitDatum) < 2 {
			return nil, "", fmt.Errorf("Invalid data argument: %s", datum)
		}

		key := splitDatum[0]
		val := splitDatum[1]

		// Param values that are prefixed with @ will be parsed as a form file
		if strings.HasPrefix(val, "@") {
			val = val[1:]
			file, err := os.Open(val)
			if err != nil {
				return nil, "", err
			}
			defer file.Close()
			part, err := mp.CreateFormFile(key, val)
			if err != nil {
				return nil, "", err
			}
			io.Copy(part, file)
		} else {
			mp.WriteField(key, val)
		}
	}

	return &body, mp.FormDataContentType(), nil
}

// encode creates a url encoded string with the request parameters
func encode(keys []string, values []string) string {
	var buf strings.Builder

	for i := range keys {
		key := keys[i]
		value := values[i]

		keyEscaped := url.QueryEscape(key)

		// Don't use strict form encoding by changing the square bracket
		// control characters back to their literals. This is fine by the
		// server, and makes these parameter strings easier to read.
		keyEscaped = strings.ReplaceAll(keyEscaped, "%5B", "[")
		keyEscaped = strings.ReplaceAll(keyEscaped, "%5D", "]")

		if buf.Len() > 0 {
			buf.WriteByte('&')
		}

		buf.WriteString(keyEscaped)
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(value))
	}

	return buf.String()
}

func (rb *Base) setIdempotencyHeader(request *http.Request, params *RequestParameters) {
	if params.idempotency != "" {
		request.Header.Set("Idempotency-Key", params.idempotency)

		if rb.Method == http.MethodGet || rb.Method == http.MethodDelete {
			warning := fmt.Sprintf(
				"Warning: sending an idempotency key with a %s request has no effect and should be avoided, as %s requests are idempotent by definition.",
				rb.Method,
				rb.Method,
			)
			fmt.Println(warning)
		}
	}
}

func (rb *Base) setVersionHeader(request *http.Request, params *RequestParameters) {
	if params.version != "" {
		request.Header.Set("Stripe-Version", params.version)
	}
}

func (rb *Base) setStripeAccountHeader(request *http.Request, params *RequestParameters) {
	if params.stripeAccount != "" {
		request.Header.Set("Stripe-Account", params.stripeAccount)
	}
}

func (rb *Base) confirmCommand() (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	return rb.getUserConfirmation(reader)
}

func (rb *Base) getUserConfirmation(reader *bufio.Reader) (bool, error) {
	if _, needsConfirmation := confirmationCommands[rb.Method]; needsConfirmation && !rb.autoConfirm {
		confirmationPrompt := fmt.Sprintf("Are you sure you want to perform the command: %s?\nEnter 'yes' to confirm: ", rb.Method)
		fmt.Print(confirmationPrompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		// remove whitespace from either side of the input, as ReadString returns with \n at the end
		input = strings.ToLower(strings.Trim(input, " \r\n"))

		return strings.Compare(input, "yes") == 0, nil
	}

	// Always confirm the command if it does not require explicit user confirmation
	return true, nil
}

func createOrNormalizePath(arg string) (string, error) {
	if idRegex.Match([]byte(arg)) {
		matches := idRegex.FindStringSubmatch(arg)

		if path, ok := idURLMap[matches[1]]; ok {
			return path + arg, nil
		}

		return "", fmt.Errorf("Unrecognized object id: %s", arg)
	}

	return normalizePath(arg), nil
}

func normalizePath(path string) string {
	if strings.HasPrefix(path, "/v1/") {
		return path
	}

	if strings.HasPrefix(path, "v1/") {
		return "/" + path
	}

	if strings.HasPrefix(path, "/") {
		return "/v1" + path
	}

	return "/v1/" + path
}
