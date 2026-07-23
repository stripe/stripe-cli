package coop

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func deriveReviewPrompt(node NodeDefinition) string {
	switch node.Type {
	case NodeAPIRequest:
		return fmt.Sprintf("Confirm %s sends %s %s with the configured headers and parameters, handles the response, and does not expose secret keys.", node.Title, strings.ToUpper(node.Request.Method), node.Request.Path)
	case NodeAsyncHandler:
		var eventTypes []string
		for _, event := range node.Events {
			eventTypes = append(eventTypes, event.EventType)
		}
		return fmt.Sprintf("Cause %s to occur and confirm %s handles the event safely, verifies it as appropriate, and produces the expected application outcome.", strings.Join(eventTypes, " or "), node.Title)
	case NodeTestHelper:
		return fmt.Sprintf("Run the test flow for %s and confirm every configured request succeeds and produces the expected state.", node.Title)
	case NodeUIComponent:
		return fmt.Sprintf("Open %s and confirm the user-facing flow works, all links and options behave as described, and no secret key is exposed.", node.Title)
	default:
		return fmt.Sprintf("Confirm %s is complete and produces the expected observable result.", node.Title)
	}
}

func deriveReviewCommand(node NodeDefinition) string {
	if node.Type == NodeAPIRequest && node.Request != nil {
		return stripeRequestCommand(*node.Request)
	}
	if node.Type == NodeTestHelper && len(node.TestRequests) > 0 {
		return stripeRequestCommand(node.TestRequests[0].APIRequest)
	}
	return ""
}

// stripeRequestCommand returns an executable command only when every part of
// the request can be represented by Stripe CLI flags without runtime values.
func stripeRequestCommand(request APIRequest) string {
	method := strings.ToLower(request.Method)
	switch method {
	case "get", "post", "delete":
	default:
		return ""
	}
	if containsUnresolvedValue(request.Path) {
		return ""
	}

	args := []string{"stripe", method, shellQuoteArgument(request.Path)}
	headerNames := make([]string, 0, len(request.Headers))
	for name := range request.Headers {
		headerNames = append(headerNames, name)
	}
	sort.Strings(headerNames)
	for _, name := range headerNames {
		value := request.Headers[name]
		if containsUnresolvedValue(value) {
			return ""
		}
		switch strings.ToLower(name) {
		case "stripe-account":
			args = append(args, "--stripe-account", shellQuoteArgument(value))
		case "stripe-context":
			args = append(args, "--stripe-context", shellQuoteArgument(value))
		case "stripe-version":
			args = append(args, "--stripe-version", shellQuoteArgument(value))
		case "idempotency-key":
			args = append(args, "--idempotency", shellQuoteArgument(value))
		default:
			return ""
		}
	}

	sourceParams, ok := request.Params.(map[string]any)
	if request.Params != nil && !ok {
		return ""
	}
	params := cloneMap(sourceParams)
	if params == nil {
		params = make(map[string]any)
	}
	hiddenParams, ok := request.HiddenParams.(map[string]any)
	if request.HiddenParams != nil && !ok {
		return ""
	}
	deepMerge(params, hiddenParams)
	if containsUnresolvedValue(params) {
		return ""
	}
	if len(params) == 0 {
		return strings.Join(args, " ")
	}
	if strings.HasPrefix(request.Path, "/v2/") {
		encoded, err := json.Marshal(params)
		if err != nil {
			return ""
		}
		return strings.Join(append(args, "-d", shellQuoteArgument(string(encoded))), " ")
	}

	data, ok := flattenStripeParams(params)
	if !ok {
		return ""
	}
	for _, value := range data {
		args = append(args, "-d", shellQuoteArgument(value))
	}
	return strings.Join(args, " ")
}

func flattenStripeParams(params map[string]any) ([]string, bool) {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var flattened []string
	for _, key := range keys {
		values, ok := flattenStripeParam(key, params[key])
		if !ok {
			return nil, false
		}
		flattened = append(flattened, values...)
	}
	return flattened, true
}

func flattenStripeParam(key string, value any) ([]string, bool) {
	switch value := value.(type) {
	case string:
		return []string{key + "=" + value}, true
	case bool, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return []string{key + "=" + fmt.Sprint(value)}, true
	case map[string]any:
		if len(value) == 0 {
			return nil, false
		}
		keys := make([]string, 0, len(value))
		for child := range value {
			keys = append(keys, child)
		}
		sort.Strings(keys)
		var flattened []string
		for _, child := range keys {
			values, ok := flattenStripeParam(key+"["+child+"]", value[child])
			if !ok {
				return nil, false
			}
			flattened = append(flattened, values...)
		}
		return flattened, true
	case []any:
		if len(value) == 0 {
			return nil, false
		}
		var flattened []string
		for index, child := range value {
			childKey := key + "[]"
			switch child.(type) {
			case map[string]any, []any:
				childKey = key + "[" + strconv.Itoa(index) + "]"
			}
			values, ok := flattenStripeParam(childKey, child)
			if !ok {
				return nil, false
			}
			flattened = append(flattened, values...)
		}
		return flattened, true
	default:
		return nil, false
	}
}

func containsUnresolvedValue(value any) bool {
	switch value := value.(type) {
	case string:
		return strings.Contains(value, "${")
	case map[string]any:
		for key, child := range value {
			if containsUnresolvedValue(key) || containsUnresolvedValue(child) {
				return true
			}
		}
	case map[string]string:
		for key, child := range value {
			if containsUnresolvedValue(key) || containsUnresolvedValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if containsUnresolvedValue(child) {
				return true
			}
		}
	}
	return false
}

func shellQuoteArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
