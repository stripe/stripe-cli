package coop

import (
	"fmt"
	"strings"
)

// evaluateInclusion handles the equality expression shape currently returned
// by Workbench. Unknown operators fail closed instead of silently adding the
// wrong step or node to a session.
func evaluateInclusion(condition any, settings map[string]string) (bool, error) {
	switch value := condition.(type) {
	case nil:
		return true, nil
	case bool:
		return value, nil
	case map[string]any:
		if len(value) != 1 {
			return false, fmt.Errorf("expected one inclusion operator, got %d", len(value))
		}
		rawOperands, ok := value["=="]
		if !ok {
			for operator := range value {
				return false, fmt.Errorf("unsupported inclusion operator %q", operator)
			}
		}
		operands, ok := rawOperands.([]any)
		if !ok || len(operands) != 2 {
			return false, fmt.Errorf("== requires two operands")
		}
		left, err := resolveInclusionOperand(operands[0], settings)
		if err != nil {
			return false, err
		}
		right, err := resolveInclusionOperand(operands[1], settings)
		if err != nil {
			return false, err
		}
		return inclusionScalar(left) == inclusionScalar(right), nil
	}
	return false, fmt.Errorf("unsupported inclusion value %T", condition)
}

func resolveInclusionOperand(value any, settings map[string]string) (any, error) {
	text, ok := value.(string)
	if !ok {
		return value, nil
	}
	for _, prefix := range []string{"${params:", "${settings:"} {
		if strings.HasPrefix(text, prefix) && strings.HasSuffix(text, "}") {
			key := strings.TrimSuffix(strings.TrimPrefix(text, prefix), "}")
			resolved, exists := settings[key]
			if !exists {
				return nil, fmt.Errorf("inclusion references unknown setting %q", key)
			}
			return resolved, nil
		}
	}
	return text, nil
}

func inclusionScalar(value any) string {
	if value == nil {
		return "null"
	}
	return stringifyDefault(value)
}
