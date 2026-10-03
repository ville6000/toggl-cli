package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Workspaces returns the workspaces the user belongs to.
func (c *Client) Workspaces(ctx context.Context) ([]Workspace, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/workspaces", nil)
	if err != nil {
		return nil, err
	}

	var workspaces []Workspace
	if reqErr := c.doRequest(req, &workspaces); reqErr != nil {
		return nil, reqErr
	}

	return workspaces, nil
}

// CurrentTimeEntry returns the running time entry. When nothing is running
// the returned entry has a zero ID.
func (c *Client) CurrentTimeEntry(ctx context.Context) (*TimeEntryItem, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/me/time_entries/current", nil)
	if err != nil {
		return nil, err
	}

	var entry TimeEntryItem
	if reqErr := c.doRequest(req, &entry); reqErr != nil {
		return nil, reqErr
	}

	return &entry, nil
}

// TimeEntry returns the user's time entry with the given ID.
func (c *Client) TimeEntry(ctx context.Context, id int) (*TimeEntryItem, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/me/time_entries/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var entry TimeEntryItem
	if reqErr := c.doRequest(req, &entry); reqErr != nil {
		return nil, reqErr
	}

	return &entry, nil
}

// CreateTimeEntry creates entry in the given workspace and returns it as
// stored by Toggl.
func (c *Client) CreateTimeEntry(ctx context.Context, workspaceID int, entry TimeEntry) (*TimeEntry, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries", workspaceID)
	req, err := c.newRequest(ctx, http.MethodPost, endpoint, entry)
	if err != nil {
		return nil, err
	}

	var createdEntry TimeEntry
	if reqErr := c.doRequest(req, &createdEntry); reqErr != nil {
		return nil, reqErr
	}

	return &createdEntry, nil
}

// NewTimeEntry returns a running time entry, started now, ready to be passed
// to CreateTimeEntry.
func NewTimeEntry(description string, workspaceID, projectID int, billable bool) TimeEntry {
	return TimeEntry{
		CreatedWith: "toggl-cli",
		Description: description,
		Tags:        []string{},
		Billable:    billable,
		WorkspaceID: workspaceID,
		Duration:    -1,
		Start:       time.Now().Format(time.RFC3339),
		Stop:        nil,
		ProjectID:   projectID,
	}
}

// StopTimeEntry stops the running entry entryID and returns the stopped entry.
func (c *Client) StopTimeEntry(ctx context.Context, workspaceID int, entryID int) (*TimeEntryItem, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries/%d/stop", workspaceID, entryID)
	req, err := c.newRequest(ctx, http.MethodPatch, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var stoppedEntry TimeEntryItem
	if reqErr := c.doRequest(req, &stoppedEntry); reqErr != nil {
		return nil, reqErr
	}

	return &stoppedEntry, nil
}

// UpdateTimeEntry replaces the fields of entry entryID with entry and returns
// the updated entry.
func (c *Client) UpdateTimeEntry(ctx context.Context, workspaceID int, entryID int, entry TimeEntry) (*TimeEntryItem, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries/%d", workspaceID, entryID)
	req, err := c.newRequest(ctx, http.MethodPut, endpoint, entry)
	if err != nil {
		return nil, err
	}

	var updatedEntry TimeEntryItem
	if reqErr := c.doRequest(req, &updatedEntry); reqErr != nil {
		return nil, reqErr
	}

	return &updatedEntry, nil
}

// Projects returns the projects in the workspace, from the cache when it holds
// a fresh copy, otherwise from the API (refreshing the cache).
func (c *Client) Projects(ctx context.Context, workspaceID int) ([]Project, error) {
	if c.Cache != nil {
		cachedProjects, cacheErr := c.Cache.Projects(workspaceID)
		if cacheErr == nil {
			return cachedProjects, nil
		}
	}

	endpoint := fmt.Sprintf("/workspaces/%d/projects", workspaceID)
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if reqErr := c.doRequest(req, &projects); reqErr != nil {
		return nil, reqErr
	}

	// The cache is best-effort: a failed write only means the next call
	// fetches the projects again.
	if c.Cache != nil {
		_ = c.Cache.SaveProjects(workspaceID, projects)
	}

	return projects, nil
}

// ProjectIDByName returns the ID of the project whose name matches
// projectName, ignoring case.
func (c *Client) ProjectIDByName(ctx context.Context, workspaceID int, projectName string) (int, error) {
	projects, err := c.Projects(ctx, workspaceID)
	if err != nil {
		return 0, err
	}

	for _, project := range projects {
		if strings.EqualFold(project.Name, projectName) {
			return project.ID, nil
		}
	}

	return 0, fmt.Errorf("project '%s' not found", projectName)
}

// TimeEntries returns the user's time entries between from and to. Either
// bound may be nil, leaving the range to the API's default.
func (c *Client) TimeEntries(ctx context.Context, from, to *time.Time) ([]TimeEntryItem, error) {
	endpoint := "/me/time_entries"
	queryParams := make([]string, 0)
	// RFC3339 instants rather than bare dates: a bare date is interpreted as
	// UTC by the API, which pulls in entries from the neighbouring local day.
	if from != nil {
		queryParams = append(queryParams, "start_date="+url.QueryEscape(from.Format(time.RFC3339)))
	}

	if to != nil {
		queryParams = append(queryParams, "end_date="+url.QueryEscape(to.Format(time.RFC3339)))
	}

	if len(queryParams) > 0 {
		endpoint += "?" + strings.Join(queryParams, "&")
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var timeEntries []TimeEntryItem
	if reqErr := c.doRequest(req, &timeEntries); reqErr != nil {
		return nil, reqErr
	}

	return timeEntries, nil
}

// ProjectNames returns the workspace's project names keyed by project ID.
func (c *Client) ProjectNames(ctx context.Context, workspaceID int) (map[int]string, error) {
	projects, err := c.Projects(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	lookup := make(map[int]string)
	for _, project := range projects {
		lookup[project.ID] = project.Name
	}

	return lookup, nil
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body any) (*http.Request, error) {
	req, err := newJSONRequest(ctx, method, c.BaseURL+endpoint, body)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	return req, nil
}

func (c *Client) doRequest(req *http.Request, result any) error {
	return doJSON(c.HTTPClient, req, result)
}

func (c *Client) setAuthHeader(req *http.Request) {
	token := base64.StdEncoding.EncodeToString([]byte(c.AuthToken + ":api_token"))

	req.Header.Set("Authorization", "Basic "+token)
}
