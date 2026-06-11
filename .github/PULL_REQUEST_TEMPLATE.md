<!-- Title: use conventional commit format — type(scope): description -->
<!-- e.g. feat(search): add fuzzy matching for doc queries           -->
<!-- Types: feat fix refactor docs test chore build ci style perf    -->

### Summary and motivation

<!-- Summary: Use one concise sentence describing what changed. Keep this short. -->

<!-- Context (complex changes only): Add a blank line above this paragraph. Describe context around our change that a reviewer might not be familiar with. -->

<!-- Motivation (complex changes only): Add a blank line above this paragraph. Why did we make this change? Include relevant documents and links (issues, discussions, other PRs, etc.) -->

### Test plan

<!-- Update the following when our changes include unit tests, and if we've tested them manually -->

- [ ] Changes are covered by unit tests (including failures and edge cases)
- [ ] Changes were tested manually

<!-- Unit tests: If we added unit tests, briefly describe what they're testing at a high-level. Keep this brief, using only 1-2 sentences. -->

<!-- Instructions: write detailed instructions explaining how we tested our changes manually, or how a reviewer could verify our work. -->

<!-- - Include code snippets when testing CLI changes -->
<!-- - Include screenshots where relevant -->

<!-- Example: -->
<!-- Run the following command to test the change: -->
<!-- ```bash -->
<!-- stripe docs search "webhooks" -->
<!-- ``` -->
<!-- You should see a list of matching docs pages printed to the terminal. -->

### Checklist

- [ ] PR title follows [conventional commit format](https://www.conventionalcommits.org/en/v1.0.0/) (`type(scope): description`)
- [ ] Commit messages follow [conventional commit format](https://www.conventionalcommits.org/en/v1.0.0/)
- [ ] I have read [CONTRIBUTING.md](../CONTRIBUTING.md)
- [ ] I have added/updated tests where appropriate
- [ ] I have updated documentation where appropriate
- [ ] This does not introduce a breaking change (if it does, I have described the migration path above)
