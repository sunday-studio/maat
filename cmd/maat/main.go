package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/sunday-studio/maat/internal/maat"
	"github.com/sunday-studio/maat/internal/version"
)

var agentUse bool
var jsonUse bool

var latestReleaseURL = "https://api.github.com/repos/sunday-studio/maat/releases/latest"
var updateHTTPClient = http.DefaultClient

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "maat: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	agentUse = false
	jsonUse = false
	var err error
	args, err = splitGlobalFlags(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return nil
	case "version":
		return versionCommand(args[1:])
	case "update":
		return updateCommand(args[1:])
	case "uninstall":
		return uninstallCommand(args[1:])
	case "setup":
		return setupCommand(args[1:])
	case "initialize":
		return agentInitializeCommand(args[1:])
	case "index":
		return indexCommand(args[1:])
	case "projects":
		filtered, jsonOut := splitJSONFlag(args[1:])
		store, err := loadStoreForRead(filtered)
		if err != nil {
			return err
		}
		progress("projects.load", "loading projects", map[string]string{"storage": store})
		projects, err := loadProjectListItems(store)
		if err != nil {
			return err
		}
		if agentUse {
			return agentUpdate("projects.loaded", "ok", "projects loaded", map[string]any{
				"projects": projects,
				"count":    len(projects),
			})
		}
		if jsonOut {
			return writeJSON(projects)
		}
		ok("projects.loaded", fmt.Sprintf("loaded %d projects", len(projects)), nil)
		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return nil
		}
		fmt.Printf("%-18s %-9s %-8s %s\n", "Project", "Status", "Layout", "Title")
		for _, project := range projects {
			fmt.Printf("%-18s %-9s %-8s %s\n", project.ID, colorStatus(project.Status), project.Layout, project.Title)
		}
		return nil
	case "project":
		return projectCommand(args[1:])
	case "goal":
		return goalCommand(args[1:])
	case "ticket":
		return ticketCommand(args[1:])
	case "status":
		filtered, jsonOut := splitJSONFlag(args[1:])
		store, err := loadStoreForRead(filtered)
		if err != nil {
			return err
		}
		summary, err := maat.Status(store)
		if err != nil {
			return err
		}
		if agentUse {
			return agentUpdate("status.ready", "ok", "status ready", summary)
		}
		if jsonOut {
			return writeJSON(summary)
		}
		ok("status.ready", "status ready", nil)
		fmt.Println("Maat status")
		printField("Projects", colorNumber(summary.Projects))
		printField("Goals", fmt.Sprintf("%s active, %s done, %s total", colorNumber(summary.ActiveGoals), colorNumber(summary.DoneGoals), colorNumber(summary.Goals)))
		printField("Tickets", fmt.Sprintf("%s open, %s done, %s total", colorNumber(summary.OpenTickets), colorNumber(summary.DoneTickets), colorNumber(summary.Tickets)))
		return nil
	case "validate":
		return validateCommand(args[1:])
	case "sync":
		return syncCommand(args[1:])
	case "search":
		return searchCommand(args[1:])
	case "tui":
		return tuiCommand(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Print(`maat - Git-backed project state for agents

Usage:
  maat <command> [flags]

Common:
  maat setup [--storage <absolute-git-repo-path>] [--actor <name>] [--json]
  maat setup doctor [--storage <path>] [--fix] [--json]
  maat initialize [--project <project-key>] [--storage <path>] [--json]
  maat status [--storage <path>] [--json]
  maat projects [--storage <path>] [--json]
  maat search <query> [--storage <path>] [--json]
  maat sync [--storage <path>] [--message <msg>] [--push] [--status] [--json]

Projects:
  maat project link [source-path] [--storage <path>] [--key <project-key>] [--name <display-name>] [--json]
  maat project show <project-id> [--storage <path>] [--json]
  maat goal create [project-key] <title> --outcome <text> [--storage <path>] [--json]

Tickets:
  maat ticket create [project-key] <title> [--goal <goal-id>] --description <text> --acceptance <text>... [--storage <path>] [--json]
  maat ticket list [--project <project-key>] [--storage <path>] [--json]
  maat ticket show <ticket-id> [--project <project-key>] [--storage <path>] [--json]
  maat ticket claim <ticket-id> [--agent <agent>] [--ttl <duration>] [--project <project-key>] [--storage <path>] [--json]
  maat ticket comment <ticket-id> <comment> [--project <project-key>] [--storage <path>] [--json]
  maat ticket complete <ticket-id> --evidence <text> [--project <project-key>] [--storage <path>] [--json]

Setup and maintenance:
  maat setup [--storage <absolute-git-repo-path>] [--actor <name>] [--auto-pull|--no-auto-pull] [--auto-commit|--no-auto-commit] [--auto-push|--no-auto-push] [--json]
  maat setup doctor [--storage <path>] [--fix] [--json]
  maat update [--source <path>] [--install-dir <path>] [--binary-name <name>] [--json]
  maat uninstall [--install-dir <path>] [--binary-name <name>] [--purge-config] [--json]
  maat index rebuild [--storage <path>]
  maat validate [--storage <path>] [--json]
  maat tui [--storage <path>]
  maat version [--json]

Global flags:
  --agent-use   emit newline-delimited JSON updates for agents

Markdown plus Git is the source of truth. The SQLite index is a rebuildable local cache.
`)
}

func versionCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	if len(filtered) > 0 {
		return errors.New("usage: maat version [--json]")
	}
	info := version.Current()
	if agentUse {
		return agentUpdate("version.ready", "ok", "version ready", info)
	}
	if jsonOut {
		return writeJSON(info)
	}
	fmt.Println(info.String())
	return nil
}

type installCommandResult struct {
	Action           string `json:"action"`
	BinaryName       string `json:"binary_name"`
	InstallDir       string `json:"install_dir"`
	TargetPath       string `json:"target_path"`
	SourcePath       string `json:"source_path,omitempty"`
	InstallRecorded  bool   `json:"install_recorded,omitempty"`
	CurrentVersion   string `json:"current_version,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	AssetName        string `json:"asset_name,omitempty"`
	AssetURL         string `json:"asset_url,omitempty"`
	ChecksumVerified bool   `json:"checksum_verified,omitempty"`
	Removed          bool   `json:"removed,omitempty"`
	ConfigPath       string `json:"config_path,omitempty"`
	ConfigPurged     bool   `json:"config_purged,omitempty"`
}

type githubRelease struct {
	TagName string               `json:"tag_name"`
	Name    string               `json:"name"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func updateCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	sourcePath := ""
	installDir := ""
	binaryName := defaultBinaryName()
	for i := 0; i < len(filtered); i++ {
		switch filtered[i] {
		case "--source":
			if i+1 >= len(filtered) {
				return errors.New("--source requires a path")
			}
			sourcePath = filtered[i+1]
			i++
		case "--install-dir":
			if i+1 >= len(filtered) {
				return errors.New("--install-dir requires a path")
			}
			installDir = filtered[i+1]
			i++
		case "--binary-name":
			if i+1 >= len(filtered) {
				return errors.New("--binary-name requires a name")
			}
			binaryName = strings.TrimSpace(filtered[i+1])
			i++
		default:
			return fmt.Errorf("unexpected update argument %q", filtered[i])
		}
	}
	if binaryName == "" {
		return errors.New("--binary-name cannot be empty")
	}
	if installDir == "" {
		installDir = defaultUpdateInstallDir(binaryName)
	}
	absInstallDir, err := filepath.Abs(installDir)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(absInstallDir, binaryName)

	var sourceInfo os.FileInfo
	var absSource string
	result := installCommandResult{
		BinaryName:     binaryName,
		InstallDir:     absInstallDir,
		TargetPath:     targetPath,
		CurrentVersion: version.Current().Version,
	}

	cleanupSource := func() {}
	defer cleanupSource()
	if sourcePath == "" {
		progress("update.release", "checking latest GitHub release", map[string]string{"current_version": result.CurrentVersion})
		release, err := fetchLatestRelease()
		if err != nil {
			return err
		}
		result.LatestVersion = release.TagName
		if release.Name != "" && result.LatestVersion == "" {
			result.LatestVersion = release.Name
		}
		if releaseIsCurrent(result.CurrentVersion, result.LatestVersion) {
			result.Action = "update.current"
			if agentUse {
				return agentUpdate("update.ready", "ok", "already on latest version", result)
			}
			if jsonOut {
				return writeJSON(result)
			}
			ok("update.ready", "already on latest version", nil)
			printField("Version", result.CurrentVersion)
			return nil
		}
		asset, ok := selectReleaseAsset(release, binaryName, runtime.GOOS, runtime.GOARCH)
		if !ok {
			return fmt.Errorf("no release asset found for %s/%s in %s", runtime.GOOS, runtime.GOARCH, result.LatestVersion)
		}
		result.AssetName = asset.Name
		result.AssetURL = asset.BrowserDownloadURL
		progress("update.download", "downloading release asset", map[string]string{"asset": asset.Name})
		archiveData, err := downloadURL(asset.BrowserDownloadURL)
		if err != nil {
			return err
		}
		if checksumAsset, ok := selectChecksumAsset(release); ok {
			checksums, err := downloadURL(checksumAsset.BrowserDownloadURL)
			if err != nil {
				return err
			}
			if err := verifyArchiveChecksum(archiveData, asset.Name, string(checksums)); err != nil {
				return err
			}
			result.ChecksumVerified = true
		}
		tmpPath, err := materializeReleaseBinary(archiveData, asset.Name, binaryName)
		if err != nil {
			return err
		}
		absSource = tmpPath
		cleanupSource = func() { _ = os.Remove(tmpPath) }
		sourceInfo, err = os.Stat(absSource)
		if err != nil {
			return err
		}
	} else {
		var err error
		absSource, err = filepath.Abs(sourcePath)
		if err != nil {
			return err
		}
		sourceInfo, err = os.Stat(absSource)
		if err != nil {
			return fmt.Errorf("read update source: %w", err)
		}
		if sourceInfo.IsDir() {
			return fmt.Errorf("update source is a directory: %s", absSource)
		}
	}
	result.SourcePath = absSource
	if samePath(absSource, targetPath) {
		result.Action = "update.skipped"
		cfgPath, err := rememberInstalledBinary(binaryName, absInstallDir, targetPath)
		if err != nil {
			return err
		}
		result.ConfigPath = cfgPath
		result.InstallRecorded = true
		if agentUse {
			return agentUpdate("update.ready", "ok", "binary already installed at target path", result)
		}
		if jsonOut {
			return writeJSON(result)
		}
		ok("update.ready", "binary already installed at target path", nil)
		printField("Binary", binaryName)
		printField("Target", targetPath)
		return nil
	}
	progress("update.prepare", "preparing install directory", map[string]string{"install_dir": absInstallDir})
	if err := os.MkdirAll(absInstallDir, 0o755); err != nil {
		return err
	}
	progress("update.install", "installing binary", map[string]string{"source": absSource, "target": targetPath})
	if err := installBinary(absSource, targetPath, sourceInfo.Mode().Perm()); err != nil {
		return err
	}
	result.Action = "update.installed"
	cfgPath, err := rememberInstalledBinary(binaryName, absInstallDir, targetPath)
	if err != nil {
		return err
	}
	result.ConfigPath = cfgPath
	result.InstallRecorded = true
	if agentUse {
		return agentUpdate("update.ready", "ok", "binary updated", result)
	}
	if jsonOut {
		return writeJSON(result)
	}
	ok("update.ready", "binary updated", nil)
	printField("Binary", binaryName)
	if result.LatestVersion != "" {
		printField("Version", result.LatestVersion)
	}
	printField("Source", absSource)
	printField("Target", targetPath)
	if result.ChecksumVerified {
		printField("Checksum", "verified")
	}
	if !dirOnPath(absInstallDir) {
		warn("update.path", absInstallDir+" is not on PATH", nil)
	}
	return nil
}

func uninstallCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	installDir := ""
	installDirProvided := false
	binaryName := defaultBinaryName()
	binaryNameProvided := false
	purgeConfig := false
	for i := 0; i < len(filtered); i++ {
		switch filtered[i] {
		case "--install-dir":
			if i+1 >= len(filtered) {
				return errors.New("--install-dir requires a path")
			}
			installDir = filtered[i+1]
			installDirProvided = true
			i++
		case "--binary-name":
			if i+1 >= len(filtered) {
				return errors.New("--binary-name requires a name")
			}
			binaryName = strings.TrimSpace(filtered[i+1])
			binaryNameProvided = true
			i++
		case "--purge-config":
			purgeConfig = true
		default:
			return fmt.Errorf("unexpected uninstall argument %q", filtered[i])
		}
	}
	if binaryName == "" {
		return errors.New("--binary-name cannot be empty")
	}
	if !installDirProvided || !binaryNameProvided {
		cfg, err := readConfig()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			if !installDirProvided {
				switch {
				case strings.TrimSpace(cfg.InstallDir) != "":
					installDir = strings.TrimSpace(cfg.InstallDir)
				case strings.TrimSpace(cfg.BinaryPath) != "":
					installDir = filepath.Dir(strings.TrimSpace(cfg.BinaryPath))
				}
			}
			if !binaryNameProvided {
				switch {
				case strings.TrimSpace(cfg.BinaryName) != "":
					binaryName = strings.TrimSpace(cfg.BinaryName)
				case strings.TrimSpace(cfg.BinaryPath) != "":
					binaryName = filepath.Base(strings.TrimSpace(cfg.BinaryPath))
				}
			}
		}
	}
	if installDir == "" {
		installDir = defaultInstallDir()
	}
	absInstallDir, err := filepath.Abs(installDir)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(absInstallDir, binaryName)
	removed := false
	if _, err := os.Lstat(targetPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else {
		progress("uninstall.remove", "removing installed binary", map[string]string{"target": targetPath})
		if err := os.Remove(targetPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		} else {
			removed = true
		}
	}
	result := installCommandResult{
		Action:     "uninstall.removed",
		BinaryName: binaryName,
		InstallDir: absInstallDir,
		TargetPath: targetPath,
		Removed:    removed,
	}
	if !purgeConfig {
		cfgPath, err := forgetInstalledBinary(targetPath)
		if err != nil {
			return err
		}
		result.ConfigPath = cfgPath
	}
	if purgeConfig {
		cfgPath, err := configPath()
		if err != nil {
			return err
		}
		result.ConfigPath = cfgPath
		if err := os.Remove(cfgPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		} else {
			result.ConfigPurged = true
		}
	}
	if agentUse {
		return agentUpdate("uninstall.ready", "ok", "uninstall complete", result)
	}
	if jsonOut {
		return writeJSON(result)
	}
	if removed {
		ok("uninstall.ready", "removed installed binary", nil)
	} else {
		warn("uninstall.ready", "installed binary was not found", nil)
	}
	printField("Binary", binaryName)
	printField("Target", targetPath)
	if purgeConfig {
		if result.ConfigPurged {
			printField("Config", "removed "+result.ConfigPath)
		} else {
			printField("Config", "not found "+result.ConfigPath)
		}
	}
	return nil
}

type setupCommandResult struct {
	Action               string `json:"action"`
	StoragePath          string `json:"storage_path"`
	ConfigPath           string `json:"config_path"`
	DefaultActor         string `json:"default_actor"`
	AutoPullBeforeRead   bool   `json:"auto_pull_before_read"`
	AutoCommitAfterWrite bool   `json:"auto_commit_after_write"`
	AutoPushAfterCommit  bool   `json:"auto_push_after_commit"`
}

type setupDoctorResult struct {
	Action      string             `json:"action"`
	OK          bool               `json:"ok"`
	Fixed       bool               `json:"fixed"`
	Fix         bool               `json:"fix"`
	ConfigPath  string             `json:"config_path"`
	StoragePath string             `json:"storage_path,omitempty"`
	Checks      []setupDoctorCheck `json:"checks"`
}

type setupDoctorCheck struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	Path             string `json:"path,omitempty"`
	CanFix           bool   `json:"can_fix,omitempty"`
	Fixed            bool   `json:"fixed,omitempty"`
	RequiresApproval bool   `json:"requires_approval,omitempty"`
	Suggestion       string `json:"suggestion,omitempty"`
}

func setupCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	if len(filtered) > 0 && filtered[0] == "doctor" {
		return setupDoctorCommand(filtered[1:], jsonOut)
	}
	cfg := defaultConfig()
	storagePath := ""
	storageProvided := false
	for i := 0; i < len(filtered); i++ {
		switch filtered[i] {
		case "--storage":
			if i+1 >= len(filtered) {
				return errors.New("--storage requires an absolute path")
			}
			storagePath = filtered[i+1]
			storageProvided = true
			i++
		case "--actor":
			if i+1 >= len(filtered) {
				return errors.New("--actor requires a name")
			}
			cfg.DefaultActor = strings.TrimSpace(filtered[i+1])
			if cfg.DefaultActor == "" {
				return errors.New("--actor cannot be empty")
			}
			i++
		case "--auto-pull":
			cfg.AutoPullBeforeRead = true
		case "--no-auto-pull":
			cfg.AutoPullBeforeRead = false
		case "--auto-commit":
			cfg.AutoCommitAfterWrite = true
		case "--no-auto-commit":
			cfg.AutoCommitAfterWrite = false
		case "--auto-push":
			cfg.AutoPushAfterCommit = true
		case "--no-auto-push":
			cfg.AutoPushAfterCommit = false
		default:
			return fmt.Errorf("unexpected setup argument %q", filtered[i])
		}
	}
	if storagePath == "" {
		if storageProvided || jsonOut || agentUse || len(filtered) > 0 {
			return errors.New("usage: maat setup [--storage <absolute-git-repo-path>] [--actor <name>] [--auto-pull|--no-auto-pull] [--auto-commit|--no-auto-commit] [--auto-push|--no-auto-push] [--json]")
		}
		if err := promptSetup(&cfg, &storagePath); err != nil {
			return err
		}
	}
	abs, err := validateSetupStoragePath(storagePath)
	if err != nil {
		return err
	}
	cfg.StoragePath = abs
	preserveInstalledBinaryConfig(&cfg)
	configFile, err := persistConfig(cfg)
	if err != nil {
		return err
	}
	result := setupCommandResult{
		Action:               "setup.configured",
		StoragePath:          cfg.StoragePath,
		ConfigPath:           configFile,
		DefaultActor:         cfg.DefaultActor,
		AutoPullBeforeRead:   cfg.AutoPullBeforeRead,
		AutoCommitAfterWrite: cfg.AutoCommitAfterWrite,
		AutoPushAfterCommit:  cfg.AutoPushAfterCommit,
	}
	if agentUse {
		return agentUpdate("setup.configured", "ok", "setup complete", result)
	}
	if jsonOut {
		return writeJSON(result)
	}
	ok("setup.configured", "setup complete", nil)
	printField("Storage", result.StoragePath)
	printField("Config", result.ConfigPath)
	printField("Actor", result.DefaultActor)
	printField("Auto-pull before reads", formatBool(result.AutoPullBeforeRead))
	printField("Auto-commit after writes", formatBool(result.AutoCommitAfterWrite))
	printField("Auto-push after commits", formatBool(result.AutoPushAfterCommit))
	return nil
}

func setupDoctorCommand(args []string, jsonOut bool) error {
	storagePath := ""
	fix := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--storage":
			if i+1 >= len(args) {
				return errors.New("--storage requires a path")
			}
			storagePath = args[i+1]
			i++
		case "--fix":
			fix = true
		default:
			return fmt.Errorf("unexpected setup doctor argument %q", args[i])
		}
	}

	result, err := runSetupDoctor(storagePath, fix)
	if err != nil {
		return err
	}
	if agentUse {
		status := "ok"
		message := "setup healthy"
		if !result.OK {
			status = "warning"
			message = "setup needs attention"
		}
		return agentUpdate("setup.doctor", status, message, result)
	}
	if jsonOut {
		return writeJSON(result)
	}
	printSetupDoctorResult(result)
	return nil
}

