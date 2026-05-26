package maat

import (
	"fmt"
	"strings"
)

const agentInstructionsSnippet = `Use Maat as the canonical project memory for this repo. Before material work, run ` + "`maat sync`" + ` if available, then inspect state with ` + "`maat status`" + `, ` + "`maat project show <project>`" + `, or ` + "`maat search <query>`" + `. Create or claim a ticket before working. Record meaningful progress with ticket comments or events. When finished, complete the ticket with evidence, validate Maat, and sync. Do not mark work done without evidence.`

func agentInstructionsSnippetText() string {
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
maat version
`+"```"+`

2. Clone or locate the Maat storage repo. It must be a Git-controlled directory that agents can read, write, commit, pull, and push.

3. Link this machine or sandbox to that storage directory:

`+"```sh"+`
maat init %[2]s
# or
maat storage link %[2]s
`+"```"+`

4. Check the current state:

`+"```sh"+`
maat sync --storage %[2]s --status
maat status --storage %[2]s
maat projects --storage %[2]s
`+"```"+`

## Save Into The Agent's Instructions

Save this setup document, or the shorter snippet below, into the instruction file or memory surface that the agent reads before it starts work:

- Codex: add it to the repo's `+"`AGENTS.md`"+` or to a Codex skill/instruction file for this workspace.
- Claude Code: add it to `+"`CLAUDE.md`"+` in the repo, or to the user-level Claude instruction file if this should apply everywhere.
- Cursor or Cursor Cloud: add it to the repo's Cursor rules or project instructions.
- Cloud or hosted agents: add it to the project prompt, system prompt, memory, or skill document that the agent reads before work.
- Other agents: use the closest equivalent persistent instruction file.

Use this exact project-level snippet when the target file already has other instructions:

`+"```text"+`
%[3]s
`+"```"+`

After saving it, follow the command loop below. Do not rely on the human to manually update Maat state.

## Operating Loop

This repo is registered in Maat as ` + "`%[1]s`" + `. Use that exact project key in all project-scoped commands.

Before material work:

`+"```sh"+`
maat sync --storage %[2]s --status
maat status --storage %[2]s
maat project show %[1]s --storage %[2]s
maat search "<query>" --storage %[2]s
`+"```"+`

When planning work:

`+"```sh"+`
maat goal create %[1]s "<goal title>" --storage %[2]s
maat ticket create %[1]s "<ticket title>" --goal <goal-id> --storage %[2]s
`+"```"+`

When starting work:

`+"```sh"+`
maat ticket claim <ticket-id> --project %[1]s --agent "<agent-id>" --ttl 2h --storage %[2]s
`+"```"+`

During work:

`+"```sh"+`
maat ticket comment <ticket-id> "short factual progress note" --project %[1]s --storage %[2]s
maat search "<thing you need>" --storage %[2]s
`+"```"+`

When finished:

`+"```sh"+`
maat ticket complete <ticket-id> --project %[1]s --evidence "tests, commit, PR, or exact verification" --storage %[2]s
maat validate --storage %[2]s
maat sync --storage %[2]s --message "status(%[1]s): update maat" --push
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
		project, storage, agentInstructionsSnippetText())
}
