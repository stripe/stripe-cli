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

const workbenchBlueprintsPath = "/v1/_unstable/workbench/blueprints"

// BlueprintRepository is the API-backed source of Workbench blueprints.
type BlueprintRepository interface {
	List(context.Context) ([]WorkbenchBlueprintSummary, error)
	Retrieve(context.Context, string) (*WorkbenchBlueprint, error)
}

// APIKeyProvider supplies the configured Stripe API key for a mode.
type APIKeyProvider interface {
	GetAPIKey(livemode bool) (string, error)
}

// WorkbenchClient loads Workbench blueprints with the configured test-mode key.
type WorkbenchClient struct {
	apiBaseURL string
	profile    APIKeyProvider
	httpClient *http.Client
}

// NewWorkbenchClient creates an API-backed blueprint repository.
func NewWorkbenchClient(profile APIKeyProvider, apiBaseURL string, httpClient *http.Client) *WorkbenchClient {
	if apiBaseURL == "" {
		apiBaseURL = stripe.DefaultAPIBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &WorkbenchClient{
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
		profile:    profile,
		httpClient: httpClient,
	}
}

func (c *WorkbenchClient) List(ctx context.Context) ([]WorkbenchBlueprintSummary, error) {
	var response struct {
		Data []WorkbenchBlueprintSummary `json:"data"`
	}
	if err := c.get(ctx, workbenchBlueprintsPath, &response, nil); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *WorkbenchClient) Retrieve(ctx context.Context, blueprintKey string) (*WorkbenchBlueprint, error) {
	path := workbenchBlueprintsPath + "/" + url.PathEscape(blueprintKey)
	var blueprint WorkbenchBlueprint
	var raw json.RawMessage
	if err := c.get(ctx, path, &blueprint, &raw); err != nil {
		return nil, err
	}
	blueprint.raw = raw
	return &blueprint, nil
}

func (c *WorkbenchClient) get(ctx context.Context, path string, destination any, raw *json.RawMessage) error {
	if c.profile == nil {
		return fmt.Errorf("loading Workbench blueprints: no Stripe profile configured")
	}
	apiKey, err := c.profile.GetAPIKey(false)
	if err != nil {
		return fmt.Errorf("loading Workbench blueprints with the test-mode API key: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating Workbench blueprint request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Stripe-Version", requests.StripePreviewVersionHeaderValue)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting Workbench blueprints: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading Workbench blueprint response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeWorkbenchAPIError(resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("decoding Workbench blueprint response: %w", err)
	}
	if raw != nil {
		*raw = append((*raw)[:0], body...)
	}
	return nil
}

func decodeWorkbenchAPIError(status int, body []byte) error {
	var response struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err == nil && response.Error.Message != "" {
		return &WorkbenchAPIError{
			StatusCode: status,
			Message:    response.Error.Message,
			Type:       response.Error.Type,
			Code:       response.Error.Code,
		}
	}
	return &WorkbenchAPIError{
		StatusCode: status,
		Message:    strings.TrimSpace(string(body)),
	}
}

// WorkbenchAPIError is a non-success response from the blueprint API.
type WorkbenchAPIError struct {
	StatusCode int
	Message    string
	Type       string
	Code       string
}

func (e *WorkbenchAPIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("Workbench blueprint API returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("Workbench blueprint API returned status %d: %s", e.StatusCode, e.Message)
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

type WorkbenchBlueprintSummary struct {
	ID               string             `json:"id"`
	BlueprintType    string             `json:"blueprint_type"`
	BlueprintVersion int                `json:"blueprint_version"`
	Description      MessageDescriptor  `json:"description"`
	Key              string             `json:"key"`
	Metadata         BlueprintMetadata  `json:"metadata"`
	StepRefs         []WorkbenchStepRef `json:"step_refs"`
	TemplateVersion  int                `json:"template_version"`
	Title            MessageDescriptor  `json:"title"`
}

type BlueprintMetadata struct {
	Products []string `json:"products"`
}

type WorkbenchStepRef struct {
	StepKey     string `json:"step_key"`
	StepVersion int    `json:"step_version"`
}

type WorkbenchBlueprint struct {
	WorkbenchBlueprintDefinition
	Steps []WorkbenchStep `json:"steps"`
	raw   json.RawMessage
}

type WorkbenchBlueprintDefinition struct {
	WorkbenchBlueprintSummary
	BlueprintSettings []WorkbenchSettingGroup `json:"blueprint_settings"`
}

type WorkbenchStep struct {
	WorkbenchStepDefinition
	Nodes []WorkbenchBlueprintNode `json:"nodes"`
}

type WorkbenchStepDefinition struct {
	Key             string                  `json:"key"`
	StepVersion     int                     `json:"step_version"`
	TemplateVersion int                     `json:"template_version"`
	Title           MessageDescriptor       `json:"title"`
	Description     MessageDescriptor       `json:"description"`
	Required        bool                    `json:"required"`
	Settings        []WorkbenchSettingGroup `json:"settings"`
	Config          WorkbenchStepConfig     `json:"config"`
	Outputs         []WorkbenchStepOutput   `json:"outputs"`
}

type WorkbenchStepOutput struct {
	Name   string         `json:"name"`
	Source string         `json:"source"`
	Schema map[string]any `json:"schema"`
}

type WorkbenchStepConfig struct {
	Settings map[string]string `json:"settings"`
	Params   map[string]string `json:"params"`
}

type WorkbenchSettingGroup struct {
	Key      string           `json:"key"`
	Settings []WorkbenchField `json:"settings"`
}

type WorkbenchField struct {
	Name   string               `json:"name"`
	Schema WorkbenchFieldSchema `json:"schema"`
}

type WorkbenchFieldSchema struct {
	DefaultValue any `json:"default_value"`
}

type WorkbenchBlueprintNode struct {
	NodeType            NodeType                      `json:"node_type"`
	Key                 string                        `json:"key"`
	Title               MessageDescriptor             `json:"title"`
	Description         MessageDescriptor             `json:"description"`
	IsInformationalNode bool                          `json:"is_informational_node"`
	APIRequestDetails   *WorkbenchAPIRequestDetails   `json:"api_request_details"`
	AsyncHandlerDetails *WorkbenchAsyncHandlerDetails `json:"async_handler_details"`
	TestHelperDetails   *WorkbenchTestHelperDetails   `json:"test_helper_details"`
	UIComponentDetails  *WorkbenchUIComponentDetails  `json:"ui_component_details"`
}

type WorkbenchAPIRequestDetails struct {
	Fixture WorkbenchRequestFixture `json:"fixture"`
}

type WorkbenchAsyncHandlerDetails struct {
	Events []AsyncEvent `json:"events"`
}

type WorkbenchTestHelperDetails struct {
	Requests []WorkbenchRequestFixture `json:"requests"`
}

type WorkbenchUIComponentDetails struct {
	ConfiguredDetails   []WorkbenchUIConfiguredDetails `json:"configured_details,omitempty"`
	Display             string                         `json:"display"`
	DisplayComponentRef *UIComponentReference          `json:"display_component_ref"`
	StripeElementRef    map[string]any                 `json:"stripe_element_ref"`
	Options             []WorkbenchUIOption            `json:"options"`
}

type WorkbenchUIConfiguredDetails struct {
	ConfigValue         map[string]string     `json:"config_value"`
	Display             string                `json:"display"`
	DisplayComponentRef *UIComponentReference `json:"display_component_ref"`
	StripeElementRef    map[string]any        `json:"stripe_element_ref"`
	Options             []WorkbenchUIOption   `json:"options"`
}

type WorkbenchUIOption struct {
	Type     string                    `json:"type"`
	Title    MessageDescriptor         `json:"title"`
	Link     string                    `json:"link"`
	Requests []WorkbenchRequestFixture `json:"requests"`
}

type WorkbenchRequestFixture struct {
	Key               string                       `json:"key"`
	Method            string                       `json:"method"`
	Path              string                       `json:"path"`
	Headers           map[string]string            `json:"headers"`
	Params            map[string]any               `json:"params"`
	HiddenParams      map[string]any               `json:"hidden_params"`
	ConfiguredDetails []WorkbenchConfiguredDetails `json:"configured_details,omitempty"`
	ExpectedErrorType string                       `json:"expected_error_type,omitempty"`
	ProcessingDetails *APIProcessingDetails        `json:"processing_details"`
	RegenerateEnv     bool                         `json:"regenerate_env"`
}

type WorkbenchConfiguredDetails struct {
	ConfigValue       map[string]string `json:"config_value"`
	Headers           map[string]string `json:"headers"`
	Params            map[string]any    `json:"params"`
	HiddenParams      map[string]any    `json:"hidden_params"`
	ExpectedErrorType string            `json:"expected_error_type"`
}
