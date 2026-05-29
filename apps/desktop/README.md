# Maat Desktop

This is the Tauri macOS shell for Maat. It keeps the `maat` CLI as the product
API and never reads or writes Markdown state directly.

## Local Development

Install dependencies once:

```sh
npm install
```

Run the frontend type check:

```sh
npm run typecheck
```

Run the desktop app:

```sh
npm run tauri:dev
```

Use `MAAT_DESKTOP_CLI=/absolute/path/to/maat` to force the app to use a specific
CLI binary. Without that variable, the app first checks its app-private CLI path
and then falls back to `maat` on `PATH`.
