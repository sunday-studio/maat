package maat

type Project struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Owner     string   `json:"owner,omitempty"`
	Updated   string   `json:"updated,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Current   string   `json:"current,omitempty"`
	Path      string   `json:"path"`
	Goals     []Goal   `json:"goals,omitempty"`
	Blockers  []string `json:"blockers,omitempty"`
	Decisions []string `json:"decisions,omitempty"`
}

type Goal struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Status  string   `json:"status"`
	Updated string   `json:"updated,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Tickets []Ticket `json:"tickets,omitempty"`
}

type Ticket struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

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
	Version   int        `json:"version"`
	Projects  []Project  `json:"projects"`
	Documents []Document `json:"documents"`
}

type SearchResult struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
}
