package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ville6000/toggl-cli/internal/data"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type ProjectService interface {
	GetProjects(workspaceId int) ([]data.Project, error)
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

func (c *Client) CreateTimeEntry(workspaceId int, entry data.TimeEntry) (*data.TimeEntry, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries", workspaceId)
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

func (c *Client) StopTimeEntry(workspaceId int, entryId int) (*data.TimeEntryItem, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries/%d/stop", workspaceId, entryId)
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

func (c *Client) GetProjects(workspaceId int) ([]data.Project, error) {
	cachedProjects, err := c.Cache.GetProjects(workspaceId)
	if err == nil {
		return cachedProjects, nil
	}

	endpoint := fmt.Sprintf("/workspaces/%d/projects", workspaceId)
	req, err := c.newRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var projects []data.Project
	if reqErr := c.doRequest(req, http.StatusOK, &projects); reqErr != nil {
		return nil, reqErr
	}

	err = c.Cache.SaveProjects(workspaceId, projects)
	if err != nil {
		log.Printf("Failed to save projects to cache: %v", err)
	}

	return projects, nil
}

func (c *Client) GetProjectIdByName(workspaceId int, projectName string) (int, error) {
	projects, err := c.GetProjects(workspaceId)
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
	if from != nil {
		queryParams = append(queryParams, fmt.Sprintf("start_date=%s", from.Format("2006-01-02")))
	}

	if to != nil {
		queryParams = append(queryParams, fmt.Sprintf("end_date=%s", to.Format("2006-01-02")))
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

func (c *Client) GetProjectsLookupMap(workspaceId int) (map[int]string, error) {
	projects, err := c.GetProjects(workspaceId)
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
	var buf io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, buf)
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	return req, nil
}

func (c *Client) doRequest(req *http.Request, expectedStatus int, result any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("request failed: %s", resp.Status)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func (c *Client) setDefaultRequestHeaders(req *http.Request) {
	token := base64.StdEncoding.EncodeToString([]byte(c.AuthToken + ":api_token"))

	req.Header.Set("Authorization", "Basic "+token)
	req.Header.Set("Content-Type", "application/json")
}