func runSetupDoctor(storageArg string, fix bool) (setupDoctorResult, error) {
	configFile, err := configPath()
	if err != nil {
		return setupDoctorResult{}, err
	}
	result := setupDoctorResult{
		Action:     "setup.doctor",
		Fix:        fix,
		ConfigPath: configFile,
	}

	cfg, configErr := readConfig()
	configMissing := errors.Is(configErr, os.ErrNotExist)
	configReadable := configErr == nil
	if configReadable {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:      "config",
			Status:  "ok",
			Message: "config file is readable",
			Path:    configFile,
		})
	} else if configMissing {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "config",
			Status:     "warning",
			Message:    "config file is missing",
			Path:       configFile,
			CanFix:     strings.TrimSpace(storageArg) != "",
			Suggestion: "run maat setup --storage <absolute-git-repo-path>, or run maat setup doctor --storage <path> --fix",
		})
		cfg = defaultConfig()
	} else {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "config",
			Status:     "error",
			Message:    "config file could not be read: " + configErr.Error(),
			Path:       configFile,
			Suggestion: "inspect permissions or repair the JSON config manually",
		})
		cfg = defaultConfig()
	}

	storagePath := strings.TrimSpace(storageArg)
	storageProvided := storagePath != ""
	if storagePath == "" && configReadable {
		storagePath = strings.TrimSpace(cfg.StoragePath)
	}
	if storagePath == "" {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "storage_configured",
			Status:     "error",
			Message:    "no storage path is configured",
			CanFix:     false,
			Suggestion: "run maat setup --storage <absolute-git-repo-path>",
		})
		finalizeSetupDoctorResult(&result)
		return result, nil
	}

	abs, err := filepath.Abs(storagePath)
	if err != nil {
		return setupDoctorResult{}, err
	}
	result.StoragePath = abs

	storageOK, storageWritable := inspectSetupDoctorStorage(&result, abs)
	repoOK := false
	if storageOK {
		repoOK = inspectSetupDoctorGit(&result, abs)
		inspectSetupDoctorValidation(&result, abs)
		if storageWritable {
			inspectSetupDoctorIndexes(&result, abs, fix)
		}
	}

	if configMissing && fix && storageProvided && storageOK && repoOK {
		repaired := defaultConfig()
		repaired.StoragePath = abs
		preserveInstalledBinaryConfig(&repaired)
		if path, err := persistConfig(repaired); err != nil {
			result.Checks = append(result.Checks, setupDoctorCheck{
				ID:         "config_repair",
				Status:     "error",
				Message:    "config file could not be written: " + err.Error(),
				Path:       configFile,
				Suggestion: "check config directory permissions",
			})
		} else {
			result.ConfigPath = path
			markSetupDoctorCheckFixed(&result, "config", "config file was created")
		}
	}

	finalizeSetupDoctorResult(&result)
	return result, nil
}

func inspectSetupDoctorStorage(result *setupDoctorResult, storagePath string) (bool, bool) {
	info, err := os.Stat(storagePath)
	if err != nil {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "storage_path",
			Status:     "error",
			Message:    "storage path does not exist: " + err.Error(),
			Path:       storagePath,
			Suggestion: "create or clone a Git-backed Maat storage repo, then run maat setup --storage <path>",
		})
		return false, false
	}
	if !info.IsDir() {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "storage_path",
			Status:     "error",
			Message:    "storage path is not a directory",
			Path:       storagePath,
			Suggestion: "choose a directory that contains the Maat storage repo",
		})
		return false, false
	}
	result.Checks = append(result.Checks, setupDoctorCheck{
		ID:      "storage_path",
		Status:  "ok",
		Message: "storage path exists",
		Path:    storagePath,
	})
	if err := checkWritableDirectory(storagePath); err != nil {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "storage_writable",
			Status:     "error",
			Message:    "storage path is not writable: " + err.Error(),
			Path:       storagePath,
			Suggestion: "fix directory ownership or permissions before running write commands",
		})
		return true, false
	}
	result.Checks = append(result.Checks, setupDoctorCheck{
		ID:      "storage_writable",
		Status:  "ok",
		Message: "storage path is writable",
		Path:    storagePath,
	})
	return true, true
}

func inspectSetupDoctorGit(result *setupDoctorResult, storagePath string) bool {
	git := maat.GitSync{Store: storagePath}
	repo, err := git.Info(context.Background())
	if err != nil {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "git_repository",
			Status:     "error",
			Message:    "git repository could not be inspected: " + err.Error(),
			Path:       storagePath,
			Suggestion: "run git status in the storage repo and resolve the reported Git error",
		})
		return false
	}
	if !repo.IsRepository {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:               "git_repository",
			Status:           "warning",
			Message:          "storage path is not a Git repository",
			Path:             storagePath,
			RequiresApproval: true,
			Suggestion:       "run git -C " + storagePath + " init after confirming this is the intended storage repo",
		})
		return false
	}
	result.Checks = append(result.Checks, setupDoctorCheck{
		ID:      "git_repository",
		Status:  "ok",
		Message: "storage path is a Git repository",
		Path:    storagePath,
	})
	if strings.TrimSpace(repo.RemoteURL) == "" {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:               "git_upstream",
			Status:           "warning",
			Message:          "storage repo has no origin remote",
			Path:             storagePath,
			RequiresApproval: true,
			Suggestion:       "run git -C " + storagePath + " remote add origin <url>, then push with -u",
		})
		return true
	}
	upstream, err := gitBranchUpstream(storagePath)
	if err != nil {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:               "git_upstream",
			Status:           "warning",
			Message:          "current branch has no upstream tracking branch",
			Path:             storagePath,
			RequiresApproval: true,
			Suggestion:       "run git -C " + storagePath + " push -u origin " + repo.Branch + " after confirming the remote",
		})
		return true
	}
	result.Checks = append(result.Checks, setupDoctorCheck{
		ID:      "git_upstream",
		Status:  "ok",
		Message: "current branch tracks " + upstream,
		Path:    storagePath,
	})
	return true
}

func inspectSetupDoctorValidation(result *setupDoctorResult, storagePath string) {
	report, err := maat.ValidateStore(storagePath)
	if err != nil {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "validation",
			Status:     "error",
			Message:    "storage validation failed to run: " + err.Error(),
			Path:       storagePath,
			Suggestion: "inspect storage files and rerun maat validate",
		})
		return
	}
	if !report.OK() {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:         "validation",
			Status:     "error",
			Message:    fmt.Sprintf("storage has %d validation issues", len(report.Issues)),
			Path:       storagePath,
			Suggestion: "run maat validate --storage " + storagePath + " and fix the reported Markdown issues",
		})
		return
	}
	result.Checks = append(result.Checks, setupDoctorCheck{
		ID:      "validation",
		Status:  "ok",
		Message: fmt.Sprintf("storage validates with %d files", report.Files),
		Path:    storagePath,
	})
}

func inspectSetupDoctorIndexes(result *setupDoctorResult, storagePath string, fix bool) {
	stale, message := setupDoctorIndexesStale(storagePath)
	if !stale {
		result.Checks = append(result.Checks, setupDoctorCheck{
			ID:      "indexes",
			Status:  "ok",
			Message: message,
			Path:    filepath.Join(storagePath, ".maat"),
		})
		return
	}
	check := setupDoctorCheck{
		ID:         "indexes",
		Status:     "warning",
		Message:    message,
		Path:       filepath.Join(storagePath, ".maat"),
		CanFix:     true,
		Suggestion: "run maat index rebuild --storage " + storagePath + ", or run maat setup doctor --storage " + storagePath + " --fix",
	}
	if fix {
		idx, err := maat.BuildIndex(storagePath)
		if err != nil {
			check.Status = "error"
			check.Message = "index rebuild failed while reading storage: " + err.Error()
		} else if _, err := maat.WriteIndex(storagePath, idx); err != nil {
			check.Status = "error"
			check.Message = "json index rebuild failed: " + err.Error()
		} else if _, err := maat.RebuildSQLiteIndex(storagePath); err != nil {
			check.Status = "error"
			check.Message = "sqlite index rebuild failed: " + err.Error()
		} else {
			check.Status = "fixed"
			check.Fixed = true
			check.Message = "indexes were rebuilt"
			check.Suggestion = ""
		}
	}
	result.Checks = append(result.Checks, check)
}

func setupDoctorIndexesStale(storagePath string) (bool, string) {
	jsonIndex := filepath.Join(storagePath, ".maat", "index.json")
	sqliteIndex := filepath.Join(storagePath, ".maat", "index.sqlite")
	jsonInfo, jsonErr := os.Stat(jsonIndex)
	sqliteInfo, sqliteErr := os.Stat(sqliteIndex)
	if jsonErr != nil || sqliteErr != nil {
		return true, "one or more rebuildable indexes are missing"
	}
	latest, err := latestMarkdownModTime(storagePath)
	if err != nil {
		return true, "could not inspect Markdown files for index freshness: " + err.Error()
	}
	if latest.After(jsonInfo.ModTime()) || latest.After(sqliteInfo.ModTime()) {
		return true, "one or more rebuildable indexes are stale"
	}
	return false, "rebuildable indexes are present and current"
}

