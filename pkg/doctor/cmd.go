package doctor

// cmd.go — the command surface, shaped like the real product would be:
// a generic verb (doctor) with the migration TOPIC as an argument,
// mirroring the architecture (generic engine, rules as data).
//
//	stripe doctor [topic] [dir]   diagnose; degrades to scan-only without creds
//	stripe doctor guide                  agent playbook
//
// scan/webhookcheck are no longer commands: scan is doctor's
// credential-less degradation and the webhook round-trip check is
// `doctor --live`. doctor only diagnoses — remediation is manual, guided
// by each finding's verdict.
//
// Every command supports --json (pure JSON on stdout, logs on stderr) and the
// exit-code contract: 0 clean/verified, 1 findings/not-verified, 2 error.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/judge"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
	// Registers "dpm" with the remediation registry via init(). Other
	// teams' topics get the same one-line blank import once their
	// package exists — see pkg/doctor/remediation's package doc for the
	// full three-step guide.
	_ "github.com/stripe/stripe-cli/pkg/doctor/remediation/paymentmethods"
	"github.com/stripe/stripe-cli/pkg/doctor/scan"
	"github.com/stripe/stripe-cli/pkg/doctor/webhookcheck"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

var (
	flagJSON          bool
	flagStripeAccount string
)

// topicList renders the registry for help and error text.
func topicList() string {
	var names []string
	for n := range catalog.Rules {
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

// topicAndDir resolves the optional [topic] [dir] argument pair: an argument
// naming a known topic is the topic; anything else is the directory.
func topicAndDir(args []string) (string, string, error) {
	topic, dir := "dpm", "."
	var nonTopic []string
	for _, a := range args {
		if _, ok := catalog.Rules[a]; ok {
			topic = a
		} else {
			nonTopic = append(nonTopic, a)
		}
	}
	// With two args, the first must be a topic; two non-topics means the
	// first was a typo'd topic, not a directory.
	if len(nonTopic) > 1 || (len(args) == 2 && len(nonTopic) == 2) {
		return "", "", errorcategory.Errorf(errorcategory.UserInput, "unknown topic %q (available: %s)", nonTopic[0], topicList())
	}
	if len(nonTopic) == 1 {
		dir = nonTopic[0]
	}
	return topic, dir, nil
}

// commonFlags registers the flags every doctor-family command shares.
func commonFlags(c *cobra.Command) {
	c.Flags().BoolVar(&flagJSON, "json", false, "machine-readable output on stdout, logs on stderr")
	c.Flags().StringVar(&flagStripeAccount, "stripe-account", "", "Connect: connected account (acct_...) whose configuration governs direct charges")
}

// NewDoctorCmd builds the `stripe doctor` command tree: diagnosis is the
// root action, with guide and the hidden parse-dump nested beneath it —
// neither makes sense as a top-level verb in the CLI's namespace. Topic
// names and subcommand names never overlap, so `doctor dpm .` and
// `doctor guide` resolve unambiguously. cfg is currently unused by the
// doctor command itself (the rule comes from the embedded dpm.json, not a
// live fetch — see catalog/dpm.go) but is threaded through so this keeps
// working once judge.FetchDPMRule is swapped back in.
func NewDoctorCmd(cfg *config.Config) *cobra.Command {
	root := newDoctorCmd(cfg)
	commonFlags(root)

	root.AddCommand(newGuideCmd(), newDumpCmd())
	return root
}

// ---------- doctor ----------

func newDoctorCmd(cfg *config.Config) *cobra.Command {
	var live bool
	c := &cobra.Command{
		Use:   "doctor [topic] [dir]",
		Short: "Diagnose a migration: code findings checked against the migration rule (read-only)",
		Long: `Scans the directory and judges each finding against the migration rule —
the rule already carries whatever account-dependent guidance applies, so no
separate account lookup is made. --live additionally proves runtime
behavior (webhook round-trip via stripe listen/trigger).`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic, dir, err := topicAndDir(args)
			if err != nil {
				fail(err)
			}
			// Using the embedded dpm.json for now, not judge.FetchDPMRule's
			// live doctor_rules fetch, so local edits to it are picked up
			// immediately — see catalog/dpm.go. Swap this back once that's
			// no longer needed.
			rule := catalog.Rules[topic]

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
				if rep.RemediationResult != nil && rep.RemediationResult.FindingsVerdict != "" {
					applyFindingsVerdict(rep, rep.RemediationResult.FindingsVerdict)
				}
			}

			failedLive := false
			if live {
				var checkResult *webhookcheck.Report
				err := withSpinnerUnlessJSON("Live check: webhook round-trip (listen + trigger)", func() error {
					var derr error
					checkResult, derr = webhookcheck.Run()
					return derr
				})
				if err != nil {
					fail(err)
				}
				rep.WebhookCheck = checkResult
				failedLive = !checkResult.Verified
			}

			if flagJSON {
				emitJSON(rep)
			} else {
				renderDoctor(rep)
			}
			if len(rep.Findings) > 0 || failedLive {
				exitWith(1)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&live, "live", false, "also run behavioral checks (spawns stripe listen/trigger)")
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

// runRemediation runs the rule's declared RemediationTopic through the
// pkg/doctor/remediation registry. A topic with no registered
// Implementation (a team's Go code hasn't shipped yet) or one whose Run
// fails (e.g. a credentials error) is still surfaced as a StatusUnknown
// Result naming why, rather than being dropped silently — the raw
// RemediationTopic string is always visible too
// (DoctorReport.DoctorTopic), so this is purely "what doctor was able to
// compute from it."
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

// applyFindingsVerdict substitutes verdict for every finding's own
// intent-based verdict, uniformly, and retallies rep.Summary to match.
// Used when a remediation's Result.FindingsVerdict carries an
// account-wide fact (e.g. API-version safety) that applies the same way
// to every finding — something judge's per-finding intent classification
// has no way to see on its own.
func applyFindingsVerdict(rep *DoctorReport, verdict string) {
	class := verdict
	if i := strings.Index(verdict, ":"); i > 0 {
		class = verdict[:i]
	}
	rep.Summary = map[string]int{}
	for i := range rep.Findings {
		rep.Findings[i].Verdict = verdict
		rep.Findings[i].Class = class
		rep.Summary[class]++
	}
}

// topicHint decorates a scan error when the "directory" looks like a typo'd
// topic name (e.g. `doctor dmp`).
func topicHint(err error, dir string) error {
	for name := range catalog.Rules {
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
	// tiny DP is overkill at these sizes; do full Levenshtein
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
