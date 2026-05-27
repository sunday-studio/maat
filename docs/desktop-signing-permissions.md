# Desktop Signing And Permissions Hardening

This plan extends the macOS app architecture in `docs/macos-app-architecture.md`.
The desktop app remains a thin Tauri shell over the `maat` CLI: Markdown plus
Git stays authoritative, the SQLite index stays rebuildable, and the app does
not become a credential vault or storage owner.

## Target Outcome

- The distributed macOS app is signed with a Developer ID Application
  certificate.
- Every executable item inside the app bundle, including the bundled `maat`
  binary, is signed before the bundle is notarized.
- The DMG or ZIP artifact is notarized, stapled when supported by the artifact
  type, and validated before publication.
- The app asks for access to a storage repo only after the user chooses or
  creates that repo.
- Git authentication is delegated to the user's existing Git configuration,
  credential helpers, SSH agent, or Keychain-backed tooling. The desktop app
  does not store Git usernames, passwords, personal access tokens, private keys,
  or app-specific passwords for repository access.

## Release Pipeline Checkpoints

The macOS desktop release should run on a macOS runner because signing,
notarization, stapling, and Gatekeeper validation depend on Apple's platform
tools. Keep the existing Go release checks for the CLI, then add a desktop
release job with these checkpoints:

1. Build and test the CLI with the release version metadata.
2. Build the Tauri frontend and app shell.
3. Copy the release `maat` binary into the app bundle location chosen by the
   Tauri config, preferably `Contents/Resources/bin/maat` or
   `Contents/SharedSupport/bin/maat`.
4. Sign the bundled `maat` binary with the same Developer ID Application team as
   the app.
5. Build the macOS app bundle and DMG or ZIP artifact.
6. Verify the app bundle signature, including nested code.
7. Submit the distributable artifact to Apple's notarization service.
8. Staple the notarization ticket to the artifact or app bundle where supported.
9. Re-run local validation against the signed and stapled artifact.
10. Upload only artifacts that passed validation.

Required CI secrets should be limited to signing and notarization:

| Secret | Purpose |
| --- | --- |
| `APPLE_CERTIFICATE` | Base64-encoded Developer ID Application `.p12`. |
| `APPLE_CERTIFICATE_PASSWORD` | Password for the exported signing certificate. |
| `APPLE_SIGNING_IDENTITY` | Developer ID Application identity name or hash. |
| `APPLE_TEAM_ID` | Apple Developer team ID. |
| `APPLE_API_ISSUER` | App Store Connect issuer ID for notarization. |
| `APPLE_API_KEY` | App Store Connect API key ID. |
| `APPLE_API_KEY_P8` or `APPLE_API_KEY_PATH` | Private key material or runner path for notarization. |
| `KEYCHAIN_PASSWORD` | Temporary CI keychain password. |

Do not add Git hosting credentials for Maat storage repos to desktop release
secrets. Storage repo authentication is a runtime user concern and should remain
outside the release pipeline.

## Signing Rules

Use Developer ID Application signing for direct distribution outside the Mac App
Store. Ad-hoc signing is acceptable only for local development builds and must
not be published as a trusted release artifact.

Signing should be explicit about nested code. If the app ships with App Sandbox
enabled, sign the bundled CLI with the child-process entitlement file described
below. If the first signed build is not sandboxed, omit the CLI entitlement flag
but keep the same signing and verification order.

```sh
codesign --force --timestamp --options runtime \
  --entitlements src-tauri/CliEntitlements.plist \
  --sign "$APPLE_SIGNING_IDENTITY" \
  "src-tauri/target/release/bundle/macos/Maat.app/Contents/Resources/bin/maat"

codesign --force --timestamp --options runtime \
  --entitlements src-tauri/Entitlements.plist \
  --sign "$APPLE_SIGNING_IDENTITY" \
  "src-tauri/target/release/bundle/macos/Maat.app"
```

The exact bundle paths can change when the Tauri project is introduced, but the
ordering should not: sign nested executables first, then sign the containing app
bundle, then create the distributable archive.

Local validation must fail the release if any command fails:

```sh
codesign --verify --deep --strict --verbose=2 "Maat.app"
codesign --display --verbose=4 "Maat.app"
codesign --display --entitlements :- "Maat.app"
codesign --verify --strict --verbose=2 "Maat.app/Contents/Resources/bin/maat"
spctl --assess --type execute --verbose=4 "Maat.app"
```

