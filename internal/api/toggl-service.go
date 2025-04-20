package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (c *APIClient) GetWorkspaces() ([]Workspace, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/workspaces", nil)
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get workspaces: %s", resp.Status)
	}

	var workspaces []Workspace
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, err
	}

	return workspaces, nil
}

func (c *APIClient) GetCurrentTimerEntry() (*CurrentTimeEntry, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/me/time_entries/current", nil)
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get current timer entry: %s", resp.Status)
	}

	var entry CurrentTimeEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

func (c *APIClient) CreateTimeEntry(workspaceId int, entry TimeEntry) (*TimeEntry, error) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/workspaces/%d/time_entries", workspaceId)
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create time entry: %s", resp.Status)
	}

	var createdEntry TimeEntry
	if err := json.NewDecoder(resp.Body).Decode(&createdEntry); err != nil {
		return nil, err
	}

	return &createdEntry, nil
}

func (c *APIClient) StopTimeEntry(workspaceId int, entryId int) (*TimeEntry, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/time_entries/%d/stop", workspaceId, entryId)
	req, err := http.NewRequest(http.MethodPatch, c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to stop time entry: %s", resp.Status)
	}

	var stoppedEntry TimeEntry
	if err := json.NewDecoder(resp.Body).Decode(&stoppedEntry); err != nil {
		return nil, err
	}

	return &stoppedEntry, nil
}

func (c *APIClient) GetProjects(workspaceId int) ([]Project, error) {
	endpoint := fmt.Sprintf("/workspaces/%d/projects", workspaceId)
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get projects: %s", resp.Status)
	}
	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (c *APIClient) GetProjectIdByName(workspaceId int, projectName string) (int, error) {
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

func (c *APIClient) GetHistory(from, to time.Time) ([]CurrentTimeEntry, error) {
	endpoint := fmt.Sprintf("/me/time_entries?start_date=%s&end_date=%s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get history: %s", resp.Status)
	}
	var timeEntries []CurrentTimeEntry
	if err := json.NewDecoder(resp.Body).Decode(&timeEntries); err != nil {
		return nil, err
	}
	return timeEntries, nil
}

func FormatDuration(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func (c *APIClient) setDefaultRequestHeaders(req *http.Request) {
	token := base64.StdEncoding.EncodeToString([]byte(c.AuthToken + ":api_token"))

	req.Header.Set("Authorization", "Basic "+token)
	req.Header.Set("Content-Type", "application/json")
}
