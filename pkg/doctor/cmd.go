package doctor

// cmd.go — the command surface, shaped like the real product would be:
// a generic verb (doctor) with the migration TOPIC as an argument,
// mirroring the architecture (generic engine, rule packs as data).
//
//	stripe doctor [topic] [dir]   diagnose; degrades to scan-only without creds
//	stripe doctor guide                  agent playbook
//
// scan/drill are no longer commands: scan is doctor's credential-less
// degradation and the webhook drill is `doctor --live`. doctor only
// diagnoses — remediation is manual, guided by each finding's verdict.
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
	"github.com/stripe/stripe-cli/pkg/doctor/drill"
	"github.com/stripe/stripe-cli/pkg/doctor/judge"
	"github.com/stripe/stripe-cli/pkg/doctor/scan"
	"github.com/stripe/stripe-cli/pkg/doctor/signals"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

var (
	flagJSON          bool
	flagStripeAccount string
)

// topicList renders the registry for help and error text.
func topicList() string {
	var names []string
	for n := range catalog.Packs {
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
// naming a known pack is the topic; anything else is the directory.
func topicAndDir(args []string) (string, string, error) {
	topic, dir := "dpm", "."
	var nonTopic []string
	for _, a := range args {
		if _, ok := catalog.Packs[a]; ok {
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
// `doctor guide` resolve unambiguously. cfg supplies profile credentials,
// resolved downstream in whichever mode (test or live) the active context
// is actually logged into.
func NewDoctorCmd(cfg *config.Config) *cobra.Command {
	root := newDoctorCmd(cfg)
	commonFlags(root)

	root.AddCommand(newGuideCmd(), newDumpCmd())
	return root
}

// ---------- doctor ----------

func newDoctorCmd(cfg *config.Config) *cobra.Command {
	var live, offline bool
	c := &cobra.Command{
		Use:   "doctor [topic] [dir]",
		Short: "Diagnose a migration: code findings checked against your live account (read-only)",
		Long: `Fetches the migration rule live, scans the directory for its findings, and
judges each against live account facts (API versions in recent traffic,
Dashboard configuration). --offline skips the account-facts judgment only
— the rule fetch itself always requires network access. --live additionally
proves runtime behavior (webhook round-trip via stripe listen/trigger).`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic, dir, err := topicAndDir(args)
			if err != nil {
				fail(err)
			}
			// dpm has no local rule copy — it only exists via a live fetch
			// from the doctor_rules API (--offline skips account facts, not
			// this: there's nothing to scan with otherwise). This is
			// deliberately special-cased rather than a generic per-pack
			// hook until a second topic needs its own live source.
			var rule catalog.Rule
			err = withSpinnerUnlessJSON("Fetching migration rules (read-only)", func() error {
				var derr error
				rule, derr = judge.FetchDPMRule(cfg, flagStripeAccount)
				return derr
			})
			if err != nil {
				fail(err)
			}

			findings, scanned, parsed, err := scan.Scan(dir, rule)
			if err != nil {
				fail(topicHint(err, dir))
			}
			scan.SortFindings(findings)
			stats := ScanStats{FilesScanned: scanned, FilesParsed: parsed, Skipped: scanned - parsed}

			rep := &DoctorReport{Command: "doctor", Topic: topic, Stats: stats}
			applyJudgment := func(jr *judge.Report) {
				rep.Account, rep.Findings, rep.Summary = jr.Account, jr.Findings, jr.Summary
			}
			if offline {
				rep.Degraded = "offline requested (--offline)"
				applyJudgment(judge.ScanOnly(rule, findings, rep.Degraded))
			} else {
				var jr *judge.Report
				err := withSpinnerUnlessJSON("Fetching account facts (read-only)", func() error {
					var derr error
					jr, derr = judge.Judge(findings, cfg, flagStripeAccount, rule)
					return derr
				})
				if err != nil {
					// Graceful degradation: no creds -> scan-only, clearly labeled.
					rep.Degraded = err.Error()
					applyJudgment(judge.ScanOnly(rule, findings, rep.Degraded))
				} else {
					applyJudgment(jr)
				}
			}

			// Code signals need no credentials — attach in every mode.
			rep.WebhookHandlers = signals.ScanSignals(dir, catalog.Packs[topic].Signals)

			failedLive := false
			if live {
				var drillResult *drill.Report
				err := withSpinnerUnlessJSON("Live check: webhook round-trip (listen + trigger)", func() error {
					var derr error
					drillResult, derr = drill.Run()
					return derr
				})
				if err != nil {
					fail(err)
				}
				rep.LiveDrill = drillResult
				failedLive = !drillResult.Verified
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
	c.Flags().BoolVar(&offline, "offline", false, "skip account facts; scan-only (the migration rule itself is still fetched live)")
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

// topicHint decorates a scan error when the "directory" looks like a typo'd
// topic name (e.g. `doctor dmp`).
func topicHint(err error, dir string) error {
	for name := range catalog.Packs {
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
