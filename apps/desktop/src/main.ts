import { invoke } from "@tauri-apps/api/core";
import "./styles.css";

type CommandRunResult = {
  ok: boolean;
  stdout: string;
  stderr: string;
  exit_code: number | null;
  duration_ms: number;
  cli_path: string;
};

type CliStatus = {
  cli_path: string;
  app_private_cli_path: string;
  version: CommandRunResult;
};

type StatusSummary = {
  projects: number;
  goals: number;
  active_goals: number;
  done_goals: number;
  tickets: number;
  open_tickets: number;
  done_tickets: number;
};

type ProjectListItem = {
  id?: string;
  key?: string;
  title?: string;
  status?: string;
  layout?: string;
};

type TicketItem = {
  id: string;
  title: string;
  status: string;
  goal_id?: string;
  description?: string;
  acceptance?: string[];
};

type SearchResult = {
  type: string;
  path: string;
  line: number;
  title?: string;
  excerpt?: string;
};

type AppState = {
  actor: string;
  storagePath: string;
  selectedProject: string;
  selectedTicket: string;
  ticketTitle: string;
  ticketDescription: string;
  ticketAcceptance: string;
  comment: string;
  evidence: string;
  query: string;
  cliStatus?: CliStatus;
  status?: StatusSummary;
  projects: ProjectListItem[];
  tickets: TicketItem[];
  results: SearchResult[];
  busy: boolean;
  message: string;
  error: string;
  lastCommand?: CommandRunResult;
};

const state: AppState = {
  actor: "desktop",
  storagePath: "",
  selectedProject: "maat",
  selectedTicket: "",
  ticketTitle: "",
  ticketDescription: "",
  ticketAcceptance: "",
  comment: "",
  evidence: "",
  query: "",
  projects: [],
  tickets: [],
  results: [],
  busy: false,
  message: "",
  error: ""
};

const app = document.querySelector<HTMLDivElement>("#app");

