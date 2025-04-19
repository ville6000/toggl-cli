package api

type TimeEntry struct {
	CreatedWith string   `json:"created_with"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Billable    bool     `json:"billable"`
	WorkspaceID int      `json:"workspace_id"`
	Duration    int      `json:"duration"`
	Start       string   `json:"start"`
	Stop        *string  `json:"stop"`
}

type Workspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
