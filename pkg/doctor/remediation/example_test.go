package remediation_test

// example_test.go is runnable documentation (go test -run Example -v)
// for pkg/doctor/remediation — see that package's doc comment for how
// these fit together. Keeping them as Example functions rather than a
// comment-only code sample means `go test` fails if they ever stop
// compiling or their output drifts from what's documented.

import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

// exampleImplementation is the minimum viable remediation.Implementation:
// Run reads whatever doctor already scanned (ctx.Findings) and returns a
// Result with just a Status and Message — no live account evidence, no
// custom text/JSON rendering. Register it from your own package's
// init(), e.g.:
//
//	func init() { remediation.Register("my_topic", exampleImplementation{}) }
//
// (not called here, to avoid colliding with other topics' registrations
// in the same test binary).
type exampleImplementation struct{}

func (exampleImplementation) Run(ctx remediation.Context) (remediation.Result, error) {
	return remediation.Result{
		Topic:   "example_topic",
		Status:  remediation.StatusReview,
		Message: fmt.Sprintf("found %d location(s) that need manual review", len(ctx.Findings)),
	}, nil
}

func Example() {
	impl := exampleImplementation{}
	res, err := impl.Run(remediation.Context{
		Findings: []catalog.Finding{
			{File: "app.rb", Line: 12},
			{File: "app.rb", Line: 30},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Status)
	fmt.Println(res.Message)
	// Output:
	// review
	// found 2 location(s) that need manual review
}

// Example_actionPlan demonstrates the opt-in pieces a richer
// Implementation can set: Title names the migration for doctor's text
// action-plan renderer, and Findings is the numbered, human-facing
// action plan — each Finding is also convertible to the generic --json
// finding shape via FindingsJSON, so a topic building its own
// Result.Report doesn't have to re-describe the same facts twice (see
// Finding's doc comment).
func Example_actionPlan() {
	findings := []remediation.Finding{
		{
			ID:      "hardcoded_list",
			Heading: "Review the hardcoded list",
			Body: []string{
				"This value is set directly in code.",
				"Replace it with the account's configured default.",
			},
			Items: []remediation.Item{
				{
					Heading: "app.rb:12",
					Body:    "Current value: ['a', 'b']",
					Fields:  map[string]any{"line": 12},
				},
			},
		},
	}

	result := remediation.Result{
		Topic:    "example_topic",
		Title:    "Example Migration",
		Status:   remediation.StatusReview,
		Findings: findings,
	}

	// doctor's text renderer (pkg/doctor's render.go) consumes
	// result.Title and result.Findings directly, printing
	// Title/Heading/Body/Items. A topic building its own --json contract
	// instead calls FindingsJSON:
	fmt.Println(result.Topic+":", result.Title)
	out, err := json.MarshalIndent(remediation.FindingsJSON(result.Findings, result.Status), "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	// Output:
	// example_topic: Example Migration
	// [
	//   {
	//     "id": "hardcoded_list",
	//     "status": "review_required",
	//     "evidence": {
	//       "items": [
	//         {
	//           "line": 12
	//         }
	//       ]
	//     },
	//     "implication": "This value is set directly in code.",
	//     "recommended_action": {
	//       "description": "Replace it with the account's configured default.",
	//       "safe_to_apply_without_review": false
	//     }
	//   }
	// ]
}
