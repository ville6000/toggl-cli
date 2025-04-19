package api

import "time"

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

type CurrentTimeEntry struct {
	ID          int       `json:"id"`
	WID         int       `json:"wid"`
	PID         int       `json:"pid"`
	Billable    bool      `json:"billable"`
	Start       time.Time `json:"start"`
	Duration    int       `json:"duration"`
	Description string    `json:"description"`
	At          string    `json:"at"`
}

type Workspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
