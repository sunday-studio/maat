package maat

const agentInstructionsSnippet = `Use Maat as the canonical project memory for this repo. Before material work, run ` + "`matt sync`" + ` if available, then inspect state with ` + "`matt status`" + `, ` + "`matt project show <project>`" + `, or ` + "`matt search <query>`" + `. Create or claim a ticket before working. Record meaningful progress with ticket comments or events. When finished, complete the ticket with evidence, validate Maat, and sync. Do not mark work done without evidence.`

// AgentInstructionsSnippet returns the minimal instructions that should be
// copied into project-level AGENTS.md files so agents know how to report work
// into Maat.
func AgentInstructionsSnippet() string {
	return agentInstructionsSnippet
}
