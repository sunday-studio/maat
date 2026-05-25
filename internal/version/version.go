package version

import "fmt"

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
	return fmt.Sprintf("matt %s (%s, %s)", info.Version, info.Commit, info.Date)
}
