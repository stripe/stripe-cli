package coop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const blueprintsPath = "/v1/_unstable/workbench/blueprints"

// BlueprintRepository is an API-backed source of blueprints.
type BlueprintRepository interface {
	List(context.Context) ([]BlueprintSummary, error)
	Retrieve(context.Context, string) (*Blueprint, error)
}

// APIKeyProvider supplies the configured Stripe API key for a mode.
type APIKeyProvider interface {
	GetAPIKey(livemode bool) (string, error)
}

// BlueprintClient loads blueprints with the configured test-mode key.
type BlueprintClient struct {
	apiBaseURL string
	profile    APIKeyProvider
	httpClient *http.Client
}

// NewBlueprintClient creates an API-backed blueprint repository.
func NewBlueprintClient(profile APIKeyProvider, apiBaseURL string, httpClient *http.Client) *BlueprintClient {
	if apiBaseURL == "" {
		apiBaseURL = stripe.DefaultAPIBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &BlueprintClient{
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
		profile:    profile,
		httpClient: httpClient,
	}
}

func (c *BlueprintClient) List(ctx context.Context) ([]BlueprintSummary, error) {
	var response struct {
		Data []BlueprintSummary `json:"data"`
	}
	if err := c.get(ctx, blueprintsPath, &response, nil); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *BlueprintClient) Retrieve(ctx context.Context, blueprintKey string) (*Blueprint, error) {
	path := blueprintsPath + "/" + url.PathEscape(blueprintKey)
	var blueprint Blueprint
	var raw json.RawMessage
	if err := c.get(ctx, path, &blueprint, &raw); err != nil {
		return nil, err
	}
	blueprint.raw = raw
	return &blueprint, nil
}

func (c *BlueprintClient) get(ctx context.Context, path string, destination any, raw *json.RawMessage) error {
	if c.profile == nil {
		return fmt.Errorf("loading blueprints: no Stripe profile configured")
	}
	apiKey, err := c.profile.GetAPIKey(false)
	if err != nil {
		return fmt.Errorf("loading blueprints with the test-mode API key: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating blueprint request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Stripe-Version", requests.StripePreviewVersionHeaderValue)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting blueprints: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading blueprint response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeBlueprintAPIError(resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("decoding blueprint response: %w", err)
	}
	if raw != nil {
		*raw = append((*raw)[:0], body...)
	}
	return nil
}

func decodeBlueprintAPIError(status int, body []byte) error {
	var response struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err == nil && response.Error.Message != "" {
		return &BlueprintAPIError{
			StatusCode: status,
			Message:    response.Error.Message,
			Type:       response.Error.Type,
			Code:       response.Error.Code,
		}
	}
	return &BlueprintAPIError{
		StatusCode: status,
		Message:    strings.TrimSpace(string(body)),
	}
}

// BlueprintAPIError is a non-success response from the blueprint API.
type BlueprintAPIError struct {
	StatusCode int
	Message    string
	Type       string
	Code       string
}

func (e *BlueprintAPIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("blueprint API returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("blueprint API returned status %d: %s", e.StatusCode, e.Message)
}

type MessageDescriptor struct {
	DefaultMessage string `json:"default_message"`
}

// UnmarshalJSON also accepts titles stored by earlier co-op sessions.
func (m *MessageDescriptor) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &m.DefaultMessage)
	}
	type descriptor MessageDescriptor
	return json.Unmarshal(data, (*descriptor)(m))
}

type BlueprintSummary struct {
	ID               string             `json:"id"`
	BlueprintType    string             `json:"blueprint_type"`
	BlueprintVersion int                `json:"blueprint_version"`
	Description      MessageDescriptor  `json:"description"`
	Key              string             `json:"key"`
	Metadata         BlueprintMetadata  `json:"metadata"`
	StepRefs         []BlueprintStepRef `json:"step_refs"`
	TemplateVersion  int                `json:"template_version"`
	Title            MessageDescriptor  `json:"title"`
}

type BlueprintMetadata struct {
	Products []string `json:"products"`
}

type BlueprintStepRef struct {
	StepKey     string            `json:"step_key"`
	StepVersion int               `json:"step_version"`
	Settings    map[string]string `json:"settings"`
	Params      map[string]string `json:"params"`
}

type Blueprint struct {
	BlueprintDefinition
	Steps []BlueprintStep `json:"steps"`
	raw   json.RawMessage
}

type BlueprintDefinition struct {
	BlueprintSummary
	BlueprintSettings []BlueprintSettingGroup `json:"blueprint_settings"`
	BlueprintParams   []BlueprintParamGroup   `json:"blueprint_params"`
}