The release log should show a Developer ID Application authority for both the
app and the bundled CLI. It should not show Apple Development, Mac Developer, or
ad-hoc authorities for published artifacts.

## Entitlements

Developer ID signing and notarization do not by themselves require App Sandbox.
The preferred hardening target is a sandboxed app because it lets macOS enforce
the user-selected storage boundary. If sandboxing prevents the bundled CLI, Git,
or the user's credential helpers from working in the first implementation, the
release may temporarily ship as a hardened-runtime Developer ID app without App
Sandbox, but it must still enforce user-selected storage in app logic and keep a
tracked follow-up to re-enable sandboxing.

Start with the narrowest entitlements that allow the app to launch, render the
Tauri webview, spawn the bundled CLI, persist access to a user-selected storage
repo, and use Git networking. Every added entitlement needs a product reason and
a validation step.

Recommended first-release app entitlements when sandboxing is enabled:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.app-sandbox</key>
  <true/>
  <key>com.apple.security.files.user-selected.read-write</key>
  <true/>
  <key>com.apple.security.files.bookmarks.app-scope</key>
  <true/>
  <key>com.apple.security.network.client</key>
  <true/>
</dict>
</plist>
```

Entitlement rationale:

| Entitlement | Reason | Boundary |
| --- | --- | --- |
| `com.apple.security.app-sandbox` | Keeps the desktop app inside macOS sandbox expectations. | Do not add broad home-directory or full-disk access. |
| `com.apple.security.files.user-selected.read-write` | Lets the user choose an existing storage repo or create one through an explicit picker. | Access starts only after selection and should be stored as a security-scoped bookmark if the sandbox requires durable access. |
| `com.apple.security.files.bookmarks.app-scope` | Allows durable app-scoped bookmarks for previously selected storage repos. | Only store bookmarks for paths selected through the setup or repo-switcher flow. |
| `com.apple.security.network.client` | Allows Git remotes, update checks, and CLI sync operations initiated by the user or configured Maat behavior. | The app still delegates Git auth to the user's Git tooling. |

Recommended bundled CLI entitlements when it is launched only as a child process
of the sandboxed app:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.app-sandbox</key>
  <true/>
  <key>com.apple.security.inherit</key>
  <true/>
</dict>
</plist>
```

The app-private CLI can use child-process inheritance because the desktop app is
its parent and owns the selected storage access. The optional terminal-facing
CLI install should be produced as a separate release binary without
`com.apple.security.inherit`, because terminal users run it directly and should
not depend on the desktop app's sandbox context.

Avoid these entitlements unless a later implementation proves they are required:

- `com.apple.security.files.downloads.read-write`;
- `com.apple.security.files.documents.read-write`;
- broad permanent file access outside the user-selected storage directory;
- camera, microphone, contacts, calendar, location, automation, and Apple Events
  entitlements.

The app does not need privileged helper tools, login items, system extensions,
endpoint security, file provider extensions, or full disk access for the desktop
workflow described in the architecture doc.

## Bundled CLI Handling

The bundled `maat` binary is product code and must be treated like nested
executable code in the app bundle.

- Build the CLI from the same commit and version as the desktop release.
- Place it in a deterministic app-bundle path.
- Sign it before signing the app bundle.
- At first launch, copy or install it to the app-private support path described
  in the architecture doc:

  ```text
  ~/Library/Application Support/maat/bin/maat
  ```

- Preserve executable permissions during copy or install.
- Verify the installed binary with `maat version --json` before using it for
  setup, reads, writes, sync, or updates.
- Store only the selected binary path and version metadata in app settings.

The app should prefer the app-private CLI for desktop operations. The optional
terminal-facing install into `PATH` must be a separate user action so the app
does not silently alter shell behavior.

## Notarization Path

Use Tauri's notarization support when it can perform the complete build, submit,
and staple flow for the selected artifact. If the release pipeline needs more
control, use Apple's tools directly after the app and nested CLI are signed.

Direct notarization flow:

```sh
xcrun notarytool submit "Maat.dmg" \
  --issuer "$APPLE_API_ISSUER" \
  --key-id "$APPLE_API_KEY" \
  --key "$APPLE_API_KEY_PATH" \
  --wait

xcrun stapler staple "Maat.dmg"
xcrun stapler validate "Maat.dmg"
spctl --assess --type open --context context:primary-signature --verbose=4 "Maat.dmg"
```

