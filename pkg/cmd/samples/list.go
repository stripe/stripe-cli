package samples

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// ListCmd prints a list of all the available sample projects that users can
// generate
type ListCmd struct {
	Cmd *cobra.Command
	
	// New filter flags
	category string
	difficulty string
	language string
	tag string
	compact bool
}

// NewListCmd creates and returns a list command for samples
func NewListCmd() *ListCmd {
	listCmd := &ListCmd{}
	listCmd.Cmd = &cobra.Command{
		Use:   "list",
		Args:  validators.NoArgs,
		Short: "List Stripe Samples supported by the CLI",
		Long: `A list of available Stripe Sample integrations that can be setup and bootstrap by
the CLI.

The samples are organized by category and include metadata such as difficulty level,
primary language, and tags for easier discovery.`,
		RunE: listCmd.runListCmd,
	}

	// Add filter flags
	listCmd.Cmd.Flags().StringVar(&listCmd.category, "category", "", "Filter samples by category")
	listCmd.Cmd.Flags().StringVar(&listCmd.difficulty, "difficulty", "", "Filter samples by difficulty level")
	listCmd.Cmd.Flags().StringVar(&listCmd.language, "language", "", "Filter samples by primary language")
	listCmd.Cmd.Flags().StringVar(&listCmd.tag, "tag", "", "Filter samples by tag")
	listCmd.Cmd.Flags().BoolVar(&listCmd.compact, "compact", false, "Display samples in compact format")
	
	// Add usage examples
	listCmd.Cmd.Example = `  stripe samples list
  stripe samples list --category "Payments"
  stripe samples list --difficulty "Beginner"
  stripe samples list --language "JavaScript"
  stripe samples list --tag "webhook"
  stripe samples list --compact`

	return listCmd
}

func (lc *ListCmd) runListCmd(cmd *cobra.Command, args []string) error {
	fmt.Println("A list of available Stripe Samples:")
	fmt.Println()

	spinner := ansi.StartNewSpinner("Loading...", os.Stdout)

	list, err := samples.GetSamples("list")
	if err != nil {
		ansi.StopSpinner(spinner, "Error: please check your internet connection and try again!", os.Stdout)
		return err
	}
	ansi.StopSpinner(spinner, "", os.Stdout)

	// Apply filters if specified
	filteredList := lc.applyFilters(list)
	
	if len(filteredList) == 0 {
		fmt.Println("No samples found matching the specified criteria.")
		return nil
	}

	// Group samples by category for better organization
	groupedSamples := samples.GroupByCategory(filteredList)
	categories := samples.GetCategories(filteredList)

	// Display samples grouped by category
	for _, category := range categories {
		fmt.Printf("%s\n", ansi.Bold(fmt.Sprintf("📁 %s", category)))
		fmt.Println(strings.Repeat("-", len(category)+4))
		
		samplesInCategory := groupedSamples[category]
		// Sort samples within each category by display name
		sort.Slice(samplesInCategory, func(i, j int) bool {
			nameI := samplesInCategory[i].DisplayName
			if nameI == "" {
				nameI = samplesInCategory[i].Name
			}
			nameJ := samplesInCategory[j].DisplayName
			if nameJ == "" {
				nameJ = samplesInCategory[j].Name
			}
			return nameI < nameJ
		})
		
		for _, sample := range samplesInCategory {
			if lc.compact {
				lc.displaySampleCompact(sample)
			} else {
				lc.displaySample(sample)
			}
		}
		fmt.Println()
	}

	// Display summary
	fmt.Printf("Total: %d samples across %d categories\n", len(filteredList), len(categories))
	fmt.Println("\n💡 Tip: Use 'stripe samples create <name>' to set up a sample project")

	return nil
}

// applyFilters applies the specified filters to the sample list
func (lc *ListCmd) applyFilters(list map[string]*samples.SampleData) map[string]*samples.SampleData {
	filtered := list
	
	if lc.category != "" {
		filtered = samples.FilterByCategory(filtered, lc.category)
	}
	
	if lc.difficulty != "" {
		temp := make(map[string]*samples.SampleData)
		for key, sample := range filtered {
			if sample.GetDifficulty() == lc.difficulty {
				temp[key] = sample
			}
		}
		filtered = temp
	}
	
	if lc.language != "" {
		temp := make(map[string]*samples.SampleData)
		for key, sample := range filtered {
			if sample.GetLanguage() == lc.language {
				temp[key] = sample
			}
		}
		filtered = temp
	}
	
	if lc.tag != "" {
		filtered = samples.FilterByTag(filtered, lc.tag)
	}
	
	return filtered
}

// displaySample displays a single sample with enhanced formatting
func (lc *ListCmd) displaySample(sample *samples.SampleData) {
	// Display name with fallback to original name
	displayName := sample.DisplayName
	if displayName == "" {
		displayName = sample.Name
	}
	
	fmt.Printf("  %s\n", ansi.Bold(displayName))
	
	// Display description
	if sample.Description != "" {
		fmt.Printf("    %s\n", sample.Description)
	}
	
	// Display metadata if available
	metadata := []string{}
	if sample.GetDifficulty() != "Intermediate" {
		metadata = append(metadata, fmt.Sprintf("Difficulty: %s", sample.GetDifficulty()))
	}
	if sample.GetLanguage() != "Multiple" {
		metadata = append(metadata, fmt.Sprintf("Language: %s", sample.GetLanguage()))
	}
	if len(sample.Tags) > 0 {
		metadata = append(metadata, fmt.Sprintf("Tags: %s", strings.Join(sample.Tags, ", ")))
	}
	
	if len(metadata) > 0 {
		fmt.Printf("    %s\n", strings.Join(metadata, " | "))
	}
	
	// Display repository URL
	fmt.Printf("    Repository: %s\n", sample.URL)
	fmt.Println()
}

// displaySampleCompact displays a single sample in compact format
func (lc *ListCmd) displaySampleCompact(sample *samples.SampleData) {
	displayName := sample.DisplayName
	if displayName == "" {
		displayName = sample.Name
	}
	
	metadata := []string{}
	if sample.GetDifficulty() != "Intermediate" {
		metadata = append(metadata, sample.GetDifficulty())
	}
	if sample.GetLanguage() != "Multiple" {
		metadata = append(metadata, sample.GetLanguage())
	}
	
	metadataStr := ""
	if len(metadata) > 0 {
		metadataStr = fmt.Sprintf(" [%s]", strings.Join(metadata, ", "))
	}
	
	fmt.Printf("  %s%s\n", ansi.Bold(displayName), metadataStr)
}
