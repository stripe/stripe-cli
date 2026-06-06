package cmd

import (
	"fmt"
	"os"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func buildSuggestions(session *coop.Session, env projectEnvironment) []suggestion {
	var suggestions []suggestion

	// Deploy suggestion
	switch {
	case env.HasStripeProjects:
		suggestions = append(suggestions, suggestion{
			ID:          "deploy",
			Title:       "Deploy with Stripe Projects",
			Description: "Your project is already configured — run stripe projects deploy",
			Available:   true,
			Reason:      "stripe.json found",
		})
	case !env.HasExistingDeploy:
		suggestions = append(suggestions, suggestion{
			ID:          "deploy",
			Title:       "Deploy with Stripe Projects",
			Description: "Set up hosting, CI/CD, and environment management",
			Available:   true,
			Reason:      "No deploy configuration detected",
		})
	default:
		target := env.deployTarget()
		suggestions = append(suggestions, suggestion{
			ID:          "deploy-update",
			Title:       "Deploy your changes",
			Description: fmt.Sprintf("Push your new integration code to %s", target),
			Available:   true,
			Reason:      fmt.Sprintf("Detected: %s", target),
		})
	}

	suggestions = append(suggestions, suggestion{
		ID:          "summarize",
		Title:       "Write a STRIPE.md summary",
		Description: "Generate a STRIPE.md with what was built, API resources created, environment setup, and how to run",
		Available:   true,
	})

	suggestions = append(suggestions, suggestion{
		ID:          "add-integration",
		Title:       "Add another Stripe feature",
		Description: "Subscriptions, Connect, billing portal, and more",
		Available:   true,
	})

	suggestions = append(suggestions, suggestion{
		ID:          "done",
		Title:       "I'm done",
		Description: "End the co-op session",
		Available:   true,
	})

	return suggestions
}

func buildSummarizePrompt(session *coop.Session) string {
	return fmt.Sprintf(`The developer wants a STRIPE.md summary. Create a STRIPE.md file in the project root with:

## What was built
- Integration: %s
- Blueprint steps completed

## Stripe resources created
- List any product IDs, price IDs, customer IDs created during the session

## Environment variables
- STRIPE_SECRET_KEY — your Stripe test secret key
- STRIPE_WEBHOOK_SECRET — webhook signing secret (from stripe listen)

## How to run
- Commands to install deps and start the server

## Webhook events handled
- List the events this integration listens for

## Useful links
- Stripe Dashboard: https://dashboard.stripe.com/test
- API docs: https://docs.stripe.com/api

After writing the file, run "stripe coop next-steps --session=%s --completed=summarize" again to offer more options.`, session.Blueprint, session.ID)
}

// projectEnvironment captures deploy targets detected in the working directory.
type projectEnvironment struct {
	HasStripeProjects bool
	HasVercel         bool
	HasFly            bool
	HasNetlify        bool
	HasExistingDeploy bool
}

func detectProjectEnvironment() projectEnvironment {
	env := projectEnvironment{}
	env.HasStripeProjects = fileExists("stripe.json") || dirExists(".stripe")
	env.HasVercel = fileExists("vercel.json") || fileExists(".vercel/project.json")
	env.HasFly = fileExists("fly.toml")
	env.HasNetlify = fileExists("netlify.toml")
	hasDocker := fileExists("Dockerfile") || fileExists("docker-compose.yml") || fileExists("docker-compose.yaml")
	hasRailway := fileExists("railway.json") || fileExists("railway.toml")
	env.HasExistingDeploy = env.HasVercel || env.HasFly || hasDocker || hasRailway || env.HasNetlify
	return env
}

func (env projectEnvironment) deployTarget() string {
	switch {
	case env.HasVercel:
		return "Vercel"
	case env.HasFly:
		return "Fly.io"
	case env.HasNetlify:
		return "Netlify"
	default:
		return "existing infrastructure"
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func dirExists(name string) bool {
	info, err := os.Stat(name)
	return err == nil && info.IsDir()
}
