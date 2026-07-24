package coopcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type coopRecommendCmd struct {
	cmd            *cobra.Command
	all            bool
	includeTesting bool
}

func newCoopRecommendCmd() *coopRecommendCmd {
	rc := &coopRecommendCmd{}
	rc.cmd = &cobra.Command{
		Use:   "recommend",
		Short: "List blueprints for an agent to recommend",
		Long: `List available blueprint summaries so an agent can choose the best match
for the developer's requested integration.`,
		Example: `  stripe coop recommend --all`,
		RunE:    rc.runRecommendCmd,
	}

	rc.cmd.Flags().BoolVar(&rc.all, "all", false, "Return all learning blueprint summaries for agent selection")
	rc.cmd.Flags().BoolVar(&rc.includeTesting, "include-testing", false, "Include testing blueprints in addition to learning blueprints")

	return rc
}

func (rc *coopRecommendCmd) runRecommendCmd(cmd *cobra.Command, args []string) error {
	if !rc.all {
		return fmt.Errorf("recommend requires --all to list blueprint summaries")
	}
	repository := coopBlueprintRepository()
	if repository == nil {
		return fmt.Errorf("loading blueprints: no blueprint repository configured")
	}
	blueprints, err := repository.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("loading blueprints: %w", err)
	}

	type bpEntry struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Type        string   `json:"type"`
		Products    []string `json:"products,omitempty"`
		// NodeCount is retained as a nullable compatibility field because the
		// Workbench list endpoint does not expose nodes.
		NodeCount *int   `json:"node_count"`
		StepCount int    `json:"step_count"`
		Command   string `json:"command"`
	}

	var catalog []bpEntry
	for _, bp := range blueprints {
		if !rc.includeTesting && bp.BlueprintType != "learning" {
			continue
		}
		entry := bpEntry{
			ID:          bp.Key,
			Title:       bp.Title.DefaultMessage,
			Description: bp.Description.DefaultMessage,
			Type:        bp.BlueprintType,
			Products:    bp.Metadata.Products,
			StepCount:   len(bp.StepRefs),
			Command:     fmt.Sprintf("stripe coop run %s", bp.Key),
		}
		catalog = append(catalog, entry)
	}

	response := map[string]interface{}{
		"blueprints": catalog,
		"agent_instructions": `Review every blueprint summary and pick the best match for the developer's request.
Consider: what they're building, whether it's one-time or recurring, if it involves platforms/marketplaces.
If multiple could fit, ask the developer to clarify between the top 2-3 options.
Once decided, run the "command" field for that blueprint.`,
	}

	return outputJSON(response)
}
