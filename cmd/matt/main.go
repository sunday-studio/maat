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
		store, err := loadStore(args[1:])
		if err != nil {
			return err
		}
		projects, err := maat.LoadProjects(store)
		if err != nil {
			return err
		}
		for _, project := range projects {
			fmt.Printf("%-12s %-9s %s\n", project.ID, project.Status, project.Title)
		}
		return nil
	case "project":
		return projectCommand(args[1:])
	case "status":
		store, err := loadStore(args[1:])
		if err != nil {
			return err
		}
		summary, err := maat.Status(store)
		if err != nil {
			return err
		}
		fmt.Printf("Projects: %d\n", summary.Projects)
		fmt.Printf("Goals:    %d active, %d done, %d total\n", summary.ActiveGoals, summary.DoneGoals, summary.Goals)
		fmt.Printf("Tickets:  %d open, %d done, %d total\n", summary.OpenTickets, summary.DoneTickets, summary.Tickets)
		return nil
	case "search":
		return searchCommand(args[1:])
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
  matt projects [--storage <path>]
  matt project show <project-id> [--storage <path>]
  matt status [--storage <path>]
  matt search <query> [--storage <path>] [--json]

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
	fmt.Printf("indexed %d projects and %d documents\n", len(idx.Projects), len(idx.Documents))
	fmt.Println(path)
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
	results, err := maat.Search(store, strings.Join(rest, " "))
	if err != nil {
		return err
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	for _, result := range results {
		fmt.Printf("%s:%d [%s] %s\n", result.Path, result.Line, result.Type, result.Title)
		if result.Excerpt != "" {
			fmt.Printf("  %s\n", result.Excerpt)
		}
	}
	return nil
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
