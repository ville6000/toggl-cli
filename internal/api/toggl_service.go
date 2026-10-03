package api

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ville6000/toggl-cli/internal/data"
)

type ProjectService interface {
	GetProjects(workspaceID int) ([]data.Project, error)
}

func (c *Client) GetWorkspaces() ([]data.Workspace, error) {
	req, err := c.newRequest(http.MethodGet, "/workspaces", nil)
	if err != nil {
		return nil, err
	}

	var workspaces []data.Workspace
	if reqErr := c.doRequest(req, http.StatusOK, &workspaces); reqErr != nil {
		return nil, reqErr
	}

	return workspaces, nil
}

func (c *Client) GetCurrentTimerEntry() (*data.TimeEntryItem, error) {
	req, err := c.newRequest(http.MethodGet, "/me/time_entries/current", nil)
	if err != nil {
		return nil, err
	}

	var entry data.TimeEntryItem
	if reqErr := c.doRequest(req, http.StatusOK, &entry); reqErr != nil {
		return nil, reqErr
	}

	return &entry, nil
}

func (c *Client) CreateTimeEntry(workspaceID int, entry data.TimeEntry) (*data.TimeEntry, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries", workspaceID)
	req, err := c.newRequest(http.MethodPost, endpoint, entry)
	if err != nil {
		return nil, err
	}

	var createdEntry data.TimeEntry
	if reqErr := c.doRequest(req, http.StatusOK, &createdEntry); reqErr != nil {
		return nil, reqErr
	}

	return &createdEntry, nil
}

func (c *Client) NewTimeEntry(description string,
	workspaceID int,
	projectID int,
	billable bool,
) data.TimeEntry {
	return data.TimeEntry{
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

func (c *Client) StopTimeEntry(workspaceID int, entryID int) (*data.TimeEntryItem, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries/%d/stop", workspaceID, entryID)
	req, err := c.newRequest(http.MethodPatch, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var stoppedEntry data.TimeEntryItem
	if reqErr := c.doRequest(req, http.StatusOK, &stoppedEntry); reqErr != nil {
		return nil, reqErr
	}

	return &stoppedEntry, nil
}

func (c *Client) UpdateTimeEntry(workspaceID int, entryID int, entry data.TimeEntry) (*data.TimeEntryItem, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries/%d", workspaceID, entryID)
	req, err := c.newRequest(http.MethodPut, endpoint, entry)
	if err != nil {
		return nil, err
	}

	var updatedEntry data.TimeEntryItem
	if reqErr := c.doRequest(req, http.StatusOK, &updatedEntry); reqErr != nil {
		return nil, reqErr
	}

	return &updatedEntry, nil
}

func (c *Client) GetProjects(workspaceID int) ([]data.Project, error) {
	if c.Cache != nil {
		cachedProjects, cacheErr := c.Cache.GetProjects(workspaceID)
		if cacheErr == nil {
			return cachedProjects, nil
		}
	}

	endpoint := fmt.Sprintf("/workspaces/%d/projects", workspaceID)
	req, err := c.newRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var projects []data.Project
	if reqErr := c.doRequest(req, http.StatusOK, &projects); reqErr != nil {
		return nil, reqErr
	}

	if c.Cache != nil {
		if saveErr := c.Cache.SaveProjects(workspaceID, projects); saveErr != nil {
			log.Printf("Failed to save projects to cache: %v", saveErr)
		}
	}

	return projects, nil
}

func (c *Client) GetProjectIDByName(workspaceID int, projectName string) (int, error) {
	projects, err := c.GetProjects(workspaceID)
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

func (c *Client) GetHistory(from, to *time.Time) ([]data.TimeEntryItem, error) {
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

	req, err := c.newRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var timeEntries []data.TimeEntryItem
	if reqErr := c.doRequest(req, http.StatusOK, &timeEntries); reqErr != nil {
		return nil, reqErr
	}

	return timeEntries, nil
}

func (c *Client) GetProjectsLookupMap(workspaceID int) (map[int]string, error) {
	projects, err := c.GetProjects(workspaceID)
	if err != nil {
		return nil, err
	}

	lookup := make(map[int]string)
	for _, project := range projects {
		lookup[project.ID] = project.Name
	}

	return lookup, nil
}

func FormatDuration(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func (c *Client) newRequest(method, endpoint string, body any) (*http.Request, error) {
	req, err := newJSONRequest(method, c.BaseURL+endpoint, body)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	return req, nil
}

func (c *Client) doRequest(req *http.Request, expectedStatus int, result any) error {
	return doJSON(c.HTTPClient, req, expectedStatus, result)
}

func (c *Client) setAuthHeader(req *http.Request) {
	token := base64.StdEncoding.EncodeToString([]byte(c.AuthToken + ":api_token"))

	req.Header.Set("Authorization", "Basic "+token)
}
