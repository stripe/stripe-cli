package samples

import (
	"testing"
)

func TestSampleData_GetCategory(t *testing.T) {
	tests := []struct {
		name     string
		sample   SampleData
		expected string
	}{
		{
			name: "with category",
			sample: SampleData{
				Name:     "test",
				Category: "Payments",
			},
			expected: "Payments",
		},
		{
			name: "without category",
			sample: SampleData{
				Name: "test",
			},
			expected: "General",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sample.GetCategory(); got != tt.expected {
				t.Errorf("SampleData.GetCategory() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSampleData_GetDifficulty(t *testing.T) {
	tests := []struct {
		name     string
		sample   SampleData
		expected string
	}{
		{
			name: "with difficulty",
			sample: SampleData{
				Name:       "test",
				Difficulty: "Beginner",
			},
			expected: "Beginner",
		},
		{
			name: "without difficulty",
			sample: SampleData{
				Name: "test",
			},
			expected: "Intermediate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sample.GetDifficulty(); got != tt.expected {
				t.Errorf("SampleData.GetDifficulty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSampleData_GetLanguage(t *testing.T) {
	tests := []struct {
		name     string
		sample   SampleData
		expected string
	}{
		{
			name: "with language",
			sample: SampleData{
				Name:     "test",
				Language: "JavaScript",
			},
			expected: "JavaScript",
		},
		{
			name: "without language",
			sample: SampleData{
				Name: "test",
			},
			expected: "Multiple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sample.GetLanguage(); got != tt.expected {
				t.Errorf("SampleData.GetLanguage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGroupByCategory(t *testing.T) {
	samples := map[string]*SampleData{
		"sample1": {
			Name:     "sample1",
			Category: "Payments",
		},
		"sample2": {
			Name:     "sample2",
			Category: "Payments",
		},
		"sample3": {
			Name:     "sample3",
			Category: "Subscriptions",
		},
	}

	grouped := GroupByCategory(samples)

	if len(grouped) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(grouped))
	}

	if len(grouped["Payments"]) != 2 {
		t.Errorf("Expected 2 samples in Payments category, got %d", len(grouped["Payments"]))
	}

	if len(grouped["Subscriptions"]) != 1 {
		t.Errorf("Expected 1 sample in Subscriptions category, got %d", len(grouped["Subscriptions"]))
	}
}

func TestGetCategories(t *testing.T) {
	samples := map[string]*SampleData{
		"sample1": {
			Name:     "sample1",
			Category: "Payments",
		},
		"sample2": {
			Name:     "sample2",
			Category: "Subscriptions",
		},
		"sample3": {
			Name:     "sample3",
			Category: "Payments",
		},
	}

	categories := GetCategories(samples)

	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}

	// Check if categories are sorted
	if categories[0] != "Payments" || categories[1] != "Subscriptions" {
		t.Errorf("Categories not sorted correctly: %v", categories)
	}
}

func TestFilterByCategory(t *testing.T) {
	samples := map[string]*SampleData{
		"sample1": {
			Name:     "sample1",
			Category: "Payments",
		},
		"sample2": {
			Name:     "sample2",
			Category: "Subscriptions",
		},
		"sample3": {
			Name:     "sample3",
			Category: "Payments",
		},
	}

	filtered := FilterByCategory(samples, "Payments")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 samples in Payments category, got %d", len(filtered))
	}

	for _, sample := range filtered {
		if sample.Category != "Payments" {
			t.Errorf("Expected category Payments, got %s", sample.Category)
		}
	}
}

func TestFilterByTag(t *testing.T) {
	samples := map[string]*SampleData{
		"sample1": {
			Name: "sample1",
			Tags: []string{"checkout", "payments"},
		},
		"sample2": {
			Name: "sample2",
			Tags: []string{"subscriptions", "billing"},
		},
		"sample3": {
			Name: "sample3",
			Tags: []string{"checkout", "security"},
		},
	}

	filtered := FilterByTag(samples, "checkout")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 samples with 'checkout' tag, got %d", len(filtered))
	}

	for _, sample := range filtered {
		found := false
		for _, tag := range sample.Tags {
			if tag == "checkout" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Sample %s should have 'checkout' tag", sample.Name)
		}
	}
}
