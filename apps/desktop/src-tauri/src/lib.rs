use serde::Serialize;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::Instant;
use tauri::{AppHandle, Manager};

#[derive(Debug, Serialize)]
struct CommandRunResult {
    ok: bool,
    stdout: String,
    stderr: String,
    exit_code: Option<i32>,
    duration_ms: u128,
    cli_path: String,
}

#[derive(Debug, Serialize)]
struct CliStatus {
    cli_path: String,
    app_private_cli_path: String,
    version: CommandRunResult,
}

#[tauri::command]
fn cli_status(app: AppHandle) -> Result<CliStatus, String> {
    let cli_path = resolve_cli_path(&app)?;
    let version = run_maat_inner(&app, vec!["version".to_string(), "--json".to_string()])?;
    Ok(CliStatus {
        cli_path,
        app_private_cli_path: app_private_cli_path(&app)?.display().to_string(),
        version,
    })
}

#[tauri::command]
fn run_maat(app: AppHandle, args: Vec<String>) -> Result<CommandRunResult, String> {
    run_maat_inner(&app, args)
}

#[tauri::command]
fn install_cli_from_path(app: AppHandle, source_path: String) -> Result<CliStatus, String> {
    let source = PathBuf::from(source_path);
    if !source.is_file() {
        return Err(format!("source binary does not exist: {}", source.display()));
    }
    let target = app_private_cli_path(&app)?;
    if let Some(parent) = target.parent() {
        fs::create_dir_all(parent).map_err(|err| format!("create CLI directory: {err}"))?;
    }
    fs::copy(&source, &target).map_err(|err| format!("install CLI: {err}"))?;
    make_executable(&target)?;
    cli_status(app)
}

fn run_maat_inner(app: &AppHandle, args: Vec<String>) -> Result<CommandRunResult, String> {
    validate_args(&args)?;
    let cli_path = resolve_cli_path(app)?;
    let started = Instant::now();
    let output = Command::new(&cli_path)
        .args(&args)
        .output()
        .map_err(|err| format!("run maat: {err}"))?;
    Ok(CommandRunResult {
        ok: output.status.success(),
        stdout: String::from_utf8_lossy(&output.stdout).to_string(),
        stderr: String::from_utf8_lossy(&output.stderr).to_string(),
        exit_code: output.status.code(),
        duration_ms: started.elapsed().as_millis(),
        cli_path,
    })
}

fn validate_args(args: &[String]) -> Result<(), String> {
    let Some(command) = args.first() else {
        return Err("missing maat command".to_string());
    };
    let allowed = [
        "catalog", "goal", "index", "project", "projects", "search", "setup", "status", "sync",
        "ticket", "validate", "version",
    ];
    if !allowed.contains(&command.as_str()) {
        return Err(format!("maat command is not allowed: {command}"));
    }
    if args.iter().any(|arg| arg.contains('\0')) {
        return Err("maat argument contains a null byte".to_string());
    }
    Ok(())
}

fn resolve_cli_path(app: &AppHandle) -> Result<String, String> {
    if let Ok(path) = env::var("MAAT_DESKTOP_CLI") {
        if Path::new(&path).is_file() {
            return Ok(path);
        }
    }
    let app_private = app_private_cli_path(app)?;
    if app_private.is_file() {
        return Ok(app_private.display().to_string());
    }
    Ok("maat".to_string())
}

fn app_private_cli_path(app: &AppHandle) -> Result<PathBuf, String> {
    Ok(app
        .path()
        .app_config_dir()
        .map_err(|err| format!("resolve app config directory: {err}"))?
        .join("bin")
        .join("maat"))
}

#[cfg(unix)]
fn make_executable(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    let mut permissions = fs::metadata(path)
        .map_err(|err| format!("read installed CLI permissions: {err}"))?
        .permissions();
    permissions.set_mode(0o755);
    fs::set_permissions(path, permissions)
        .map_err(|err| format!("mark installed CLI executable: {err}"))
}

#[cfg(not(unix))]
fn make_executable(_path: &Path) -> Result<(), String> {
    Ok(())
}

pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            cli_status,
            install_cli_from_path,
            run_maat
        ])
        .run(tauri::generate_context!())
        .expect("error while running Maat desktop");
}
