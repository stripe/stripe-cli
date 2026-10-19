package releasenotes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func lines(l ...string) string {
	return strings.Join(l, "\n")
}

func TestTidy(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			// The notes GitHub generated for v1.50.11.
			name: "strips every kind of boilerplate",
			body: lines(
				"<!-- Release notes generated using configuration in .github/release.yml at v1.50.11 -->",
				"",
				"## What's Changed",
				"### Other user-facing changes",
				"* Add `stripe plugin uninstall --all` and document uninstalling by @vzhang-stripe in https://github.com/stripe/stripe-cli/pull/1974",
				"* Fold `stripe reauth` into `stripe login` by @vcheung-stripe in https://github.com/stripe/stripe-cli/pull/1995",
				"",
				"## New Contributors",
				"* @enzo-stripe made their first contribution in https://github.com/stripe/stripe-cli/pull/1865",
				"",
				"**Full Changelog**: https://github.com/stripe/stripe-cli/compare/v1.50.10...v1.50.11",
				"",
				"",
			),
			want: lines(
				"<!-- Release notes generated using configuration in .github/release.yml at v1.50.11 -->",
				"",
				"## What's Changed",
				"",
				"* Add `stripe plugin uninstall --all` and document uninstalling (#1974)",
				"* Fold `stripe reauth` into `stripe login` (#1995)",
			),
		},
		{
			name: "flattens every category into one list",
			body: lines(
				"## What's Changed",
				"### Breaking changes",
				"* Drop support for `stripe reauth` by @vcheung-stripe in https://github.com/stripe/stripe-cli/pull/1995",
				"### Bug fixes",
				"* Let Ctrl+C interrupt a plugin download by @vzhang-stripe in https://github.com/stripe/stripe-cli/pull/2033",
				"#### A nested heading",
				"* Something nested by @someone in https://github.com/stripe/stripe-cli/pull/9",
			),
			want: lines(
				"## What's Changed",
				"",
				"* Drop support for `stripe reauth` (#1995)",
				"* Let Ctrl+C interrupt a plugin download (#2033)",
				"* Something nested (#9)",
			),
		},
		{
			name: "handles fork pull requests and bot authors",
			body: lines(
				"## What's Changed",
				"* Let Ctrl+C interrupt a plugin download by @outside-contributor in #2033",
				"* Bump the pinned action digests by @dependabot[bot] in https://github.com/stripe/stripe-cli/pull/2001",
			),
			want: lines(
				"## What's Changed",
				"",
				"* Let Ctrl+C interrupt a plugin download (#2033)",
				"* Bump the pinned action digests (#2001)",
			),
		},
		{
			name: "leaves an entry without an author suffix alone",
			body: lines(
				"## What's Changed",
				"* A hand-written entry with no author suffix",
			),
			want: lines(
				"## What's Changed",
				"",
				"* A hand-written entry with no author suffix",
			),
		},
		{
			name: "keeps content that follows the new contributors section",
			body: lines(
				"## What's Changed",
				"* Fold `stripe reauth` into `stripe login` by @vcheung-stripe in https://github.com/stripe/stripe-cli/pull/1995",
				"",
				"## New Contributors",
				"* @enzo-stripe made their first contribution in https://github.com/stripe/stripe-cli/pull/1865",
				"",
				"## Upgrading",
				"Run `stripe upgrade`.",
			),
			want: lines(
				"## What's Changed",
				"",
				"* Fold `stripe reauth` into `stripe login` (#1995)",
				"",
				"## Upgrading",
				"",
				"Run `stripe upgrade`.",
			),
		},
		{
			name: "returns nothing for an empty body",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Tidy(tt.body))
		})
	}
}

func TestTidyIsIdempotent(t *testing.T) {
	body := lines(
		"<!-- Release notes generated using configuration in .github/release.yml at v1.50.11 -->",
		"",
		"## What's Changed",
		"### Other user-facing changes",
		"* Fold `stripe reauth` into `stripe login` by @vcheung-stripe in https://github.com/stripe/stripe-cli/pull/1995",
		"",
		"**Full Changelog**: https://github.com/stripe/stripe-cli/compare/v1.50.10...v1.50.11",
	)

	once := Tidy(body)
	assert.Equal(t, once, Tidy(once))
}
