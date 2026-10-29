package doctor

// stripe doctor [topic] [dir]   diagnose (requires live-mode creds to fetch the rule)
// stripe doctor guide           agent playbook
//
// Exit codes: 0 once a report completes, regardless of what it found;
// non-zero only when doctor fails to produce a report at all.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/judge"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/scan"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/webhookcheck"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"

	// Registers the "dpm" remediation via init().
	_ "github.com/stripe/stripe-cli/pkg/doctor/remediation/dpm"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

var (
	flagJSON          bool
	flagStripeAccount string
)

// topicList renders the registry for help and error text.
func topicList() string {
	var names []string
	for n := range catalog.Topics {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func exitWith(code int) { os.Exit(code) }

func emitJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func fail(err error) {
	if flagJSON {
		emitJSON(map[string]any{"error": err.Error()})
	} else {
		fmt.Fprintln(os.Stderr, failLine(err.Error()))
	}
	exitWith(2)
}

// defaultTopic returns the sole registered topic, or "" if there's more
// than one and the caller must name one explicitly.
func defaultTopic() string {
	if len(catalog.Topics) != 1 {
		return ""
	}
	for name := range catalog.Topics {
		return name
	}
	return ""
}

func topicAndDir(args []string) (string, string, error) {
	topic, dir := defaultTopic(), "."
	var nonTopic []string
	for _, a := range args {
		if catalog.Topics[a] {
			topic = a
		} else {
			nonTopic = append(nonTopic, a)
		}
	}
	if len(nonTopic) > 1 || (len(args) == 2 && len(nonTopic) == 2) {
		return "", "", errorcategory.Errorf(errorcategory.UserInput, "unknown topic %q (available: %s)", nonTopic[0], topicList())
	}
	if len(nonTopic) == 1 {
		dir = nonTopic[0]
	}
	if topic == "" {
		return "", "", errorcategory.Errorf(errorcategory.UserInput, "a topic is required (available: %s)", topicList())
	}
	return topic, dir, nil
}

func commonFlags(c *cobra.Command) {
	c.Flags().BoolVar(&flagJSON, "json", false, "machine-readable output on stdout, logs on stderr")
	c.Flags().StringVar(&flagStripeAccount, "stripe-account", "", "Connect: connected account (acct_...) whose configuration governs direct charges")
}

func NewDoctorCmd(cfg *config.Config) *cobra.Command {
	root := newDoctorCmd(cfg)
	commonFlags(root)

	root.AddCommand(newGuideCmd(), newDumpCmd())
	return root
}

// ---------- doctor ----------

func newDoctorCmd(cfg *config.Config) *cobra.Command {
	var verifyWebhooks bool
	c := &cobra.Command{
		Use:   "doctor [topic] [dir]",
		Short: "Diagnose a migration: code findings checked against the migration rule (read-only)",
		Long: `Scans the directory and judges each finding against the migration rule —
the rule already carries whatever account-dependent guidance applies, so no
separate account lookup is made. --verify-webhooks additionally proves
runtime behavior (webhook round-trip via stripe listen/trigger).`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic, dir, err := topicAndDir(args)
			if err != nil {
				fail(err)
			}
			rule, err := judge.FetchRule(cfg, topic, flagStripeAccount)
			if err != nil {
				fail(err)
			}

			findings, scanned, parsed, err := scan.Scan(dir, rule)
			if err != nil {
				fail(topicHint(err, dir))
			}
			scan.SortFindings(findings)
			stats := ScanStats{FilesScanned: scanned, FilesParsed: parsed, Skipped: scanned - parsed}

			rep := &DoctorReport{
				Command:     "doctor",
				Topic:       topic,
				Message:     rule.Message,
				Action:      rule.ActionText,
				Docs:        rule.Docs,
				DoctorTopic: rule.RemediationTopic,
				Stats:       stats,
			}
			jr := judge.Evaluate(rule, findings)
			rep.Findings, rep.Summary = jr.Findings, jr.Summary

			if rule.RemediationTopic != "" {
				_ = withSpinnerUnlessJSON("Assessing remediation (read-only)", func() error {
					rep.RemediationResult = runRemediation(rule, dir, findings, cfg)
					return nil
				})
				if rep.RemediationResult != nil {
					applyFindingVerdicts(rep, rep.RemediationResult.FindingVerdicts)
				}
			}

			if verifyWebhooks {
				events, triggerEvent, ok := webhookCheckEvents(rule.RemediationTopic)
				if !ok {
					fail(errorcategory.Errorf(errorcategory.UserInput, "no webhook check defined for topic %q", rule.RemediationTopic))
				}
				var checkResult *webhookcheck.Report
				err := withSpinnerUnlessJSON("Verifying webhooks: round-trip (listen + trigger)", func() error {
					var derr error
					checkResult, derr = webhookcheck.Run(events, triggerEvent)
					return derr
				})
				if err != nil {
					fail(err)
				}
				rep.WebhookCheck = checkResult
			}

			if flagJSON {
				if rep.RemediationResult != nil && rep.RemediationResult.Report != nil {
					emitJSON(rep.RemediationResult.Report)
				} else {
					emitJSON(rep)
				}
			} else {
				renderDoctor(rep)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&verifyWebhooks, "verify-webhooks", false, "also run behavioral checks (spawns stripe listen/trigger)")
	return c
}

func newDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "parse-dump [dir]",
		Short:  "Print S-expression trees (grammar debugging)",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			d := "testdata"
			if len(args) > 0 {
				d = args[0]
			}
			scan.DumpTrees(d)
		},
	}
}