func latestMarkdownModTime(storagePath string) (time.Time, error) {
	var latest time.Time
	err := filepath.WalkDir(storagePath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".maat":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest, err
}

func checkWritableDirectory(dir string) error {
	file, err := os.CreateTemp(dir, ".maat-doctor-*")
	if err != nil {
		return err
	}
	name := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

func gitBranchUpstream(storagePath string) (string, error) {
	command := exec.Command("git", "-C", storagePath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func markSetupDoctorCheckFixed(result *setupDoctorResult, id, message string) {
	for i := range result.Checks {
		if result.Checks[i].ID == id {
			result.Checks[i].Status = "fixed"
			result.Checks[i].Message = message
			result.Checks[i].Fixed = true
			result.Checks[i].CanFix = false
			result.Checks[i].Suggestion = ""
			return
		}
	}
}

func finalizeSetupDoctorResult(result *setupDoctorResult) {
	result.OK = true
	for _, check := range result.Checks {
		if check.Fixed {
			result.Fixed = true
		}
		if check.Status != "ok" && check.Status != "fixed" {
			result.OK = false
		}
	}
}

func printSetupDoctorResult(result setupDoctorResult) {
	if result.OK {
		ok("setup.doctor", "setup healthy", nil)
	} else {
		warn("setup.doctor", "setup needs attention", nil)
	}
	printField("Config", result.ConfigPath)
	if result.StoragePath != "" {
		printField("Storage", result.StoragePath)
	}
	printField("Fix mode", formatBool(result.Fix))
	for _, check := range result.Checks {
		line := fmt.Sprintf("[%s] %s: %s", check.Status, check.ID, check.Message)
		fmt.Println(line)
		if check.RequiresApproval {
			fmt.Println("  requires approval")
		}
		if check.CanFix && !check.Fixed {
			fmt.Println("  can fix with --fix")
		}
		if check.Suggestion != "" {
			fmt.Println("  " + check.Suggestion)
		}
	}
}

func promptSetup(cfg *config, storagePath *string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Maat setup")
	path, err := promptString(reader, "Storage repo path", defaultSetupStoragePath())
	if err != nil {
		return err
	}
	actor, err := promptString(reader, "Default actor", cfg.DefaultActor)
	if err != nil {
		return err
	}
	if strings.TrimSpace(actor) == "" {
		return errors.New("default actor cannot be empty")
	}
	autoPull, err := promptBool(reader, "Auto-pull before reads", cfg.AutoPullBeforeRead)
	if err != nil {
		return err
	}
	autoCommit, err := promptBool(reader, "Auto-commit after writes", cfg.AutoCommitAfterWrite)
	if err != nil {
		return err
	}
	autoPush, err := promptBool(reader, "Auto-push after commits", cfg.AutoPushAfterCommit)
	if err != nil {
		return err
	}
	*storagePath = path
	cfg.DefaultActor = strings.TrimSpace(actor)
	cfg.AutoPullBeforeRead = autoPull
	cfg.AutoCommitAfterWrite = autoCommit
	cfg.AutoPushAfterCommit = autoPush
	return nil
}

func promptString(reader *bufio.Reader, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	value, err := readPromptLine(reader)
	if err != nil {
		return "", err
	}
	if value == "" {
		value = defaultValue
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return strings.TrimSpace(value), nil
}

func promptBool(reader *bufio.Reader, label string, defaultValue bool) (bool, error) {
	defaultPrompt := "y/N"
	if defaultValue {
		defaultPrompt = "Y/n"
	}
	for {
		fmt.Printf("%s [%s]: ", label, defaultPrompt)
		value, err := readPromptLine(reader)
		if err != nil {
			return false, err
		}
		if value == "" {
			return defaultValue, nil
		}
		switch strings.ToLower(value) {
		case "y", "yes", "true", "1":
			return true, nil
		case "n", "no", "false", "0":
			return false, nil
		default:
			fmt.Println("Please answer yes or no.")
		}
	}
}

func readPromptLine(reader *bufio.Reader) (string, error) {
	value, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && value != "" {
			return strings.TrimSpace(value), nil
		}
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func validateSetupStoragePath(storagePath string) (string, error) {
	if !filepath.IsAbs(storagePath) {
		return "", fmt.Errorf("setup storage path must be absolute: %s", storagePath)
	}
	abs, err := filepath.Abs(storagePath)
	if err != nil {
		return "", err
	}
	if stat, err := os.Stat(abs); err != nil {
		return "", err
	} else if !stat.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}
	if !isGitRepository(abs) {
		return "", fmt.Errorf("setup storage path must be a Git repository: %s", abs)
	}
	return abs, nil
}

func indexCommand(args []string) error {
	if len(args) == 0 || args[0] != "rebuild" {
		return errors.New("usage: maat index rebuild [--storage <path>]")
	}
	store, err := loadStore(args[1:])
	if err != nil {
		return err
	}
	progress("index.read", "reading markdown store", map[string]string{"storage": store})
	idx, err := maat.BuildIndex(store)
	if err != nil {
		return err
	}
	progress("index.json", "writing json index", nil)
	path, err := maat.WriteIndex(store, idx)
	if err != nil {
		return err
	}
	progress("index.sqlite", "building sqlite index", nil)
	sqliteInfo, err := maat.RebuildSQLiteIndex(store)
	if err != nil {
		return err
	}
	if agentUse {
		return agentUpdate("index.ready", "ok", "index rebuilt", map[string]any{
			"projects":     len(idx.Projects),
			"documents":    len(idx.Documents),
			"json_path":    path,
			"sqlite_index": sqliteInfo,
		})
	}
	ok("index.ready", fmt.Sprintf("indexed %d projects and %d documents", len(idx.Projects), len(idx.Documents)), nil)
	fmt.Printf("indexed %d projects and %d documents\n", len(idx.Projects), len(idx.Documents))
	fmt.Printf("json:   %s\n", path)
	fts := "disabled"
	if sqliteInfo.FTS {
		fts = "enabled"
	}
	fmt.Printf("sqlite: %s (%d documents, fts %s)\n", sqliteInfo.Path, sqliteInfo.Documents, fts)
	return nil
}

func syncCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	message, rest, err := consumeFlagValue(rest, "--message", false)
	if err != nil {
		return err
	}

	push := false
	statusOnly := false
	for _, arg := range rest {
		switch arg {
		case "--push":
			push = true
		case "--status":
			statusOnly = true
		default:
			return fmt.Errorf("unexpected sync argument %q", arg)
		}
	}
	if statusOnly && push {
		return errors.New("--push cannot be used with --status")
	}
	if !statusOnly && strings.TrimSpace(message) == "" {
		message = "status(maat): sync store"
	}

	progress("sync.start", "checking git storage", map[string]any{
		"storage":     store,
		"status_only": statusOnly,
		"push":        push,
	})
	result, err := maat.SyncStore(context.Background(), maat.StoreSyncOptions{
		Store:        store,
		Message:      message,
		Push:         push,
		SkipIndex:    statusOnly,
		SkipValidate: statusOnly,
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(result)
	}
	if agentUse {
		return agentUpdate("sync.ready", "ok", "sync complete", result)
	}
	printSyncResult(result, statusOnly)
	return nil
}

func printSyncResult(result maat.StoreSyncResult, statusOnly bool) {
	repo := "not a git repository"
	if result.Repository.IsRepository {
		repo = "git repository"
		if result.Repository.Branch != "" {
			repo += " on " + result.Repository.Branch
		}
	}
	fmt.Printf("Repository: %s\n", repo)
	if result.Validation.Files > 0 {
		fmt.Printf("Validation: %d files, ok\n", result.Validation.Files)
	}
	if result.JSONIndexPath != "" {
		fmt.Printf("Index:      %s\n", result.JSONIndexPath)
	}
	if result.SQLiteIndex.Path != "" {
		fts := "disabled"
		if result.SQLiteIndex.FTS {
			fts = "enabled"
		}
		fmt.Printf("SQLite:     %s (%d documents, fts %s)\n", result.SQLiteIndex.Path, result.SQLiteIndex.Documents, fts)
	}
	if statusOnly {
		printSyncDirty("Dirty", result.DirtyBeforeCommit)
		return
	}
	if result.Committed {
		fmt.Printf("Committed:  %s\n", result.CommitMessage)
	} else {
		fmt.Println("Committed:  no changes")
	}
	if result.Pushed {
		fmt.Println("Pushed:     yes")
	}
	printSyncDirty("Remaining", result.DirtyAfterSync)
}

func printSyncDirty(label string, entries []maat.GitStatusEntry) {
	if len(entries) == 0 {
		fmt.Printf("%s:      clean\n", label)
		return
	}
	fmt.Printf("%s:      %d changes\n", label, len(entries))
	for _, entry := range entries {
		fmt.Printf("  %c%c %s\n", entry.Index, entry.Work, entry.Path)
	}
}

func validateCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, err := loadStoreForRead(filtered)
	if err != nil {
		return err
	}
	progress("validate.start", "validating markdown store", map[string]string{"storage": store})
	report, err := maat.ValidateStore(store)
	if err != nil {
		return err
	}
	if agentUse {
		status := "ok"
		message := "validation passed"
		if !report.OK() {
			status = "error"
			message = "validation failed"
		}
		if err := agentUpdate("validate.ready", status, message, report); err != nil {
			return err
		}
		if !report.OK() {
			return fmt.Errorf("validation failed with %d issues", len(report.Issues))
		}
		return nil
	}
	if jsonOut {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else if report.OK() {
		ok("validate.ready", fmt.Sprintf("validated %d files: ok", report.Files), nil)
	} else {
		warn("validate.ready", fmt.Sprintf("validated %d files: %d issues", report.Files, len(report.Issues)), nil)
		for _, issue := range report.Issues {
			location := issue.Path
			if issue.Line > 0 {
				location = fmt.Sprintf("%s:%d", location, issue.Line)
			}
			fmt.Printf("%s [%s] %s\n", location, issue.Code, issue.Message)
		}
	}
	if !report.OK() {
		return fmt.Errorf("validation failed with %d issues", len(report.Issues))
	}
	return nil
}

func projectCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: maat project <show|link>")
	}
	switch args[0] {
	case "show":
		return projectShowCommand(args[1:])
	case "link":
		return projectLinkCommand(args[1:])
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func projectShowCommand(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: maat project show <project-id> [--storage <path>] [--json]")
	}
	filtered, jsonOut := splitJSONFlag(args)
	projectID := filtered[0]
	store, err := loadStoreForRead(filtered[1:])
	if err != nil {
		return err
	}
	project, err := maat.LoadObjectProject(store, projectID)
	if err != nil {
		return err
	}
	if agentUse {
		return agentUpdate("project.ready", "ok", "project loaded", project)
	}
	if jsonOut {
		return writeJSON(project)
	}
	printObjectProject(project)
	return nil
}

func printObjectProject(project maat.ObjectProject) {
	fmt.Printf("# %s\n\n", project.DisplayName)
	fmt.Printf("Key:     %s\n", project.Key)
	fmt.Printf("Status:  %s\n", project.Status)
	fmt.Printf("Updated: %s\n", project.Updated)
	if len(project.Tags) > 0 {
		fmt.Printf("Tags:    %s\n", strings.Join(project.Tags, " "))
	}
	if project.Identity["Primary Repo"] != "" {
		fmt.Printf("Repo:    %s\n", project.Identity["Primary Repo"])
	}
	if project.Identity["Remote"] != "" {
		fmt.Printf("Remote:  %s\n", project.Identity["Remote"])
	}
	if project.Summary != "" {
		fmt.Println()
		fmt.Println(project.Summary)
	}
	if len(project.Goals) > 0 {
		fmt.Println()
		for _, goal := range project.Goals {
			fmt.Printf("- %s [%s] %s\n", goal.ID, goal.Status, goal.Title)
		}
	}
	if len(project.Tickets) > 0 {
		fmt.Println()
		for _, ticket := range project.Tickets {
			goal := "standalone"
			if ticket.GoalID != "" {
				goal = ticket.GoalID
			}
			fmt.Printf("- %s [%s] %s (%s)\n", ticket.ID, ticket.Status, ticket.Title, goal)
		}
	}
}

func projectLinkCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	key, rest, err := consumeFlagValue(rest, "--key", false)
	if err != nil {
		return err
	}
	name, rest, err := consumeFlagValue(rest, "--name", false)
	if err != nil {
		return err
	}
	if len(rest) > 1 {
		return errors.New("usage: maat project link [source-path] [--storage <path>] [--key <project-key>] [--name <display-name>] [--json]")
	}
	sourcePath := "."
	if len(rest) == 1 {
		sourcePath = rest[0]
	}
	progress("project.link", "linking source repo", map[string]string{"source": sourcePath})
	linked, err := maat.LinkProject(context.Background(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  sourcePath,
		ProjectKey:  key,
		DisplayName: name,
	})
	if err != nil {
		return err
	}
	warnIndexRefresh(refreshIndexesBestEffort(store))
	if linked.Created {
		warnAutoSync(autoSyncWrite(store, linked.ProjectKey, "project.linked"))
	}
	if agentUse {
		return agentUpdate("project.linked", "ok", "project linked", linked)
	}
	if jsonOut {
		return writeJSON(linked)
	}
	message := "linked project " + linked.ProjectKey
	if linked.Created {
		ok("project.linked", message, nil)
	} else {
		message = "project " + linked.ProjectKey + " already linked"
		ok("project.linked", message, nil)
	}
	printField("Project", linked.ProjectKey)
	printField("Name", linked.DisplayName)
	printField("Source", linked.SourcePath)
	if linked.RemoteURL != "" {
		printField("Remote", linked.RemoteURL)
	}
	return nil
}

func goalCommand(args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return errors.New("usage: maat goal create <project-key> <title> --outcome <text> [--storage <path>] [--json]")
	}
	filtered, jsonOut := splitJSONFlag(args[1:])
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	outcome, rest, err := consumeFlagValue(rest, "--outcome", false)
	if err != nil {
		return err
	}
	projectKey, title, err := resolveCreateProjectAndTitle(store, rest, "goal")
	if err != nil {
		return err
	}
	if strings.TrimSpace(outcome) == "" {
		return errors.New("--outcome is required; describe what this goal should achieve")
	}
	writer := maat.NewWriteStore(store)
	goal, event, err := writer.CreateGoal(maat.CreateGoalInput{
		ProjectKey: projectKey,
		Title:      title,
		Outcome:    outcome,
		Actor:      defaultActor(),
	})
	if err != nil {
		return err
	}
	return finishWrite(store, goal.ProjectKey, writeCommandResult{
		Action:     "goal.created",
		ProjectKey: goal.ProjectKey,
		GoalID:     goal.ID,
		EventID:    event.ID,
	}, jsonOut)
}

func ticketCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: maat ticket <create|list|show|claim|comment|complete>")
	}
	switch args[0] {
	case "create":
		return ticketCreateCommand(args[1:])
	case "list":
		return ticketListCommand(args[1:])
	case "show":
		return ticketShowCommand(args[1:])
	case "claim":
		return ticketClaimCommand(args[1:])
	case "comment":
		return ticketCommentCommand(args[1:])
	case "complete":
		return ticketCompleteCommand(args[1:])
	default:
		return fmt.Errorf("unknown ticket command %q", args[0])
	}
}

