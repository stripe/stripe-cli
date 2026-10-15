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
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

var (
	flagJSON          bool
	flagStripeAccount string
)

// A Pack pairs a rule with its declared signals (expected webhook events,
// legacy frontend tokens, package version floors). Adding a migration means
// adding an entry here — the verbs never change.
type Pack struct {
	Rule    Rule
	Signals *PackSignals
	Triage  []TriageBranch
}

var packs = map[string]Pack{
	"dpm":               {Rule: dpmRule, Signals: &dpmSignals},
	"tax-percent":       {Rule: taxPercentRule},
	"collection-method": {Rule: collectionMethodRule},
	"prorate":           {Rule: prorateRule},
	"source-types":      {Rule: sourceTypesRule},
	"ewcs":              {Rule: ewcsRule, Signals: &ewcsSignals},
	"flex":              {Rule: flexRule},
	"pe":                {Rule: peRule, Signals: &peSignals},
	"ct":                {Rule: ctRule, Signals: &ctSignals},
	"elements":          {Rule: elementsRule, Triage: elementsTriage},
}

// topicList renders the registry for help and error text.
func topicList() string {
	var names []string
	for n := range packs {
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
		if _, ok := packs[a]; ok {
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
// `doctor guide` resolve unambiguously. cfg supplies profile credentials
// (test-mode keys only — enforced downstream).
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
		Long: `Scans the directory for the topic's findings and judges each against live
account facts (API versions in recent traffic, Dashboard configuration).
Without credentials it degrades to scan-only. --live additionally proves
runtime behavior (webhook round-trip via stripe listen/trigger).`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic, dir, err := topicAndDir(args)
			if err != nil {
				fail(err)
			}
			rule := packs[topic].Rule

			findings, scanned, parsed, err := scan(dir, rule)
			if err != nil {
				fail(topicHint(err, dir))
			}
			sortFindings(findings)
			stats := ScanStats{FilesScanned: scanned, FilesParsed: parsed, Skipped: scanned - parsed}

			var rep *DoctorReport
			if offline {
				rep = scanOnlyReport(topic, rule, findings, stats, "offline requested (--offline)")
			} else {
				err := withSpinnerUnlessJSON("Fetching account facts (read-only)", func() error {
					var derr error
					rep, derr = buildDoctorReport(findings, cfg, flagStripeAccount, rule)
					return derr
				})
				if err != nil {
					// Graceful degradation: no creds -> scan-only, clearly labeled.
					rep = scanOnlyReport(topic, rule, findings, stats, err.Error())
				} else {
					rep.Topic = topic
					rep.Stats = stats
				}
			}

			// Code signals need no credentials — attach in every mode.
			rep.WebhookHandlers, rep.FrontendSignals, rep.ManifestChecks = scanSignals(dir, packs[topic].Signals)
			rep.Triage = scanTriage(dir, packs[topic].Triage)

			failedLive := false
			if live {
				var drill *DrillReport
				err := withSpinnerUnlessJSON("Live check: webhook round-trip (listen + trigger)", func() error {
					var derr error
					drill, derr = drillRun()
					return derr
				})
				if err != nil {
					fail(err)
				}
				rep.LiveDrill = drill
				failedLive = !drill.Verified
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
	c.Flags().BoolVar(&offline, "offline", false, "skip account facts; scan-only")
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
			dumpTrees(d)
		},
	}
}

// topicHint decorates a scan error when the "directory" looks like a typo'd
// topic name (e.g. `doctor dmp`).
func topicHint(err error, dir string) error {
	for name := range packs {
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