// runRemediation looks up and runs the rule's declared RemediationTopic.
// A missing implementation or a Run error both come back as a
// StatusUnknown Result rather than being dropped.
func runRemediation(rule catalog.Rule, dir string, findings []catalog.Finding, cfg *config.Config) *remediation.Result {
	topic := rule.RemediationTopic
	impl, ok := remediation.Lookup(topic)
	if !ok {
		return &remediation.Result{
			Topic:   topic,
			Status:  remediation.StatusUnknown,
			Message: fmt.Sprintf("no remediation implementation registered for topic %q", topic),
		}
	}
	res, err := impl.Run(remediation.Context{
		Dir:           dir,
		Rule:          rule,
		Findings:      findings,
		Config:        cfg,
		StripeAccount: flagStripeAccount,
	})
	if err != nil {
		res = remediation.Result{Topic: topic, Status: remediation.StatusUnknown, Message: err.Error()}
	}
	if res.Topic == "" {
		res.Topic = topic
	}
	return &res
}

// applyFindingVerdicts overrides each finding whose key appears in
// verdicts and retallies rep.Summary. Safe to call with a nil map.
func applyFindingVerdicts(rep *DoctorReport, verdicts map[string]string) {
	rep.Summary = map[string]int{}
	for i := range rep.Findings {
		f := &rep.Findings[i]
		if v, ok := verdicts[remediation.FindingKey(f.File, f.Line)]; ok {
			f.Verdict = v
			f.Class = v
			if idx := strings.Index(v, ":"); idx > 0 {
				f.Class = v[:idx]
			}
		}
		rep.Summary[f.Class]++
	}
}

// webhookCheckEvents returns topic's declared webhook round-trip events,
// if its remediation implements WebhookChecker.
func webhookCheckEvents(topic string) (events []string, triggerEvent string, ok bool) {
	impl, found := remediation.Lookup(topic)
	if !found {
		return nil, "", false
	}
	lc, ok := impl.(remediation.WebhookChecker)
	if !ok {
		return nil, "", false
	}
	events, triggerEvent = lc.WebhookCheckEvents()
	return events, triggerEvent, true
}

// topicHint decorates a scan error when dir looks like a typo'd topic name.
func topicHint(err error, dir string) error {
	for name := range catalog.Topics {
		if dir != name && editDistanceAtMost2(dir, name) {
			return fmt.Errorf("%w (did you mean topic %q?)", err, name)
		}
	}
	return err
}

func editDistanceAtMost2(a, b string) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	if len(b)-len(a) > 2 || len(b) > 12 {
		return false
	}
	prev := make([]int, len(a)+1)
	cur := make([]int, len(a)+1)
	for i := range prev {
		prev[i] = i
	}
	for j := 1; j <= len(b); j++ {
		cur[0] = j
		for i := 1; i <= len(a); i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[i] = min(min(cur[i-1]+1, prev[i]+1), prev[i-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(a)] <= 2
}

// ---------- helpers ----------

func withSpinnerUnlessJSON(msg string, fn func() error) error {
	if flagJSON {
		fmt.Fprintf(os.Stderr, "%s...\n", msg)
		return fn()
	}
	return withSpinner(msg, fn)
}
