package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/useragent"
)

const (
	unknownCmdBatchFile = "unknown_commands.json"
	unknownCmdBatchSize = 10
	unknownCmdEventName = "Unknown Command Attempted"

	// unknownCmdMaxTokens bounds how many leading argv tokens are recorded,
	// and unknownCmdMaxTokenLen bounds each token's length, so a mistyped
	// command can't carry flag values or other positional user data (IDs,
	// emails, secrets) into telemetry.
	unknownCmdMaxTokens   = 3
	unknownCmdMaxTokenLen = 40
)

// sanitizeUnknownCommand returns the leading run of non-flag argv tokens,
// capped in count and length, for use as the "command" telemetry field.
// It stops at the first flag so flag values are never recorded, since
// anything past the attempted command name is not a command name.
func sanitizeUnknownCommand(args []string) string {
	tokens := make([]string, 0, unknownCmdMaxTokens)
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		if len(arg) > unknownCmdMaxTokenLen {
			arg = arg[:unknownCmdMaxTokenLen]
		}
		tokens = append(tokens, arg)
		if len(tokens) == unknownCmdMaxTokens {
			break
		}
	}
	return strings.Join(tokens, " ")
}

type unknownCommandEntry struct {
	Command   string `json:"command"`
	Timestamp string `json:"timestamp"`
	Agent     string `json:"agent"`
}

// recordUnknownCommand records an unknown command attempt in agent environments.
// It batches entries locally and sends telemetry every unknownCmdBatchSize occurrences.
func recordUnknownCommand(ctx context.Context, command string) {
	agent := useragent.DetectAIAgent(os.Getenv)
	if agent == "" {
		return
	}

	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient == nil {
		return
	}

	configFolder := Config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	batchPath := filepath.Join(configFolder, unknownCmdBatchFile)

	entries := loadUnknownCommandBatch(batchPath)

	entries = append(entries, unknownCommandEntry{
		Command:   command,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Agent:     agent,
	})

	if len(entries) >= unknownCmdBatchSize {
		sendUnknownCommandBatch(ctx, telemetryClient, entries)
		_ = os.Remove(batchPath)
	} else {
		saveUnknownCommandBatch(batchPath, entries)
	}
}

func loadUnknownCommandBatch(path string) []unknownCommandEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var entries []unknownCommandEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.Debugf("Failed to parse unknown command batch file: %v", err)
		return nil
	}
	return entries
}

func saveUnknownCommandBatch(path string, entries []unknownCommandEntry) {
	data, err := json.Marshal(entries)
	if err != nil {
		log.Debugf("Failed to marshal unknown command batch: %v", err)
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Debugf("Failed to create config directory for unknown command batch: %v", err)
		return
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		log.Debugf("Failed to write unknown command batch file: %v", err)
	}
}

func sendUnknownCommandBatch(ctx context.Context, client stripe.TelemetryClient, entries []unknownCommandEntry) {
	data, err := json.Marshal(entries)
	if err != nil {
		log.Debugf("Failed to marshal unknown command batch for telemetry: %v", err)
		return
	}

	client.SendEvent(ctx, unknownCmdEventName, string(data))
}
