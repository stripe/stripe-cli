package agent

import "testing"

func TestDetectWith_AgentDetected(t *testing.T) {
	for _, def := range agents {
		for _, key := range def.envKeys {
			t.Run(key, func(t *testing.T) {
				getEnv := func(k string) string {
					if k == key {
						return "1"
					}
					return ""
				}
				got := DetectWith(getEnv)
				if got != def.agent {
					t.Errorf("key %s: got %q, want %q", key, got, def.agent)
				}
			})
		}
	}
}

func TestDetectWith_NoAgent(t *testing.T) {
	getEnv := func(string) string { return "" }
	if got := DetectWith(getEnv); got != NotDetected {
		t.Errorf("got %q, want NotDetected", got)
	}
}
