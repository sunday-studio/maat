# Self Improvement Notes

These notes capture friction found while using Maat on the Maat repository itself.

## 2026-05-26: `maat initialize` Dogfood

Command run from `/Users/casprine/Desktop/vendor/sunday-studio/maat`:

```sh
maat initialize
```

Result:

- Maat registered the repo as project `maat`.
- Storage used the configured repo at `/Users/casprine/Desktop/vendor/personal/maat-storage`.
- The project record includes the local repo path and remote `git@github.com:sunday-studio/maat.git`.

Friction:

- The first run failed inside the sandbox because the configured storage repo was outside the writable workspace.
- The error was technically correct, but it did not suggest rerunning with access to the configured storage path.
- The installed `maat` binary was older than the current source checkout, so `maat initialize` printed the older long setup document instead of the shorter current design.
- A later read command attempted auto-pull and warned on `.git/FETCH_HEAD` permission failure, then continued and printed project state. That is useful, but the warning reads like a Git failure instead of a permissions/access hint.

Improvements:

- Make `maat initialize` print a short storage access hint when writes fail with `operation not permitted`.
- Add `maat doctor` or `maat setup check` to verify binary version, configured storage path, Git access, write access, pull access, and index access.
- Make `maat initialize` mention the binary version and source commit in `--agent-use` or verbose output so stale installs are easier to spot.
- Add a concise warning for auto-pull permission failures: "state read continued, but storage sync needs filesystem/Git access".
- Keep the generated initialize document short; the older installed binary showed why the long version is too much for agent setup.
