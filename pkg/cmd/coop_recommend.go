package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopRecommendCmd struct {
	cmd   *cobra.Command
	query string
}

func newCoopRecommendCmd() *coopRecommendCmd {
	rc := &coopRecommendCmd{}
	rc.cmd = &cobra.Command{
		Use:   "recommend",
		Short: "Find blueprints matching your integration needs",
		Long: `Search available blueprints by keyword. Useful for discovering which
blueprint to use for a given integration scenario.`,
		Example: `  stripe coop recommend --query="accept payments"
  stripe coop recommend --query="subscriptions"
  stripe coop recommend --query="save card future"`,
		RunE: rc.runRecommendCmd,
	}

	rc.cmd.Flags().StringVar(&rc.query, "query", "", "Search query to match against blueprints")

	return rc
}

func (rc *coopRecommendCmd) runRecommendCmd(cmd *cobra.Command, args []string) error {
	blueprints, err := coop.ListBlueprintsWithMetadata()
	if err != nil {
		return fmt.Errorf("loading blueprints: %w", err)
	}

	type bpEntry struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Products    []string `json:"products,omitempty"`
		StepCount   int      `json:"step_count"`
		Command     string   `json:"command"`
		Score       int      `json:"score,omitempty"`
	}

	var catalog []bpEntry
	for _, bp := range blueprints {
		steps := 0
		for _, ch := range bp.Chapters {
			steps += len(ch.Nodes)
		}
		entry := bpEntry{
			ID:          bp.ID,
			Title:       bp.Title,
			Description: bp.Description,
			Products:    bp.Products,
			StepCount:   steps,
			Command:     fmt.Sprintf("stripe coop run %s", bp.ID),
		}
		if rc.query != "" {
			entry.Score = blueprintMatchScore(bp, rc.query)
			if entry.Score == 0 {
				continue
			}
		}
		catalog = append(catalog, entry)
	}

	if rc.query != "" {
		sort.SliceStable(catalog, func(i, j int) bool {
			return catalog[i].Score > catalog[j].Score
		})
	}

	response := map[string]interface{}{
		"blueprints": catalog,
		"agent_instructions": `Pick the blueprint that best matches what the developer described.
Consider: what they're building, whether it's one-time or recurring, if it involves platforms/marketplaces.
If multiple could fit, ask the developer to clarify between the top 2-3 options.
Once decided, run the "command" field for that blueprint.`,
	}

	if rc.query != "" {
		response["query"] = rc.query
	}

	return outputJSON(response)
}

func blueprintMatchScore(bp coop.Blueprint, query string) int {
	query = strings.ToLower(query)
	terms := strings.FieldsFunc(query, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == ',' || r == '.' || r == '/'
	})
	if len(terms) == 0 {
		return 0
	}

	id := strings.ToLower(bp.ID)
	title := strings.ToLower(bp.Title)
	description := strings.ToLower(bp.Description)
	products := strings.ToLower(strings.Join(bp.Products, " "))

	score := 0
	for _, term := range terms {
		switch {
		case term == "":
			continue
		case strings.Contains(id, term):
			score += 5
		case strings.Contains(title, term):
			score += 4
		case strings.Contains(products, term):
			score += 3
		case strings.Contains(description, term):
			score += 2
		}
	}
	if strings.Contains(title, query) || strings.Contains(description, query) || strings.Contains(id, strings.ReplaceAll(query, " ", "-")) {
		score += 5
	}
	return score
}
