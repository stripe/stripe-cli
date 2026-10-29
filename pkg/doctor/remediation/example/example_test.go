// Package example holds runnable usage samples for pkg/doctor/remediation
// (go test -run Example -v), kept as real Example functions so they fail
// to compile or diverge in output rather than silently going stale.
package example

import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

// exampleImplementation is the minimum viable remediation.Implementation
// — just a Status and Message, no live evidence or custom rendering.
// Register it from your own package's init():
//
//	func init() { remediation.Register("my_topic", exampleImplementation{}) }
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

// Example_actionPlan demonstrates Title/Findings, and converting a
// Finding to JSON via FindingsJSON.
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
