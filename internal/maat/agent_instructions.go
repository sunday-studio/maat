package maat

import (
	"fmt"
	"strings"
)

const agentInstructionsSnippet = `Use Maat as the canonical project memory for this repo. Before work, run ` + "`maat status`" + ` and ` + "`maat project show <project>`" + `. Create or claim a ticket before material work. New goals must include an outcome; new tickets must include a description and acceptance criteria. Add short progress comments when useful. Complete tickets only with evidence. Validate and sync when done.`

func agentInstructionsSnippetText() string {
	return agentInstructionsSnippet
}

type AgentSetupOptions struct {
	ProjectKey    string
	StoragePath   string
	BinaryVersion string
}

// AgentSetupDocument returns a complete setup document for installing Maat into
// an agent's normal instruction, skill, or memory system.
func AgentSetupDocument(opts AgentSetupOptions) string {
	project := strings.TrimSpace(opts.ProjectKey)
	if project == "" {
		project = "<project-key>"
	}
	storage := strings.TrimSpace(opts.StoragePath)
	if storage == "" {
		storage = "/absolute/path/to/maat-state"
	}
	binaryVersion := strings.TrimSpace(opts.BinaryVersion)
	if binaryVersion == "" {
		binaryVersion = "run `maat version` to confirm the installed binary"
	}

	return fmt.Sprintf(`# Maat Agent Instructions

Maat is the project memory. Markdown plus Git is the source of truth. SQLite is only a local search cache.

This repo is registered as `+"`%[1]s`"+`.
Maat binary: %[4]s.

## First Run

Run this from the project repo through the agent you want to teach:

`+"```sh"+`
maat version
maat setup --storage %[2]s
maat initialize --project %[1]s --storage %[2]s
`+"```"+`

The initialize command prints the text the agent should save into its normal instruction file.

Save the snippet below into `+"`AGENTS.md`"+`, `+"`CLAUDE.md`"+`, Cursor rules, or the closest equivalent instruction surface:

`+"```text"+`
%[3]s
`+"```"+`

## Daily Loop

`+"```sh"+`
maat sync --storage %[2]s --status
maat status --storage %[2]s
maat project show %[1]s --storage %[2]s
maat search "<query>" --storage %[2]s

maat goal create %[1]s "<goal title>" --outcome "the concrete outcome this goal should achieve" --storage %[2]s
maat ticket create %[1]s "<ticket title>" --goal <goal-id> --description "the concrete work another agent should do" --acceptance "clear completion condition" --storage %[2]s

maat ticket claim <ticket-id> --project %[1]s --agent "<agent-id>" --ttl 2h --storage %[2]s
maat ticket comment <ticket-id> "short factual progress note" --project %[1]s --storage %[2]s
maat ticket complete <ticket-id> --project %[1]s --evidence "tests, commit, PR, or exact verification" --storage %[2]s
maat validate --storage %[2]s
maat sync --storage %[2]s --message "status(%[1]s): update maat" --push
`+"```"+`

## Next Steps

- Check project state: `+"`maat status --storage %[2]s`"+`;
- List or inspect work: `+"`maat ticket list --project %[1]s --storage %[2]s`"+` and `+"`maat project show %[1]s --storage %[2]s`"+`;
- Create structured work: `+"`maat goal create %[1]s \"<goal title>\" --outcome \"<outcome>\" --storage %[2]s`"+`;
- Add a ticket: `+"`maat ticket create %[1]s \"<ticket title>\" --description \"<work>\" --acceptance \"<done condition>\" --storage %[2]s`"+`;
- Finish with evidence: `+"`maat ticket complete <ticket-id> --project %[1]s --evidence \"<verification>\" --storage %[2]s`"+`;
- Validate and share: `+"`maat validate --storage %[2]s`"+` then `+"`maat sync --storage %[2]s --status`"+`;

## Rules

- Create or claim a ticket before material work.
- Never create title-only goals or tickets.
- Every new goal must include a concrete outcome.
- Every new ticket must include a concrete description and at least one acceptance criterion.
- Add comments for meaningful progress, blockers, handoffs, and decisions.
- Complete a ticket only when there is clear evidence.
- Commit finished product changes in the product repo.
- Commit and push Maat storage changes.
- Do not store primary project state outside Markdown.
`,
		project, storage, agentInstructionsSnippetText(), binaryVersion)
}
