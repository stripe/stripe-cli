package cmd

// overview_sandbox.go looks up the live account that owns the active sandbox
// for `stripe overview --force-live`, and renders the sandbox header and
// footer that tell the reader how to see their live account's insights.

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

var lookupOverviewSandboxParent = sandboxParentAccount

type sandboxFooter int

const (
	noSandboxFooter sandboxFooter = iota
	// sandboxFooterForceLive: the live account is authorized (or couldn't be
	// checked).
	sandboxFooterForceLive
	// sandboxFooterLogin: the live account isn't authorized yet.
	sandboxFooterLogin
)

// sandboxParentAccount returns the live account that owns the sandbox in
// creds, through context_statuses, which only returns the parent when it's
// within the CLI's permission grant.
func sandboxParentAccount(ctx context.Context, apiBaseURL string, profile *config.Profile, creds stripe.Credentials) (string, bool) {
	statuses, err := requests.GetContextStatuses(ctx, apiBaseURL, profile, creds, []requests.ContextStatusQuery{
		{AccountID: creds.OAKContext, Livemode: false},
	})
	if err != nil {
		log.Debugf("Could not look up the live account for sandbox %s: %v", creds.OAKContext, err)
		return "", false
	}
	for _, s := range statuses {
		if s.AccountID == creds.OAKContext && !s.Livemode && s.Valid {
			return s.ParentAccountID, true
		}
	}
	return "", true
}

func renderForcedLiveHeader(w io.Writer, target overviewTarget) {
	if !target.forcedLive {
		return
	}
	fmt.Fprintf(w, "Showing insights for your live account (%s).\n\n", target.creds.OAKContext)
}

const (
	sandboxNoticeMessage = "You're currently in a sandbox, so these insights are only applicable to your sandbox."
	loginCommand         = "stripe login"
	forceLiveCommand     = "stripe overview --force-live"
)

func renderSandboxFooter(w io.Writer, footer sandboxFooter) {
	forceLive := ansi.Bold("`" + forceLiveCommand + "`")
	switch footer {
	case sandboxFooterForceLive:
		fmt.Fprintln(w)
		fmt.Fprintln(w, sandboxNoticeMessage)
		fmt.Fprintf(w, "Run %s to see insights for your live account.\n", forceLive)
	case sandboxFooterLogin:
		fmt.Fprintln(w)
		fmt.Fprintln(w, sandboxNoticeMessage)
		fmt.Fprintf(w, "Run %s to authorize your live account.\n", ansi.Bold("`"+loginCommand+"`"))
		fmt.Fprintf(w, "Then run %s to see recommended actions for your live account.\n", forceLive)
	case noSandboxFooter:
	}
}

// sandboxNotice is the JSON form of the sandbox footer: the same message and
// the commands to run, in order. It's nil outside a sandbox.
func sandboxNotice(footer sandboxFooter) *overviewJSONNotice {
	forceLive := overviewJSONStep{Command: forceLiveCommand, Purpose: "Show recommended actions for the live account that owns this sandbox."}
	switch footer {
	case sandboxFooterForceLive:
		return &overviewJSONNotice{Message: sandboxNoticeMessage, NextSteps: []overviewJSONStep{forceLive}}
	case sandboxFooterLogin:
		login := overviewJSONStep{Command: loginCommand, Purpose: "Authorize the live account that owns this sandbox."}
		return &overviewJSONNotice{Message: sandboxNoticeMessage, NextSteps: []overviewJSONStep{login, forceLive}}
	case noSandboxFooter:
	}
	return nil
}