If notarization fails, the pipeline must fetch and preserve the notary log:

```sh
xcrun notarytool log "$NOTARY_SUBMISSION_ID" \
  --issuer "$APPLE_API_ISSUER" \
  --key-id "$APPLE_API_KEY" \
  --key "$APPLE_API_KEY_PATH" \
  notary-log.json
```

Release artifacts are publishable only when all of the following are true:

- `notarytool submit --wait` reports accepted status.
- stapling succeeds or the artifact type has a documented reason stapling is
  not applicable.
- `stapler validate` succeeds for stapled artifacts.
- `spctl` accepts the signed app and distributable artifact on a clean macOS
  machine or runner.
- the app can launch, run `maat version --json`, and complete first-run setup
  against a temporary user-selected storage repo.

## User-Selected Storage Access

The desktop app must not request broad filesystem access during launch. First
launch should open normally, install or verify the CLI, then ask the user to
create, clone, or select a storage repo.

Storage access rules:

- Use a native directory picker for selecting an existing storage repo.
- Use a native save/create location flow before creating a new local storage
  repo.
- Treat the selected path as user-owned data.
- Store the selected path in app settings only after `maat setup --storage
  <path> --json` and `maat setup doctor --storage <path> --json` succeed or
  return a user-actionable warning.
- If sandboxed durable access is required, store a security-scoped bookmark for
  the selected directory and renew access before spawning CLI commands.
- Pass the explicit storage path to CLI commands instead of relying on the
  process working directory.
- Show the path in error states so users can repair permissions or Git conflicts
  outside the app.

The app should not scan home directories, mounted volumes, or common Git folders
to find storage repos automatically. A recent-path picker is acceptable only for
paths that the user previously selected in the app.

## Credential Handling

Maat desktop should rely on Git's existing authentication paths:

- SSH remotes use the user's SSH config, keys, and agent.
- HTTPS remotes use the user's configured Git credential helper, including
  Keychain-backed helpers on macOS.
- Enterprise or custom helpers continue to run through Git because the CLI
  shells out to Git rather than implementing its own credential protocol.

The app may display Git failures returned by the CLI, but it must not collect or
persist Git secrets. Acceptable UI actions are:

- retry the failed sync after the user fixes credentials;
- open the storage repo in Finder;
- copy the failed Git remote host or command summary;
- open documentation for configuring Git credentials;
- switch auto-pull or auto-push settings by calling supported `maat setup`
  commands.

Disallowed behavior:

- storing Git passwords, personal access tokens, SSH private keys, or app
  passwords in app settings;
- embedding repository credentials in remote URLs;
- creating a custom credential cache for Maat desktop;
- logging full authenticated remote URLs;
- sending storage repo paths, remotes, or Git errors to telemetry without a
  separate privacy review.

## Implementation Checklist

- Add the Tauri macOS entitlements file with the first-release entitlement set.
- Configure `tauri.conf.json` to use the entitlements file and a Developer ID
  signing identity in release builds.
- Add a macOS release job that imports the Developer ID certificate into a
  temporary keychain.
- Build the release CLI and bundle it into the app.
- Sign the bundled CLI before signing the app bundle.
- Notarize the final DMG or ZIP.
- Staple and validate the notarization ticket where supported.
- Gate artifact upload on `codesign`, `stapler`, `spctl`, and launch smoke
  checks.
- Implement storage repo selection with native picker flows before CLI setup.
- Ensure every runtime CLI call uses the selected storage path.
- Route Git auth failures to user-actionable UI without collecting credentials.

## Acceptance Evidence For Implementation

An implementation can be accepted when a release candidate provides:

- CI logs showing CLI tests, app build, nested CLI signing, app signing,
  notarization, stapling, and Gatekeeper validation.
- `codesign --display --verbose=4` output for the app and bundled CLI showing
  Developer ID Application authority.
- `codesign --display --entitlements :-` output matching the approved
  entitlement set.
- `notarytool` accepted status and stapler validation output for the published
  artifact.
- a clean-machine launch smoke test showing first-run setup only requests a
  storage path after user selection.
- a sync smoke test using an existing Git credential helper, with no Maat
  desktop credential storage created.

## References

- Apple Developer Documentation: Notarizing macOS software before distribution.
- Tauri documentation: macOS Code Signing.
- Tauri documentation: macOS Application Bundle and entitlements.
