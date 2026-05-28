package agent

import "testing"

func TestDetectWith_AgentDetected(t *testing.T) {
	for _, key := range envVars {
		t.Run(key, func(t *testing.T) {
			getEnv := func(k string) string {
				if k == key {
					return "1"
				}
				return ""
			}
			if !DetectWith(getEnv) {
				t.Errorf("expected agent detected for %s", key)
			}
		})
	}
}

func TestDetectWith_NoAgent(t *testing.T) {
	getEnv := func(string) string { return "" }
	if DetectWith(getEnv) {
		t.Error("expected no agent detected")
	}
}
