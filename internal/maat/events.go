package maat

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Event struct {
	ID       string
	Time     time.Time
	Actor    string
	Project  string
	Type     string
	Object   string
	Commit   string
	Metadata map[string]string
	Summary  string
	Evidence []string
}

func EventFilename(eventID string) (string, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return "", fmt.Errorf("event id is required")
	}
	if strings.ContainsAny(eventID, `/\`) {
		return "", fmt.Errorf("event id must not contain path separators")
	}
	return eventID + ".md", nil
}

func EventRelativePath(projectKey string, eventTime time.Time, eventID string) (string, error) {
	projectKey = NormalizeIDPart(projectKey)
	if projectKey == "" {
		return "", fmt.Errorf("project key is required")
	}
	if eventTime.IsZero() {
		return "", fmt.Errorf("event time is required")
	}
	filename, err := EventFilename(eventID)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(
		"projects",
		projectKey,
		"events",
		eventTime.Format("2006"),
		eventTime.Format("01"),
		filename,
	)), nil
}

func RenderEventMarkdown(event Event) (string, error) {
	if err := validateEvent(event); err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Event: %s\n\n", event.Type)
	buffer.WriteString("| Field | Value |\n")
	buffer.WriteString("|---|---|\n")
	writeMarkdownField(&buffer, "Event ID", event.ID)
	writeMarkdownField(&buffer, "Time", event.Time.Format(time.RFC3339))
	writeMarkdownField(&buffer, "Actor", event.Actor)
	writeMarkdownField(&buffer, "Project", event.Project)
	writeMarkdownField(&buffer, "Type", event.Type)
	writeMarkdownField(&buffer, "Object", event.Object)
	if strings.TrimSpace(event.Commit) != "" {
		writeMarkdownField(&buffer, "Commit", event.Commit)
	}
	for _, key := range sortedMetadataKeys(event.Metadata) {
		writeMarkdownField(&buffer, key, event.Metadata[key])
	}
	buffer.WriteString("\n## Summary\n\n")
	buffer.WriteString(strings.TrimSpace(event.Summary))
	buffer.WriteString("\n")
	if len(event.Evidence) > 0 {
		buffer.WriteString("\n## Evidence\n\n")
		for _, item := range event.Evidence {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			fmt.Fprintf(&buffer, "- %s\n", item)
		}
	}
	return buffer.String(), nil
}

func validateEvent(event Event) error {
	if strings.TrimSpace(event.ID) == "" {
		return fmt.Errorf("event id is required")
	}
	if event.Time.IsZero() {
		return fmt.Errorf("event time is required")
	}
	if strings.TrimSpace(event.Actor) == "" {
		return fmt.Errorf("event actor is required")
	}
	if strings.TrimSpace(event.Project) == "" {
		return fmt.Errorf("event project is required")
	}
	if strings.TrimSpace(event.Type) == "" {
		return fmt.Errorf("event type is required")
	}
	if strings.TrimSpace(event.Object) == "" {
		return fmt.Errorf("event object is required")
	}
	if strings.TrimSpace(event.Summary) == "" {
		return fmt.Errorf("event summary is required")
	}
	return nil
}

func sortedMetadataKeys(metadata map[string]string) []string {
	keys := make([]string, 0, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeMarkdownField(buffer *bytes.Buffer, key, value string) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
	value = strings.ReplaceAll(value, "|", `\|`)
	fmt.Fprintf(buffer, "| %s | %s |\n", key, value)
}