function hasTauriBridge(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

function storageArgs(): string[] {
  return state.storagePath.trim() ? ["--storage", state.storagePath.trim()] : [];
}

async function runMaat(args: string[]): Promise<CommandRunResult> {
  if (!hasTauriBridge()) {
    throw new Error("Open the desktop app with Tauri to run maat commands.");
  }
  const result = await invoke<CommandRunResult>("run_maat", { args });
  state.lastCommand = result;
  if (!result.ok) {
    throw new Error(result.stderr.trim() || result.stdout.trim() || `maat exited ${result.exit_code}`);
  }
  return result;
}

async function runJson<T>(args: string[]): Promise<T> {
  const result = await runMaat([...args, "--json"]);
  return JSON.parse(result.stdout) as T;
}

async function withBusy(action: () => Promise<void>): Promise<void> {
  state.busy = true;
  state.error = "";
  state.message = "";
  render();
  try {
    await action();
  } catch (error) {
    state.error = error instanceof Error ? error.message : String(error);
  } finally {
    state.busy = false;
    render();
  }
}

async function loadCli(): Promise<void> {
  if (!hasTauriBridge()) {
    throw new Error("Open the desktop app with Tauri to inspect the CLI.");
  }
  state.cliStatus = await invoke<CliStatus>("cli_status");
}

async function refresh(): Promise<void> {
  await withBusy(async () => {
    await loadCli();
    state.status = await runJson<StatusSummary>(["status", ...storageArgs()]);
    state.projects = await runJson<ProjectListItem[]>(["projects", ...storageArgs()]);
    await loadTickets();
    state.message = "Refreshed";
  });
}

async function setupStorage(): Promise<void> {
  await withBusy(async () => {
    await runJson([
      "setup",
      "--storage",
      state.storagePath.trim(),
      "--actor",
      state.actor.trim() || "desktop",
      "--auto-pull",
      "--auto-commit",
      "--no-auto-push"
    ]);
    await runJson(["setup", "doctor", "--storage", state.storagePath.trim(), "--fix"]);
    state.message = "Setup complete";
    await refresh();
  });
}

async function installCliFromPath(): Promise<void> {
  const sourcePath = window.prompt("Path to maat binary");
  if (!sourcePath) {
    return;
  }
  await withBusy(async () => {
    if (!hasTauriBridge()) {
      throw new Error("Open the desktop app with Tauri to install the CLI.");
    }
    await invoke("install_cli_from_path", { sourcePath });
    await loadCli();
    state.message = "CLI installed";
  });
}

async function loadTickets(): Promise<void> {
  const project = state.selectedProject.trim();
  if (!project) {
    state.tickets = [];
    return;
  }
  state.tickets = await runJson<TicketItem[]>(["ticket", "list", "--project", project, ...storageArgs()]);
}

async function createTicket(): Promise<void> {
  await withBusy(async () => {
    const acceptance = state.ticketAcceptance
      .split("\n")
      .map((value) => value.trim())
      .filter(Boolean);
    const args = [
      "ticket",
      "create",
      state.selectedProject.trim(),
      state.ticketTitle.trim(),
      "--description",
      state.ticketDescription.trim(),
      ...acceptance.flatMap((value) => ["--acceptance", value]),
      ...storageArgs()
    ];
    await runJson(args);
    state.ticketTitle = "";
    state.ticketDescription = "";
    state.ticketAcceptance = "";
    await loadTickets();
    state.message = "Ticket created";
  });
}

async function claimTicket(): Promise<void> {
  await withBusy(async () => {
    await runJson([
      "ticket",
      "claim",
      state.selectedTicket,
      "--project",
      state.selectedProject.trim(),
      "--agent",
      state.actor.trim() || "desktop",
      "--ttl",
      "2h",
      ...storageArgs()
    ]);
    await loadTickets();
    state.message = "Ticket claimed";
  });
}

async function commentTicket(): Promise<void> {
  await withBusy(async () => {
    await runJson([
      "ticket",
      "comment",
      state.selectedTicket,
      state.comment.trim(),
      "--project",
      state.selectedProject.trim(),
      ...storageArgs()
    ]);
    state.comment = "";
    state.message = "Comment added";
  });
}

async function completeTicket(): Promise<void> {
  await withBusy(async () => {
    await runJson([
      "ticket",
      "complete",
      state.selectedTicket,
      "--project",
      state.selectedProject.trim(),
      "--evidence",
      state.evidence.trim(),
      ...storageArgs()
    ]);
    state.evidence = "";
    await loadTickets();
    state.message = "Ticket completed";
  });
}

async function syncStatus(): Promise<void> {
  await withBusy(async () => {
    await runJson(["sync", "--status", ...storageArgs()]);
    state.message = "Sync status loaded";
  });
}

async function validateStore(): Promise<void> {
  await withBusy(async () => {
    await runJson(["validate", ...storageArgs()]);
    state.message = "Storage validates";
  });
}

async function rebuildIndex(): Promise<void> {
  await withBusy(async () => {
    await runMaat(["index", "rebuild", ...storageArgs()]);
    state.message = "Index rebuilt";
  });
}

async function search(): Promise<void> {
  await withBusy(async () => {
    state.results = await runJson<SearchResult[]>(["search", state.query.trim(), ...storageArgs()]);
    state.message = "Search complete";
  });
}

function setValue<K extends keyof AppState>(key: K, value: AppState[K]): void {
  state[key] = value;
}

function projectKey(project: ProjectListItem): string {
  return project.key || project.id || "";
}

function render(): void {
  if (!app) {
    return;
  }
  const selectedTicket = state.tickets.find((ticket) => ticket.id === state.selectedTicket);
  app.innerHTML = `
    <main>
      <header class="topbar">
        <div>
          <h1>Maat</h1>
          <p>${state.cliStatus?.cli_path || "CLI not checked"}</p>
        </div>
        <div class="actions">
          <button data-action="refresh">${state.busy ? "Working" : "Refresh"}</button>
          <button data-action="install-cli">Install CLI</button>
        </div>
      </header>

      ${state.error ? `<section class="notice error">${escapeHtml(state.error)}</section>` : ""}
      ${state.message ? `<section class="notice">${escapeHtml(state.message)}</section>` : ""}

      <section class="grid two">
        <form class="panel" data-form="setup">
          <h2>Setup</h2>
          <label>Storage path<input name="storagePath" value="${escapeAttr(state.storagePath)}" /></label>
          <label>Actor<input name="actor" value="${escapeAttr(state.actor)}" /></label>
          <button data-action="setup" type="button">Run setup</button>
        </form>

        <section class="panel metrics">
          <h2>Status</h2>
          ${metric("Projects", state.status?.projects)}
          ${metric("Goals", state.status?.goals)}
          ${metric("Open tickets", state.status?.open_tickets)}
          ${metric("Done tickets", state.status?.done_tickets)}
        </section>
      </section>

      <section class="grid two">
        <section class="panel">
          <h2>Projects</h2>
          <div class="list">
            ${state.projects
              .map((project) => {
                const key = projectKey(project);
                const active = key === state.selectedProject ? " active" : "";
                return `<button class="row${active}" data-project="${escapeAttr(key)}">
                  <span>${escapeHtml(project.title || key)}</span>
                  <small>${escapeHtml(project.status || "")}</small>
                </button>`;
              })
              .join("")}
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <h2>Tickets</h2>
            <input class="compact" data-bind="selectedProject" value="${escapeAttr(state.selectedProject)}" />
          </div>
          <div class="list">
            ${state.tickets
              .map((ticket) => {
                const active = ticket.id === state.selectedTicket ? " active" : "";
                return `<button class="row${active}" data-ticket="${escapeAttr(ticket.id)}">
                  <span>${escapeHtml(ticket.title)}</span>
                  <small>${escapeHtml(ticket.status)} ${escapeHtml(ticket.id)}</small>
                </button>`;
              })
              .join("")}
          </div>
        </section>
      </section>

      <section class="grid two">
        <form class="panel" data-form="create-ticket">
          <h2>Create Ticket</h2>
          <label>Title<input data-bind="ticketTitle" value="${escapeAttr(state.ticketTitle)}" /></label>
          <label>Description<textarea data-bind="ticketDescription">${escapeHtml(state.ticketDescription)}</textarea></label>
          <label>Acceptance<textarea data-bind="ticketAcceptance">${escapeHtml(state.ticketAcceptance)}</textarea></label>
          <button data-action="create-ticket" type="button">Create</button>
        </form>

        <section class="panel">
          <h2>Work Ticket</h2>
          <p class="selected">${escapeHtml(selectedTicket?.title || state.selectedTicket || "No ticket selected")}</p>
          <textarea data-bind="comment" placeholder="Comment">${escapeHtml(state.comment)}</textarea>
          <textarea data-bind="evidence" placeholder="Evidence">${escapeHtml(state.evidence)}</textarea>
          <div class="actions">
            <button data-action="claim" type="button">Claim</button>
            <button data-action="comment" type="button">Comment</button>
            <button data-action="complete" type="button">Complete</button>
          </div>
        </section>
      </section>

      <section class="grid two">
        <section class="panel">
          <h2>Sync</h2>
          <div class="actions">
            <button data-action="sync-status">Status</button>
            <button data-action="validate">Validate</button>
            <button data-action="rebuild-index">Rebuild Index</button>
          </div>
          <pre>${escapeHtml(state.lastCommand?.stdout || state.lastCommand?.stderr || "")}</pre>
        </section>

        <section class="panel">
          <h2>Search</h2>
          <label>Query<input data-bind="query" value="${escapeAttr(state.query)}" /></label>
          <button data-action="search">Search</button>
          <div class="list results">
            ${state.results
              .map((result) => `<div class="result">
                <strong>${escapeHtml(result.title || result.path)}</strong>
                <small>${escapeHtml(result.type)} ${escapeHtml(result.path)}:${result.line}</small>
                <p>${escapeHtml(result.excerpt || "")}</p>
              </div>`)
              .join("")}
          </div>
        </section>
      </section>
    </main>
  `;
}

function metric(label: string, value?: number): string {
  return `<div><span>${label}</span><strong>${value ?? "-"}</strong></div>`;
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function escapeAttr(value: string): string {
  return escapeHtml(value);
}

document.addEventListener("input", (event) => {
  const target = event.target as HTMLInputElement | HTMLTextAreaElement;
  const key = target.dataset.bind as keyof AppState | undefined;
  if (key) {
    setValue(key, target.value as never);
  }
});

document.addEventListener("click", (event) => {
  const target = event.target as HTMLElement;
  const project = target.closest<HTMLElement>("[data-project]")?.dataset.project;
  const ticket = target.closest<HTMLElement>("[data-ticket]")?.dataset.ticket;
  const action = target.closest<HTMLElement>("[data-action]")?.dataset.action;

  if (project) {
    state.selectedProject = project;
    void withBusy(loadTickets);
    return;
  }
  if (ticket) {
    state.selectedTicket = ticket;
    render();
    return;
  }

  switch (action) {
    case "refresh":
      void refresh();
      break;
    case "setup":
      void setupStorage();
      break;
    case "install-cli":
      void installCliFromPath();
      break;
    case "create-ticket":
      void createTicket();
      break;
    case "claim":
      void claimTicket();
      break;
    case "comment":
      void commentTicket();
      break;
    case "complete":
      void completeTicket();
      break;
    case "sync-status":
      void syncStatus();
      break;
    case "validate":
      void validateStore();
      break;
    case "rebuild-index":
      void rebuildIndex();
      break;
    case "search":
      void search();
      break;
  }
});

render();
if (hasTauriBridge()) {
  void refresh();
} else {
  state.message = "Desktop preview";
  render();
}
