package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunday-studio/maat/internal/maat"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "matt: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return nil
	case "init":
		return initConfig(args[1:])
	case "storage":
		return storageCommand(args[1:])
	case "index":
		return indexCommand(args[1:])
	case "projects":
		filtered, jsonOut := splitJSONFlag(args[1:])
		store, err := loadStore(filtered)
		if err != nil {
			return err
		}
		projects, err := maat.LoadProjects(store)
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(projects)
		}
		for _, project := range projects {
			fmt.Printf("%-12s %-9s %s\n", project.ID, project.Status, project.Title)
		}
		return nil
	case "project":
		return projectCommand(args[1:])
	case "status":
		filtered, jsonOut := splitJSONFlag(args[1:])
		store, err := loadStore(filtered)
		if err != nil {
			return err
		}
		summary, err := maat.Status(store)
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(summary)
		}
		fmt.Printf("Projects: %d\n", summary.Projects)
		fmt.Printf("Goals:    %d active, %d done, %d total\n", summary.ActiveGoals, summary.DoneGoals, summary.Goals)
		fmt.Printf("Tickets:  %d open, %d done, %d total\n", summary.OpenTickets, summary.DoneTickets, summary.Tickets)
		return nil
	case "validate":
		return validateCommand(args[1:])
	case "search":
		return searchCommand(args[1:])
	case "tui":
		return tuiCommand(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Print(`matt is the Maat CLI.

Usage:
  matt init [storage-path]
  matt storage link <storage-path>
  matt index rebuild [--storage <path>]
  matt projects [--storage <path>] [--json]
  matt project show <project-id> [--storage <path>]
  matt status [--storage <path>] [--json]
  matt validate [--storage <path>] [--json]
  matt search <query> [--storage <path>] [--json]
  matt tui [--storage <path>]

Git plus Markdown remains the source of truth. The local index is rebuildable.
`)
}

func initConfig(args []string) error {
	storagePath := ""
	if len(args) > 0 {
		storagePath = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		storagePath = cwd
	}
	return writeConfig(storagePath)
}

func storageCommand(args []string) error {
	if len(args) != 2 || args[0] != "link" {
		return errors.New("usage: matt storage link <storage-path>")
	}
	return writeConfig(args[1])
}

func indexCommand(args []string) error {
	if len(args) == 0 || args[0] != "rebuild" {
		return errors.New("usage: matt index rebuild [--storage <path>]")
	}
	store, err := loadStore(args[1:])
	if err != nil {
		return err
	}
	idx, err := maat.BuildIndex(store)
	if err != nil {
		return err
	}
	path, err := maat.WriteIndex(store, idx)
	if err != nil {
		return err
	}
	sqliteInfo, err := maat.RebuildSQLiteIndex(store)
	if err != nil {
		return err
	}
	fmt.Printf("indexed %d projects and %d documents\n", len(idx.Projects), len(idx.Documents))
	fmt.Printf("json:   %s\n", path)
	fts := "disabled"
	if sqliteInfo.FTS {
		fts = "enabled"
	}
	fmt.Printf("sqlite: %s (%d documents, fts %s)\n", sqliteInfo.Path, sqliteInfo.Documents, fts)
	return nil
}

func validateCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, err := loadStore(filtered)
	if err != nil {
		return err
	}
	report, err := maat.ValidateStore(store)
	if err != nil {
		return err
	}
	if jsonOut {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else if report.OK() {
		fmt.Printf("validated %d files: ok\n", report.Files)
	} else {
		fmt.Printf("validated %d files: %d issues\n", report.Files, len(report.Issues))
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
	if len(args) < 2 || args[0] != "show" {
		return errors.New("usage: matt project show <project-id> [--storage <path>]")
	}
	projectID := args[1]
	store, err := loadStore(args[2:])
	if err != nil {
		return err
	}
	project, err := maat.LoadProject(store, projectID)
	if err != nil {
		return err
	}
	fmt.Printf("# %s\n\n", project.Title)
	fmt.Printf("ID:      %s\n", project.ID)
	fmt.Printf("Status:  %s\n", project.Status)
	fmt.Printf("Updated: %s\n", project.Updated)
	fmt.Printf("Tags:    %s\n\n", strings.Join(project.Tags, " "))
	if project.Current != "" {
		fmt.Println(project.Current)
		fmt.Println()
	}
	for _, goal := range project.Goals {
		fmt.Printf("- %s [%s] %s\n", goal.ID, goal.Status, goal.Title)
		for _, ticket := range goal.Tickets {
			box := " "
			if ticket.Done {
				box = "x"
			}
			fmt.Printf("  - [%s] %s %s\n", box, ticket.ID, ticket.Title)
		}
	}
	return nil
}

func searchCommand(args []string) error {
	jsonOut := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			continue
		}
		filtered = append(filtered, arg)
	}
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return errors.New("usage: matt search <query> [--storage <path>] [--json]")
	}
	query := strings.Join(rest, " ")
	results, err := searchWithSQLite(store, query)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(results)
	}
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

func splitJSONFlag(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
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
	StoragePath string `json:"storage_path"`
}

func writeConfig(storagePath string) error {
	abs, err := filepath.Abs(storagePath)
	if err != nil {
		return err
	}
	if stat, err := os.Stat(abs); err != nil {
		return err
	} else if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", abs)
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config{StoragePath: abs}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("linked storage: %s\n", abs)
	return nil
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
