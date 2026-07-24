package coop

import "fmt"

func deriveReviewPrompt(node WorkbenchBlueprintNode) string {
	switch node.NodeType {
	case NodeAPIRequest:
		return "Confirm the implementation calls the intended Stripe API and reuses any IDs needed by later steps."
	case NodeAsyncHandler:
		if command := deriveReviewCommand(node); command != "" {
			return fmt.Sprintf("Run `%s` and confirm the handler processes the signed event correctly.", command)
		}
		return "Cause the configured event to occur and confirm the handler processes it correctly."
	case NodeTestHelper:
		return "Run the test flow and confirm every configured request succeeds and produces the expected state."
	case NodeUIComponent:
		return "Open the app and confirm the user-facing flow works as described."
	default:
		return "Confirm the implementation produces the expected result."
	}
}

func deriveReviewCommand(node WorkbenchBlueprintNode) string {
	if node.NodeType != NodeAsyncHandler || node.AsyncHandlerDetails == nil || len(node.AsyncHandlerDetails.Events) != 1 {
		return ""
	}
	eventType := node.AsyncHandlerDetails.Events[0].EventType
	if !isEventType(eventType) {
		return ""
	}
	return "stripe trigger " + eventType
}

func isEventType(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}
