// Package agentskills implements the agentskills.io standard for discovering and representing
// AI coding agent skills. docs.stripe.com publishes a skills index at
// /.well-known/skills/index.json following this standard.
package agentskills

// Index represents the skills index from a /.well-known/skills/index.json.
type Index struct {
	Skills []Skill `json:"skills"`
}

// Skill represents a single skill entry from the skills index.
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}
