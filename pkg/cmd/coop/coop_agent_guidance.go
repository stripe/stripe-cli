package coopcmd

const (
	stripeBestPracticesSkillCommit     = "c29cd23cfd27830bf10961d58646a9fd127fa6df"
	stripeBestPracticesSkillSource     = "https://github.com/stripe/ai/tree/" + stripeBestPracticesSkillCommit + "/skills/stripe-best-practices"
	stripeBestPracticesSkillTreeURL    = "https://api.github.com/repos/stripe/ai/git/trees/" + stripeBestPracticesSkillCommit + "?recursive=1"
	stripeBestPracticesSkillRawBaseURL = "https://raw.githubusercontent.com/stripe/ai/" + stripeBestPracticesSkillCommit + "/skills/stripe-best-practices"
	stripeAgentGuidanceStart           = "=== STRIPE IMPLEMENTATION GUIDANCE ==="
	stripeAgentGuidanceEnd             = "=== END STRIPE IMPLEMENTATION GUIDANCE ==="
)

func coopAgentCoordinationInstructions() string {
	return `COORDINATION AND DELEGATION:
We will handle coordination and orchestration with you. Prefer delegating well-bounded, independent tasks to cheaper subagents wherever possible, using your best judgment. Keep work on the main agent when delegation would add more overhead than value, require tightly shared context, or block critical-path progress. The main agent remains responsible for integration decisions and Co-op lifecycle commands. Read and honor repository guidance files such as AGENTS.md and CLAUDE.md when present.`
}

func stripeAgentGuidanceInstructions() string {
	return stripeAgentGuidanceStart + `
Co-op is responsible for selecting the integration and API family through its recommender and blueprint. Once selected, node descriptions and review requirements are the task contract. Do not use documentation or the repo-scoped Stripe skill to choose or switch integrations or API families.

After Co-op selects a blueprint, it attempts to make the pinned stripe-best-practices skill available in both native local-repository locations: .agents/skills/stripe-best-practices for Codex and .claude/skills/stripe-best-practices for Claude Code. Codex detects newly installed repo skills automatically, and Co-op prepares Claude Code's empty repo skill root before discovery so Claude can detect the later addition. If a local skill already exists at either destination, Co-op leaves that destination untouched. Do not invoke this skill before Co-op selects the blueprint; afterward it is optional, supplemental implementation guidance within Co-op's selection. It does not override the Co-op task contract, and neither skill nor documentation lookup is mandatory.

If Stripe behavior, parameters, fields, events, or implementation details are ambiguous or need clarification, proactively consult current official Stripe documentation through the CLI instead of guessing.

Search official guides with:

  stripe docs search "<specific Stripe question>" --non-interactive --no-pager

Open a relevant guide when doing so would resolve the ambiguity with:

  stripe docs <result-path> --non-interactive --no-pager

Inspect exact API resources, operations, parameters, fields, and event schemas with:

  stripe docs api <resource-or-event> --non-interactive --no-pager
  stripe docs api <HTTP-method> <endpoint> --non-interactive --no-pager

Documentation lookup is optional, not a mandatory preflight or ceremony. You do not need to search or open a guide when the Co-op task contract and Stripe behavior are already clear.

STRONGLY PREFER CURRENT OFFICIAL STRIPE CLI DOCUMENTATION OVER MODEL MEMORY. Treat current official documentation returned by the CLI as authoritative for the point in question. Never rely on remembered Stripe behavior when current CLI documentation is available; if they conflict, follow the CLI documentation.
` + stripeAgentGuidanceEnd
}
