package reporting

import (
	"strings"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

const otherCommandBucket = "other"

var builtInCommandBuckets = map[string]struct{}{
	"agent":       {},
	"community":   {},
	"completion":  {},
	"config":      {},
	"daemon":      {},
	"data":        {},
	"delete":      {},
	"docs":        {},
	"feedback":    {},
	"fixtures":    {},
	"get":         {},
	"listen":      {},
	"login":       {},
	"logout":      {},
	"logs":        {},
	"open":        {},
	"plugin":      {},
	"post":        {},
	"postinstall": {},
	"provision":   {},
	"reauth":      {},
	"reporting":   {},
	"resources":   {},
	"samples":     {},
	"sandbox":     {},
	"serve":       {},
	"status":      {},
	"switch":      {},
	"trigger":     {},
	"version":     {},
	"whoami":      {},
}

func commandBucket(metadata *stripe.CLIAnalyticsEventMetadata) string {
	if metadata == nil {
		return otherCommandBucket
	}
	if metadata.GeneratedResource {
		return "resources"
	}

	path := strings.Fields(metadata.CommandPath)
	if len(path) > 0 && path[0] == "stripe" {
		path = path[1:]
	}
	if len(path) == 0 {
		return otherCommandBucket
	}
	if len(path) > 1 && path[0] == "logs" && path[1] == "tail" {
		return "logs_tail"
	}
	if _, ok := builtInCommandBuckets[path[0]]; ok {
		return path[0]
	}
	return otherCommandBucket
}
