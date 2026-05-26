package maat

type StatusSummary struct {
	Projects      int `json:"projects"`
	Goals         int `json:"goals"`
	ActiveGoals   int `json:"active_goals"`
	DoneGoals     int `json:"done_goals"`
	Tickets       int `json:"tickets"`
	OpenTickets   int `json:"open_tickets"`
	DoneTickets   int `json:"done_tickets"`
	BlockedItems  int `json:"blocked_items"`
	DecisionItems int `json:"decision_items"`
}

type Document struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Index struct {
	Version   int             `json:"version"`
	Projects  []ObjectProject `json:"projects"`
	Documents []Document      `json:"documents"`
}

type SearchResult struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
}