type ticketView struct {
	ID          string   `json:"id"`
	ProjectKey  string   `json:"project_key"`
	GoalID      string   `json:"goal_id,omitempty"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Description string   `json:"description,omitempty"`
	Acceptance  []string `json:"acceptance,omitempty"`
	Created     string   `json:"created,omitempty"`
	Path        string   `json:"path,omitempty"`
}

func ticketListCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRestForRead(filtered)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("usage: maat ticket list [--project <project-key>] [--storage <path>] [--json]")
	}
	if projectKey == "" {
		if project, err := inferProjectFromWorkingDirectory(store); err == nil {
			projectKey = project.Key
		}
	}
	tickets, err := loadTicketViews(store, projectKey)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(tickets)
	}
	if agentUse {
		return agentUpdate("tickets.ready", "ok", "tickets loaded", map[string]any{
			"tickets": tickets,
			"count":   len(tickets),
		})
	}
	for _, ticket := range tickets {
		goal := "standalone"
		if ticket.GoalID != "" {
			goal = ticket.GoalID
		}
		fmt.Printf("%-30s %-10s %-9s %-12s %s\n", ticket.ID, ticket.ProjectKey, ticket.Status, goal, ticket.Title)
	}
	return nil
}

func ticketShowCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRestForRead(filtered)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("usage: maat ticket show <ticket-id> [--project <project-key>] [--storage <path>] [--json]")
	}
	ticketID := rest[0]
	projectKey, err = resolveReadableTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	tickets, err := loadTicketViews(store, projectKey)
	if err != nil {
		return err
	}
	for _, ticket := range tickets {
		if ticket.ID != ticketID {
			continue
		}
		if agentUse {
			return agentUpdate("ticket.ready", "ok", "ticket loaded", ticket)
		}
		if jsonOut {
			return writeJSON(ticket)
		}
		fmt.Printf("# %s\n\n", ticket.Title)
		fmt.Printf("ID:      %s\n", ticket.ID)
		fmt.Printf("Project: %s\n", ticket.ProjectKey)
		fmt.Printf("Status:  %s\n", ticket.Status)
		if ticket.GoalID != "" {
			fmt.Printf("Goal:    %s\n", ticket.GoalID)
		}
		if ticket.Created != "" {
			fmt.Printf("Created: %s\n", ticket.Created)
		}
		if ticket.Path != "" {
			fmt.Printf("Path:    %s\n", ticket.Path)
		}
		if ticket.Description != "" {
			fmt.Printf("\nDescription:\n%s\n", ticket.Description)
		}
		if len(ticket.Acceptance) > 0 {
			fmt.Print("\nAcceptance:\n")
			for _, item := range ticket.Acceptance {
				fmt.Printf("- %s\n", item)
			}
		}
		return nil
	}
	return fmt.Errorf("ticket %q not found in project %q", ticketID, projectKey)
}

func ticketCreateCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	goalID, rest, err := consumeFlagValue(rest, "--goal", false)
	if err != nil {
		return err
	}
	description, rest, err := consumeFlagValue(rest, "--description", false)
	if err != nil {
		return err
	}
	acceptance, rest, err := consumeFlagValues(rest, "--acceptance")
	if err != nil {
		return err
	}
	projectKey, title, err := resolveCreateProjectAndTitle(store, rest, "ticket")
	if err != nil {
		return err
	}
	if strings.TrimSpace(description) == "" {
		return errors.New("--description is required; describe the concrete work to do")
	}
	if len(acceptance) == 0 {
		return errors.New("--acceptance is required at least once; add a completion condition")
	}
	writer := maat.NewWriteStore(store)
	ticket, event, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  projectKey,
		Title:       title,
		GoalID:      goalID,
		Description: description,
		Acceptance:  acceptance,
		Actor:       defaultActor(),
	})
	if err != nil {
		return err
	}
	return finishWrite(store, ticket.ProjectKey, writeCommandResult{
		Action:     "ticket.created",
		ProjectKey: ticket.ProjectKey,
		GoalID:     ticket.GoalID,
		TicketID:   ticket.ID,
		EventID:    event.ID,
	}, jsonOut)
}

func ticketClaimCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	agent, rest, err := consumeFlagValue(rest, "--agent", false)
	if err != nil {
		return err
	}
	ttlText, rest, err := consumeFlagValue(rest, "--ttl", false)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("ticket id is required; usage: maat ticket claim <ticket-id> [--agent <agent>] [--ttl <duration>] [--project <project-key>] [--storage <path>] [--json]")
	}
	if agent == "" {
		agent = defaultActor()
	}
	ttl := 2 * time.Hour
	if ttlText != "" {
		ttl, err = time.ParseDuration(ttlText)
		if err != nil {
			return fmt.Errorf("invalid --ttl %q: %w", ttlText, err)
		}
	}
	ticketID := rest[0]
	projectKey, err = resolveTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	now := time.Now()
	writer := maat.NewWriteStore(store)
	event, err := writer.ClaimTicket(maat.ClaimTicketInput{
		ProjectKey: projectKey,
		TicketID:   ticketID,
		Actor:      agent,
		ExpiresAt:  now.Add(ttl),
		At:         now,
	})
	if err != nil {
		return err
	}
	expiresAt := now.Add(ttl).Format(time.RFC3339)
	return finishWrite(store, projectKey, writeCommandResult{
		Action:     "ticket.claimed",
		ProjectKey: projectKey,
		TicketID:   ticketID,
		EventID:    event.ID,
		Agent:      agent,
		ExpiresAt:  expiresAt,
	}, jsonOut)
}

func ticketCommentCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		return errors.New("ticket id and comment are required; usage: maat ticket comment <ticket-id> <comment> [--project <project-key>] [--storage <path>] [--json]")
	}
	ticketID := rest[0]
	projectKey, err = resolveTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	writer := maat.NewWriteStore(store)
	event, err := writer.CommentTicket(maat.TicketCommentInput{
		ProjectKey: projectKey,
		TicketID:   ticketID,
		Actor:      defaultActor(),
		Comment:    rest[1],
	})
	if err != nil {
		return err
	}
	return finishWrite(store, projectKey, writeCommandResult{
		Action:     "ticket.commented",
		ProjectKey: projectKey,
		TicketID:   ticketID,
		EventID:    event.ID,
	}, jsonOut)
}

func ticketCompleteCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	evidence, rest, err := consumeFlagValue(rest, "--evidence", true)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("ticket id is required; usage: maat ticket complete <ticket-id> --evidence <text> [--project <project-key>] [--storage <path>] [--json]")
	}
	ticketID := rest[0]
	projectKey, err = resolveTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	writer := maat.NewWriteStore(store)
	event, err := writer.CompleteTicket(maat.CompleteTicketInput{
		ProjectKey: projectKey,
		TicketID:   ticketID,
		Actor:      defaultActor(),
		Evidence:   []string{evidence},
	})
	if err != nil {
		return err
	}
	return finishWrite(store, projectKey, writeCommandResult{
		Action:     "ticket.completed",
		ProjectKey: projectKey,
		TicketID:   ticketID,
		EventID:    event.ID,
	}, jsonOut)
}

type initializeCommandResult struct {
	Document       string             `json:"document"`
	LinkedProject  maat.LinkedProject `json:"linked_project"`
	ProjectKey     string             `json:"project_key"`
	StoragePath    string             `json:"storage_path"`
	Version        version.Info       `json:"version"`
	IndexRefreshed bool               `json:"index_refreshed"`
	IndexWarning   string             `json:"index_warning,omitempty"`
}

func agentInitializeCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	projectKey := ""
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--project":
			if i+1 >= len(rest) {
				return errors.New("--project requires a project key")
			}
			projectKey = rest[i+1]
			i++
		default:
			return fmt.Errorf("unexpected initialize argument %q", rest[i])
		}
	}

	progress("initialize.project", "registering current repo", map[string]string{"storage": store})
	linked, err := maat.LinkProject(context.Background(), maat.LinkProjectInput{
		Store:      store,
		SourcePath: ".",
		ProjectKey: projectKey,
	})
	if err != nil {
		return err
	}
	progress("initialize.index", "refreshing indexes", nil)
	refreshResult := refreshIndexesBestEffort(store)
	warnIndexRefresh(refreshResult)
	if linked.Created {
		warnAutoSync(autoSyncWrite(store, linked.ProjectKey, "initialize.project"))
	}

	binaryVersion := version.Current()
	document := maat.AgentSetupDocument(maat.AgentSetupOptions{
		ProjectKey:    linked.ProjectKey,
		StoragePath:   store,
		BinaryVersion: binaryVersion.String(),
	})
	result := initializeCommandResult{
		Document:       document,
		LinkedProject:  linked,
		ProjectKey:     linked.ProjectKey,
		StoragePath:    store,
		Version:        binaryVersion,
		IndexRefreshed: refreshResult.Refreshed,
		IndexWarning:   refreshResult.Warning,
	}
	if agentUse {
		return agentUpdate("initialize.ready", "ok", "agent setup document ready", result)
	}
	if jsonOut {
		return writeJSON(result)
	}
	if linked.Created {
		ok("initialize.project", "registered project "+linked.ProjectKey, nil)
	} else {
		ok("initialize.project", "project "+linked.ProjectKey+" already registered", nil)
	}
	fmt.Println(document)
	return nil
}

func searchCommand(args []string) error {
	jsonOut := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			jsonUse = true
			continue
		}
		filtered = append(filtered, arg)
	}
	store, rest, err := loadStoreAndRestForRead(filtered)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return errors.New("usage: maat search <query> [--storage <path>] [--json]")
	}
	query := strings.Join(rest, " ")
	progress("search.index", "refreshing search index", map[string]string{"query": query})
	results, err := searchWithSQLite(store, query)
	if err != nil {
		return err
	}
	if results == nil {
		results = []maat.SearchResult{}
	}
	if agentUse {
		return agentUpdate("search.ready", "ok", "search complete", map[string]any{
			"query":   query,
			"count":   len(results),
			"results": results,
		})
	}
	if jsonOut {
		return writeJSON(results)
	}
	ok("search.ready", fmt.Sprintf("found %d results", len(results)), nil)
	for _, result := range results {
		fmt.Printf("%s:%d [%s] %s\n", result.Path, result.Line, result.Type, result.Title)
		if result.Excerpt != "" {
			fmt.Printf("  %s\n", result.Excerpt)
		}
	}
	return nil
}

func searchWithSQLite(store, query string) ([]maat.SearchResult, error) {
	info, err := maat.RebuildSQLiteIndex(store)
	if err == nil {
		results, err := maat.SearchSQLiteIndex(info.Path, query)
		if err == nil {
			return results, nil
		}
	}
	return maat.Search(store, query)
}

type projectListItem struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Updated string `json:"updated,omitempty"`
	Path    string `json:"path"`
	Layout  string `json:"layout"`
}

func loadProjectListItems(store string) ([]projectListItem, error) {
	objectProjects, err := maat.LoadObjectProjects(store)
	if err != nil {
		return nil, err
	}
	projects := make([]projectListItem, 0, len(objectProjects))
	for _, project := range objectProjects {
		projects = append(projects, projectListItem{
			ID:      project.Key,
			Key:     project.Key,
			Title:   project.DisplayName,
			Status:  project.Status,
			Updated: project.Updated,
			Path:    project.Path,
			Layout:  "object",
		})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ID < projects[j].ID
	})
	return projects, nil
}

type writeCommandResult struct {
	Action          string           `json:"action"`
	ProjectKey      string           `json:"project_key"`
	GoalID          string           `json:"goal_id,omitempty"`
	TicketID        string           `json:"ticket_id,omitempty"`
	EventID         string           `json:"event_id"`
	Agent           string           `json:"agent,omitempty"`
	ExpiresAt       string           `json:"expires_at,omitempty"`
	IndexRefreshed  bool             `json:"index_refreshed"`
	IndexWarning    string           `json:"index_warning,omitempty"`
	AutoSync        *autoSyncSummary `json:"auto_sync,omitempty"`
	AutoSyncWarning string           `json:"auto_sync_warning,omitempty"`
}

func finishWrite(store, projectKey string, result writeCommandResult, jsonOut bool) error {
	progress("write.validate", "validating written project", map[string]string{"project": projectKey})
	if _, err := maat.LoadObjectProject(store, projectKey); err != nil {
		return fmt.Errorf("post-write validation failed for project %q: %w", projectKey, err)
	}
	progress("write.index", "refreshing indexes", nil)
	refreshResult := refreshIndexesBestEffort(store)
	result.IndexRefreshed = refreshResult.Refreshed
	result.IndexWarning = refreshResult.Warning
	warnIndexRefresh(refreshResult)
	syncResult := autoSyncWrite(store, projectKey, result.Action)
	result.AutoSync = syncResult.Result
	result.AutoSyncWarning = syncResult.Warning
	warnAutoSync(syncResult)
	if agentUse {
		return agentUpdate(result.Action, "ok", result.Action, result)
	}
	if jsonOut {
		return writeJSON(result)
	}
	printWriteResult(result)
	return nil
}

func printWriteResult(result writeCommandResult) {
	ok(result.Action, writeActionMessage(result), nil)
	switch result.Action {
	case "goal.created":
		printField("Goal", result.GoalID)
		printField("Project", result.ProjectKey)
	case "ticket.created":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
		if result.GoalID != "" {
			printField("Goal", result.GoalID)
		}
	case "ticket.claimed":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
		printField("Agent", result.Agent)
		printField("Expires", result.ExpiresAt)
	case "ticket.commented":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
	case "ticket.completed":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
	default:
		printField("Action", result.Action)
		printField("Project", result.ProjectKey)
	}
	printField("Event", result.EventID)
	if result.IndexWarning != "" {
		printField("Index", result.IndexWarning)
	}
}

func writeActionMessage(result writeCommandResult) string {
	switch result.Action {
	case "goal.created":
		return "created goal"
	case "ticket.created":
		return "created ticket"
	case "ticket.claimed":
		return "claimed ticket"
	case "ticket.commented":
		return "commented on ticket"
	case "ticket.completed":
		return "completed ticket"
	default:
		return result.Action
	}
}

type autoSyncSummary struct {
	Repository      maat.GitRepoInfo      `json:"repository"`
	Committed       bool                  `json:"committed"`
	Pushed          bool                  `json:"pushed"`
	CommitMessage   string                `json:"commit_message,omitempty"`
	CommitPathspecs []string              `json:"commit_pathspecs,omitempty"`
	DirtyAfterSync  []maat.GitStatusEntry `json:"dirty_after_sync,omitempty"`
}

type autoSyncResult struct {
	Result  *autoSyncSummary
	Warning string
}

func autoSyncWrite(store, projectKey, action string) autoSyncResult {
	cfg, err := readConfig()
	if err != nil || !cfg.AutoCommitAfterWrite {
		return autoSyncResult{}
	}
	git := maat.GitSync{Store: store}
	isRepository, err := git.IsRepository(context.Background())
	if err != nil {
		return autoSyncResult{Warning: fmt.Sprintf("auto-sync skipped after state write persisted: %v", err)}
	}
	if !isRepository {
		return autoSyncResult{}
	}
	message := autoCommitMessage(projectKey, action)
	progress("write.sync", "syncing storage repo", map[string]any{
		"project": projectKey,
		"push":    cfg.AutoPushAfterCommit,
	})
	result, err := maat.SyncStore(context.Background(), maat.StoreSyncOptions{
		Store:        store,
		Message:      message,
		Pathspecs:    autoSyncPathspecs(projectKey),
		Push:         cfg.AutoPushAfterCommit,
		SkipIndex:    true,
		SkipValidate: false,
	})
	summary := autoSyncSummary{
		Repository:      result.Repository,
		Committed:       result.Committed,
		Pushed:          result.Pushed,
		CommitMessage:   result.CommitMessage,
		CommitPathspecs: result.CommitPathspecs,
		DirtyAfterSync:  result.DirtyAfterSync,
	}
	if err != nil {
		return autoSyncResult{
			Result:  &summary,
			Warning: fmt.Sprintf("auto-sync failed after state write persisted: %v", err),
		}
	}
	return autoSyncResult{Result: &summary}
}

func autoCommitMessage(projectKey, action string) string {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		projectKey = "maat"
	}
	switch action {
	case "goal.created":
		return fmt.Sprintf("status(%s): create goal", projectKey)
	case "ticket.created":
		return fmt.Sprintf("status(%s): create ticket", projectKey)
	case "ticket.claimed":
		return fmt.Sprintf("status(%s): claim ticket", projectKey)
	case "ticket.commented":
		return fmt.Sprintf("status(%s): comment on ticket", projectKey)
	case "ticket.completed":
		return fmt.Sprintf("status(%s): complete ticket", projectKey)
	case "project.linked", "initialize.project":
		return fmt.Sprintf("status(%s): link project", projectKey)
	default:
		return fmt.Sprintf("status(%s): update maat", projectKey)
	}
}

func autoSyncPathspecs(projectKey string) []string {
	projectKey = strings.Trim(strings.TrimSpace(projectKey), "/")
	if projectKey != "" {
		return []string{path.Join("projects", projectKey)}
	}
	return []string{"projects"}
}

func warnAutoSync(result autoSyncResult) {
	if result.Warning == "" {
		return
	}
	warn("write.sync", result.Warning, map[string]any{
		"warning": result.Warning,
		"result":  result.Result,
	})
}

func refreshIndexes(store string) error {
	idx, err := maat.BuildIndex(store)
	if err != nil {
		return fmt.Errorf("rebuild json index: %w", err)
	}
	if _, err := maat.WriteIndex(store, idx); err != nil {
		return fmt.Errorf("write json index: %w", err)
	}
	if _, err := maat.RebuildSQLiteIndex(store); err != nil {
		return fmt.Errorf("rebuild sqlite index: %w", err)
	}
	return nil
}

type indexRefreshResult struct {
	Refreshed bool
	Warning   string
}

func refreshIndexesBestEffort(store string) indexRefreshResult {
	if err := refreshIndexes(store); err != nil {
		return indexRefreshResult{
			Refreshed: false,
			Warning:   fmt.Sprintf("index refresh failed after state write persisted: %v", err),
		}
	}
	return indexRefreshResult{Refreshed: true}
}

func warnIndexRefresh(result indexRefreshResult) {
	if result.Warning == "" {
		return
	}
	warn("index.refresh", result.Warning, map[string]any{
		"index_refreshed": result.Refreshed,
		"warning":         result.Warning,
	})
}

func resolveTicketProject(store, projectKey, ticketID string) (string, error) {
	if strings.TrimSpace(projectKey) != "" {
		return projectKey, nil
	}
	if project, err := inferProjectFromWorkingDirectory(store); err == nil {
		if objectProjectHasTicket(project, ticketID) {
			return project.Key, nil
		}
	}
	objectStore, err := maat.LoadObjectStore(store)
	if err != nil {
		return "", err
	}
	matches := make([]string, 0, 1)
	for _, project := range objectStore.Projects {
		for _, ticket := range project.Tickets {
			if ticket.ID == ticketID {
				matches = append(matches, project.Key)
				break
			}
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("ticket %q not found", ticketID)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ticket %q exists in multiple projects; pass --project <project-key>", ticketID)
	}
}

func resolveReadableTicketProject(store, projectKey, ticketID string) (string, error) {
	if strings.TrimSpace(projectKey) != "" {
		return projectKey, nil
	}
	if project, err := inferProjectFromWorkingDirectory(store); err == nil {
		if objectProjectHasTicket(project, ticketID) {
			return project.Key, nil
		}
	}
	tickets, err := loadTicketViews(store, "")
	if err != nil {
		return "", err
	}
	matches := make([]string, 0, 1)
	for _, ticket := range tickets {
		if ticket.ID == ticketID {
			matches = append(matches, ticket.ProjectKey)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("ticket %q not found", ticketID)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ticket %q exists in multiple projects; pass --project <project-key>", ticketID)
	}
}

func objectProjectHasTicket(project maat.ObjectProject, ticketID string) bool {
	for _, ticket := range project.Tickets {
		if ticket.ID == ticketID {
			return true
		}
	}
	return false
}

func loadTicketViews(store, projectKey string) ([]ticketView, error) {
	var tickets []ticketView
	if projectKey != "" {
		if project, err := maat.LoadObjectProject(store, projectKey); err == nil {
			tickets = append(tickets, objectTicketViews(project)...)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		return tickets, nil
	}
	objectStore, err := maat.LoadObjectStore(store)
	if err != nil {
		return nil, err
	}
	for _, project := range objectStore.Projects {
		tickets = append(tickets, objectTicketViews(project)...)
	}
	return tickets, nil
}

func objectTicketViews(project maat.ObjectProject) []ticketView {
	tickets := make([]ticketView, 0, len(project.Tickets))
	for _, ticket := range project.Tickets {
		tickets = append(tickets, ticketView{
			ID:          ticket.ID,
			ProjectKey:  ticket.ProjectKey,
			GoalID:      ticket.GoalID,
			Title:       ticket.Title,
			Status:      ticket.Status,
			Description: ticket.Description,
			Acceptance:  ticket.Acceptance,
			Created:     ticket.Created,
			Path:        ticket.Path,
		})
	}
	return tickets
}

func resolveCreateProjectAndTitle(store string, args []string, kind string) (string, string, error) {
	switch len(args) {
	case 1:
		project, err := inferProjectFromWorkingDirectory(store)
		if err != nil {
			return "", "", fmt.Errorf("project key is required when not inside a linked project; usage: maat %s create <project-key> <title>: %w", kind, err)
		}
		return project.Key, args[0], nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", fmt.Errorf("project key and %s title are required; usage: maat %s create [project-key] <title> [--storage <path>] [--json]", kind, kind)
	}
}

func inferProjectFromWorkingDirectory(store string) (maat.ObjectProject, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return maat.ObjectProject{}, err
	}
	return maat.InferProjectForPath(context.Background(), store, cwd)
}

func consumeFlagValue(args []string, flag string, required bool) (string, []string, error) {
	filtered := make([]string, 0, len(args))
	value := ""
	found := false
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			filtered = append(filtered, args[i])
			continue
		}
		if found {
			return "", nil, fmt.Errorf("%s can only be provided once", flag)
		}
		if i+1 >= len(args) {
			return "", nil, fmt.Errorf("%s requires a value", flag)
		}
		value = args[i+1]
		found = true
		i++
	}
	if required && !found {
		return "", nil, fmt.Errorf("%s is required", flag)
	}
	return value, filtered, nil
}

func consumeFlagValues(args []string, flag string) ([]string, []string, error) {
	filtered := make([]string, 0, len(args))
	values := make([]string, 0)
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			filtered = append(filtered, args[i])
			continue
		}
		if i+1 >= len(args) {
			return nil, nil, fmt.Errorf("%s requires a value", flag)
		}
		values = append(values, args[i+1])
		i++
	}
	return values, filtered, nil
}

func defaultActor() string {
	if value := strings.TrimSpace(os.Getenv("MAAT_ACTOR")); value != "" {
		return value
	}
	if cfg, err := readConfig(); err == nil && strings.TrimSpace(cfg.DefaultActor) != "" {
		return strings.TrimSpace(cfg.DefaultActor)
	}
	return defaultSystemActor()
}

func defaultSystemActor() string {
	for _, key := range []string{"USER", "USERNAME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return "agent"
}

func splitJSONFlag(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			jsonUse = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, jsonOut
}

func writeJSON(value any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func splitGlobalFlags(args []string) ([]string, error) {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--agent-use" {
			agentUse = true
			continue
		}
		filtered = append(filtered, arg)
	}
	if agentUse {
		for _, arg := range filtered {
			if arg == "--json" {
				return nil, errors.New("--agent-use cannot be combined with --json")
			}
		}
	}
	return filtered, nil
}

func progress(step, message string, data any) {
	if agentUse {
		_ = agentUpdate(step, "running", message, data)
		return
	}
	if jsonUse {
		return
	}
	fmt.Printf("%s %s\n", color("[run]", "36"), message)
}

func ok(step, message string, data any) {
	if agentUse {
		_ = agentUpdate(step, "ok", message, data)
		return
	}
	if jsonUse {
		return
	}
	fmt.Printf("%s %s\n", color("[ok]", "32"), message)
}

func warn(step, message string, data any) {
	if agentUse {
		_ = agentUpdate(step, "warning", message, data)
		return
	}
	if jsonUse {
		return
	}
	fmt.Printf("%s %s\n", color("[warn]", "33"), message)
}

func printField(label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Printf("%-10s %s\n", label+":", value)
}

func defaultBinaryName() string {
	if name := strings.TrimSpace(os.Getenv("MAAT_BINARY_NAME")); name != "" {
		return name
	}
	return "maat"
}

func defaultInstallDir() string {
	if dir := strings.TrimSpace(os.Getenv("MAAT_INSTALL_DIR")); dir != "" {
		return dir
	}
	if info, err := os.Stat("/usr/local/bin"); err == nil && info.IsDir() && isWritableDir("/usr/local/bin") {
		return "/usr/local/bin"
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".local", "bin")
	}
	return "."
}

func defaultUpdateInstallDir(binaryName string) string {
	if executable, err := os.Executable(); err == nil && filepath.Base(executable) == binaryName {
		dir := filepath.Dir(executable)
		if isWritableDir(dir) {
			return dir
		}
	}
	return defaultInstallDir()
}

func isWritableDir(path string) bool {
	file, err := os.CreateTemp(path, ".maat-write-test-*")
	if err != nil {
		return false
	}
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
	return true
}

func installBinary(sourcePath, targetPath string, sourcePerm os.FileMode) error {
	input, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := io.Copy(tmp, input); err != nil {
		_ = tmp.Close()
		return err
	}
	mode := sourcePerm
	if mode == 0 {
		mode = 0o755
	}
	mode |= 0o111
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func fetchLatestRelease() (githubRelease, error) {
	data, err := downloadURLWithAccept(latestReleaseURL, "application/json")
	if err != nil {
		return githubRelease{}, err
	}
	var release githubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return githubRelease{}, err
	}
	if release.TagName == "" && release.Name == "" {
		return githubRelease{}, errors.New("latest release response did not include a version")
	}
	return release, nil
}

func downloadURL(url string) ([]byte, error) {
	return downloadURLWithAccept(url, "application/octet-stream")
}

func downloadURLWithAccept(url, accept string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "maat-updater/"+version.Current().Version)
	response, err := updateHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		text := strings.TrimSpace(string(body))
		if text != "" {
			return nil, fmt.Errorf("download %s failed: %s: %s", url, response.Status, text)
		}
		return nil, fmt.Errorf("download %s failed: %s", url, response.Status)
	}
	return io.ReadAll(response.Body)
}

func releaseIsCurrent(current, latest string) bool {
	current = normalizeVersion(current)
	latest = normalizeVersion(latest)
	if current == "" || latest == "" || current == "dev" || current == "unknown" {
		return false
	}
	return current == latest
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "refs/tags/")
	value = strings.TrimPrefix(value, "v")
	return value
}

func selectReleaseAsset(release githubRelease, binaryName, goos, goarch string) (githubReleaseAsset, bool) {
	wantSuffix := goos + "-" + goarch + ".tar.gz"
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "checksum") {
			continue
		}
		if strings.HasPrefix(name, strings.ToLower(binaryName)+"-") && strings.HasSuffix(name, wantSuffix) && asset.BrowserDownloadURL != "" {
			return asset, true
		}
	}
	return githubReleaseAsset{}, false
}

func selectChecksumAsset(release githubRelease) (githubReleaseAsset, bool) {
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "checksum") && strings.HasSuffix(name, ".txt") && asset.BrowserDownloadURL != "" {
			return asset, true
		}
	}
	return githubReleaseAsset{}, false
}

func verifyArchiveChecksum(data []byte, assetName, checksums string) error {
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if filepath.Base(name) != assetName {
			continue
		}
		if !strings.EqualFold(fields[0], sum) {
			return fmt.Errorf("checksum mismatch for %s", assetName)
		}
		return nil
	}
	return fmt.Errorf("checksum for %s not found", assetName)
}

func materializeReleaseBinary(data []byte, assetName, binaryName string) (string, error) {
	if strings.HasSuffix(strings.ToLower(assetName), ".tar.gz") || strings.HasSuffix(strings.ToLower(assetName), ".tgz") {
		return extractBinaryFromTarGZ(data, binaryName)
	}
	tmp, err := os.CreateTemp("", binaryName+"-update-*")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func extractBinaryFromTarGZ(data []byte, binaryName string) (string, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Base(header.Name)
		if name != binaryName && !strings.HasPrefix(name, binaryName+"-") {
			continue
		}
		tmp, err := os.CreateTemp("", binaryName+"-update-*")
		if err != nil {
			return "", err
		}
		path := tmp.Name()
		if _, err := io.Copy(tmp, tarReader); err != nil {
			_ = tmp.Close()
			_ = os.Remove(path)
			return "", err
		}
		mode := os.FileMode(header.Mode).Perm()
		if mode == 0 {
			mode = 0o755
		}
		mode |= 0o111
		if err := tmp.Chmod(mode); err != nil {
			_ = tmp.Close()
			_ = os.Remove(path)
			return "", err
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(path)
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("archive did not contain %s binary", binaryName)
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func dirOnPath(dir string) bool {
	if dir == "" {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = filepath.Clean(dir)
	}
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == "" {
			continue
		}
		absEntry, err := filepath.Abs(entry)
		if err != nil {
			absEntry = filepath.Clean(entry)
		}
		if filepath.Clean(absEntry) == filepath.Clean(absDir) {
			return true
		}
	}
	return false
}

func agentUpdate(step, status, message string, data any) error {
	event := map[string]any{
		"type":    "maat.update",
		"step":    step,
		"status":  status,
		"message": message,
	}
	if data != nil {
		event["data"] = data
	}
	return json.NewEncoder(os.Stdout).Encode(event)
}

func colorNumber(value int) string {
	return color(fmt.Sprintf("%d", value), "36")
}

func colorStatus(status string) string {
	switch strings.ToLower(status) {
	case "done":
		return color(status, "32")
	case "active":
		return color(status, "36")
	case "waiting", "paused":
		return color(status, "33")
	default:
		return status
	}
}

func color(text, code string) string {
	if !colorEnabled() {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func colorEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MAAT_COLOR"))) {
	case "always", "1", "true", "yes":
		return true
	case "never", "0", "false", "no":
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func loadStore(args []string) (string, error) {
	store, rest, err := loadStoreAndRest(args)
	if err != nil {
		return "", err
	}
	if len(rest) > 0 {
		return "", fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}
	return store, nil
}

func loadStoreForRead(args []string) (string, error) {
	store, rest, err := loadStoreAndRestForRead(args)
	if err != nil {
		return "", err
	}
	if len(rest) > 0 {
		return "", fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}
	return store, nil
}

func loadStoreAndRestForRead(args []string) (string, []string, error) {
	store, rest, err := loadStoreAndRest(args)
	if err != nil {
		return "", nil, err
	}
	autoPullBeforeRead(store)
	return store, rest, nil
}

func autoPullBeforeRead(store string) {
	cfg, err := readConfig()
	if err != nil || !cfg.AutoPullBeforeRead {
		return
	}
	git := maat.GitSync{Store: store}
	isRepository, err := git.IsRepository(context.Background())
	if err != nil {
		warn("read.pull", fmt.Sprintf("auto-pull skipped before read: %v", err), map[string]any{
			"storage": store,
			"warning": err.Error(),
		})
		return
	}
	if !isRepository {
		return
	}
	progress("read.pull", "pulling storage repo", map[string]string{"storage": store})
	if err := git.PullRebase(context.Background()); err != nil {
		warn("read.pull", fmt.Sprintf("auto-pull failed before read: %v", err), map[string]any{
			"storage": store,
			"warning": err.Error(),
		})
	}
}

func loadStoreAndRest(args []string) (string, []string, error) {
	rest := make([]string, 0, len(args))
	store := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--storage" {
			if i+1 >= len(args) {
				return "", nil, errors.New("--storage requires a path")
			}
			store = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	if store == "" {
		cfg, err := readConfig()
		if err == nil && cfg.StoragePath != "" {
			store = cfg.StoragePath
		}
	}
	if store == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", nil, err
		}
		store = cwd
	}
	abs, err := filepath.Abs(store)
	if err != nil {
		return "", nil, err
	}
	return abs, rest, nil
}

type config struct {
	StoragePath          string `json:"storage_path"`
	DefaultActor         string `json:"default_actor"`
	AutoPullBeforeRead   bool   `json:"auto_pull_before_read"`
	AutoCommitAfterWrite bool   `json:"auto_commit_after_write"`
	AutoPushAfterCommit  bool   `json:"auto_push_after_commit"`
	InstallDir           string `json:"install_dir,omitempty"`
	BinaryName           string `json:"binary_name,omitempty"`
	BinaryPath           string `json:"binary_path,omitempty"`
}

func defaultConfig() config {
	return config{
		DefaultActor:         defaultSetupActor(),
		AutoPullBeforeRead:   true,
		AutoCommitAfterWrite: true,
		AutoPushAfterCommit:  true,
	}
}

func defaultSetupStoragePath() string {
	if cfg, err := readConfig(); err == nil && strings.TrimSpace(cfg.StoragePath) != "" {
		return strings.TrimSpace(cfg.StoragePath)
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

func defaultSetupActor() string {
	if value := strings.TrimSpace(os.Getenv("MAAT_ACTOR")); value != "" {
		return value
	}
	return defaultSystemActor()
}

func rememberInstalledBinary(binaryName, installDir, binaryPath string) (string, error) {
	cfg, err := readConfig()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		cfg = defaultConfig()
	}
	cfg.BinaryName = binaryName
	cfg.InstallDir = installDir
	cfg.BinaryPath = binaryPath
	return persistConfig(cfg)
}

func preserveInstalledBinaryConfig(cfg *config) {
	existing, err := readConfig()
	if err != nil {
		return
	}
	cfg.BinaryName = existing.BinaryName
	cfg.InstallDir = existing.InstallDir
	cfg.BinaryPath = existing.BinaryPath
}

func forgetInstalledBinary(targetPath string) (string, error) {
	cfg, err := readConfig()
	if err != nil {
		return "", nil
	}
	if !configMatchesBinaryPath(cfg, targetPath) {
		return "", nil
	}
	cfg.BinaryName = ""
	cfg.InstallDir = ""
	cfg.BinaryPath = ""
	path, err := persistConfig(cfg)
	if err != nil {
		return "", nil
	}
	return path, nil
}

func configMatchesBinaryPath(cfg config, targetPath string) bool {
	if strings.TrimSpace(cfg.BinaryPath) != "" && samePath(cfg.BinaryPath, targetPath) {
		return true
	}
	if strings.TrimSpace(cfg.InstallDir) == "" || strings.TrimSpace(cfg.BinaryName) == "" {
		return false
	}
	return samePath(filepath.Join(strings.TrimSpace(cfg.InstallDir), strings.TrimSpace(cfg.BinaryName)), targetPath)
}

func persistConfig(cfg config) (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func readConfig() (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func isGitRepository(path string) bool {
	command := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	output, err := command.Output()
	return err == nil && strings.TrimSpace(string(output)) == "true"
}

func formatBool(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func configPath() (string, error) {
	if override := os.Getenv("MAAT_CONFIG"); override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "maat", "config.json"), nil
}
