package api

import "time"

type TimeEntryItem struct {
	Billable      bool      `json:"billable"`
	ClientID      int       `json:"client_id"`
	ClientName    string    `json:"client_name"`
	CreatedAt     string    `json:"created_at"`
	CreatedWith   string    `json:"created_with"`
	DeletedAt     string    `json:"deleted_at"`
	Description   string    `json:"description"`
	Duration      int       `json:"duration"`
	ID            int       `json:"id"`
	ProjectID     int       `json:"project_id"`
	ProjectName   string    `json:"project_name"`
	Start         time.Time `json:"start"`
	Tags          []string  `json:"tags"`
	TaskID        int       `json:"task_id"`
	TaskName      string    `json:"task_name"`
	WorkspaceID   int       `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
}

type TimeEntry struct {
	CreatedWith string   `json:"created_with"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Billable    bool     `json:"billable"`
	WorkspaceID int      `json:"workspace_id"`
	Duration    int      `json:"duration"`
	Start       string   `json:"start"`
	Stop        *string  `json:"stop"`
	ProjectID   int      `json:"project_id"`
}

type Workspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
