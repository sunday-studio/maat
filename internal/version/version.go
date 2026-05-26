package version

import (
	"fmt"
	"strings"
)

var (
	// Version, Commit, and Date are set by release builds with -ldflags.
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func Current() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}

func (info Info) String() string {
	details := make([]string, 0, 2)
	if info.Commit != "" && info.Commit != "unknown" {
		details = append(details, info.Commit)
	}
	if info.Date != "" && info.Date != "unknown" {
		details = append(details, info.Date)
	}
	if len(details) == 0 {
		return fmt.Sprintf("maat %s", info.Version)
	}
	return fmt.Sprintf("maat %s (%s)", info.Version, strings.Join(details, ", "))
}