type BlueprintStep struct {
	BlueprintStepDefinition
	Nodes []BlueprintNode `json:"nodes"`
}

type BlueprintStepDefinition struct {
	Key             string                  `json:"key"`
	StepVersion     int                     `json:"step_version"`
	TemplateVersion int                     `json:"template_version"`
	Title           MessageDescriptor       `json:"title"`
	Description     MessageDescriptor       `json:"description"`
	Required        bool                    `json:"required"`
	IsIncluded      any                     `json:"is_included"`
	Settings        map[string]string       `json:"settings"`
	SettingsSchema  []BlueprintSettingGroup `json:"settings_schema"`
	Params          map[string]string       `json:"params"`
	ParamsSchema    []BlueprintParamGroup   `json:"params_schema"`
	Outputs         []BlueprintStepOutput   `json:"outputs"`
}

type BlueprintStepOutput struct {
	Name   string         `json:"name"`
	Source string         `json:"source"`
	Schema map[string]any `json:"schema"`
}

type BlueprintSettingGroup struct {
	Key      string           `json:"key"`
	Settings []BlueprintField `json:"settings"`
}

type BlueprintParamGroup struct {
	Key    string           `json:"key"`
	Params []BlueprintField `json:"params"`
}

type BlueprintField struct {
	Name   string               `json:"name"`
	Schema BlueprintFieldSchema `json:"schema"`
}

type BlueprintFieldSchema struct {
	DefaultValue any `json:"default_value"`
}

type BlueprintNode struct {
	NodeType            NodeType                      `json:"node_type"`
	Key                 string                        `json:"key"`
	Title               MessageDescriptor             `json:"title"`
	Description         MessageDescriptor             `json:"description"`
	IsIncluded          any                           `json:"is_included"`
	IsInformationalNode bool                          `json:"is_informational_node"`
	APIRequestDetails   *BlueprintAPIRequestDetails   `json:"api_request_details"`
	AsyncHandlerDetails *BlueprintAsyncHandlerDetails `json:"async_handler_details"`
	TestHelperDetails   *BlueprintTestHelperDetails   `json:"test_helper_details"`
	UIComponentDetails  *BlueprintUIComponentDetails  `json:"ui_component_details"`
}

type BlueprintAPIRequestDetails struct {
	Fixture BlueprintRequestFixture `json:"fixture"`
}

type BlueprintAsyncHandlerDetails struct {
	Events []AsyncEvent `json:"events"`
}

type BlueprintTestHelperDetails struct {
	Requests []BlueprintRequestFixture `json:"requests"`
}

type BlueprintUIComponentDetails struct {
	ConfiguredDetails   []BlueprintUIConfiguredDetails `json:"configured_details,omitempty"`
	Display             string                         `json:"display"`
	DisplayComponentRef *UIComponentReference          `json:"display_component_ref"`
	StripeElementRef    map[string]any                 `json:"stripe_element_ref"`
	Options             []BlueprintUIOption            `json:"options"`
}

type BlueprintUIConfiguredDetails struct {
	ConfigValue         map[string]string     `json:"config_value"`
	Display             string                `json:"display"`
	DisplayComponentRef *UIComponentReference `json:"display_component_ref"`
	StripeElementRef    map[string]any        `json:"stripe_element_ref"`
	Options             []BlueprintUIOption   `json:"options"`
}

type BlueprintUIOption struct {
	Type     string                    `json:"type"`
	Title    MessageDescriptor         `json:"title"`
	Link     string                    `json:"link"`
	Requests []BlueprintRequestFixture `json:"requests"`
}

type BlueprintRequestFixture struct {
	Key               string                       `json:"key"`
	Method            string                       `json:"method"`
	Path              string                       `json:"path"`
	Headers           map[string]string            `json:"headers"`
	Params            map[string]any               `json:"params"`
	HiddenParams      map[string]any               `json:"hidden_params"`
	ConfiguredDetails []BlueprintConfiguredDetails `json:"configured_details,omitempty"`
	ExpectedErrorType string                       `json:"expected_error_type,omitempty"`
	ProcessingDetails *APIProcessingDetails        `json:"processing_details"`
	RegenerateEnv     bool                         `json:"regenerate_env"`
}

type BlueprintConfiguredDetails struct {
	ConfigValue       map[string]string `json:"config_value"`
	Headers           map[string]string `json:"headers"`
	Params            map[string]any    `json:"params"`
	HiddenParams      map[string]any    `json:"hidden_params"`
	ExpectedErrorType string            `json:"expected_error_type"`
}
