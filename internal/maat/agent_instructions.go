package maat

import (
	"fmt"
	"strings"
)

const agentInstructionsSnippet = `Use Maat as the canonical project memory for this repo. Before material work, run ` + "`matt sync`" + ` if available, then inspect state with ` + "`matt status`" + `, ` + "`matt project show <project>`" + `, or ` + "`matt search <query>`" + `. Create or claim a ticket before working. Record meaningful progress with ticket comments or events. When finished, complete the ticket with evidence, validate Maat, and sync. Do not mark work done without evidence.`

// AgentInstructionsSnippet returns the minimal instructions that should be
// copied into project-level AGENTS.md files so agents know how to report work
// into Maat.
func AgentInstructionsSnippet() string {
	return agentInstructionsSnippet
}

type AgentSetupOptions struct {
	ProjectKey  string
	StoragePath string
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

	return fmt.Sprintf(`# Maat Agent Setup

Audience: any agent that can read files, run shell commands, and update Git

Maat is the project memory for this work. Markdown plus Git is the source of truth. SQLite is only a local search cache.

## Install And Link

1. Confirm the CLI is available:

`+"```sh"+`
matt version
`+"```"+`

2. Clone or locate the Maat storage repo. It must be a Git-controlled directory that agents can read, write, commit, pull, and push.

3. Link this machine or sandbox to that storage directory:

`+"```sh"+`
matt init %[2]s
# or
matt storage link %[2]s
`+"```"+`

4. Check the current state:

`+"```sh"+`
matt sync --storage %[2]s --status
matt status --storage %[2]s
matt projects --storage %[2]s
`+"```"+`

## Install Into The Agent

Put the Maat rule into the agent's normal instruction surface:

- Codex: add it to the repo's `+"`AGENTS.md`"+` or to a Codex skill/instruction file for this workspace.
- Claude Code: add it to `+"`CLAUDE.md`"+` in the repo, or to the user-level Claude instruction file if this should apply everywhere.
- Cloud or hosted agents: add it to the system prompt, project instructions, memory, or skill document that the agent reads before work.
- Other agents: use the closest equivalent persistent instruction file.

Use this exact project-level snippet when the target file already has other instructions:

`+"```text"+`
%[3]s
`+"```"+`

You can write the snippet directly with:

`+"```sh"+`
matt agent instructions --output AGENTS.md
`+"```"+`

## Operating Loop

Before material work:

`+"```sh"+`
matt sync --storage %[2]s --status
matt status --storage %[2]s
matt project show %[1]s --storage %[2]s
matt search "<query>" --storage %[2]s
`+"```"+`

If the project is not linked yet:

`+"```sh"+`
matt project link . --storage %[2]s --key %[1]s --name "<display-name>"
`+"```"+`

When planning work:

`+"```sh"+`
matt goal create %[1]s "<goal title>" --storage %[2]s
matt ticket create %[1]s "<ticket title>" --goal <goal-id> --storage %[2]s
`+"```"+`

When starting work:

`+"```sh"+`
matt ticket claim <ticket-id> --project %[1]s --agent "<agent-id>" --ttl 2h --storage %[2]s
`+"```"+`

During work:

`+"```sh"+`
matt ticket comment <ticket-id> "short factual progress note" --project %[1]s --storage %[2]s
matt search "<thing you need>" --storage %[2]s
`+"```"+`

When finished:

`+"```sh"+`
matt ticket complete <ticket-id> --project %[1]s --evidence "tests, commit, PR, or exact verification" --storage %[2]s
matt validate --storage %[2]s
matt sync --storage %[2]s --message "status(%[1]s): update maat" --push
`+"```"+`

## Rules

- Create or claim a ticket before material work.
- Add comments for meaningful progress, blockers, handoffs, and decisions.
- Complete a ticket only when there is clear evidence.
- Treat index warnings as cache warnings. The Markdown state may already be written; do not retry blindly.
- Commit finished product changes in the product repo.
- Commit and push Maat storage changes when the storage repo is Git-controlled.
- Do not store primary project state outside Markdown.
`,
		project, storage, AgentInstructionsSnippet())
}
