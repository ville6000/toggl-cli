package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/data"
	"github.com/ville6000/toggl-cli/internal/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ProjectConfig holds project path mappings from config.
type ProjectConfig struct {
	Paths []string `mapstructure:"paths"`
	// TicketPattern overrides the global ticket pattern for this project.
	TicketPattern string `mapstructure:"ticket_pattern"`
}

// defaultTicketPattern matches a standalone run of digits in a directory name:
// digits that are not glued to letters or other digits. `ticket-123` and
// `AB#123` yield `123`, while `php8`, `v2` and `2024.1` yield nothing. The
// first capture group is what ends up in the description.
const defaultTicketPattern = `(?:^|[^0-9A-Za-z])#?([0-9]+)(?:[^0-9A-Za-z]|$)`

var defaultTicketRe = regexp.MustCompile(defaultTicketPattern)

// StartService is the subset of api.Client used by the start command.
type StartService interface {
	GetProjectIdByName(workspaceId int, projectName string) (int, error)
	CreateTimeEntry(workspaceId int, entry data.TimeEntry) (*data.TimeEntry, error)
	GetProjectsLookupMap(workspaceId int) (map[int]string, error)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new time entry",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceId, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		projectName, err := cmd.Flags().GetString("project")
		if err != nil {
			return fmt.Errorf("failed to get project flag: %w", err)
		}

		client := api.NewAPIClient(token)
		projectId, resolvedProject, err := findProjectIdForEntry(projectName, client, workspaceId)
		if err != nil {
			return fmt.Errorf("failed to find project ID: %w", err)
		}

		description := getDescription(args, resolvedProject)
		return runStart(client, description, workspaceId, projectId)
	},
}

func runStart(client StartService, description string, workspaceId, projectId int) error {
	timeEntry := data.TimeEntry{
		CreatedWith: "toggl-cli",
		Description: description,
		Tags:        []string{},
		WorkspaceID: workspaceId,
		Duration:    -1,
		Start:       time.Now().Format(time.RFC3339),
		ProjectID:   projectId,
	}

	createdEntry, err := client.CreateTimeEntry(workspaceId, timeEntry)
	if err != nil {
		return fmt.Errorf("failed to create time entry: %w", err)
	}

	projectsMap, err := client.GetProjectsLookupMap(workspaceId)
	if err != nil {
		// Non-fatal: the entry was already created. Show it without project name.
		fmt.Fprintln(os.Stderr, "warning: failed to get projects, showing entry without project name:", err)
		projectsMap = nil
	}

	start, err := time.Parse(time.RFC3339Nano, createdEntry.Start)
	if err != nil {
		start, err = time.Parse(time.RFC3339, createdEntry.Start)
		if err != nil {
			return fmt.Errorf("failed to parse start time: %w", err)
		}
	}

	return outputCurrentEntry(&data.TimeEntryItem{
		ID:          createdEntry.ID,
		Description: createdEntry.Description,
		ProjectID:   createdEntry.ProjectID,
		Start:       start,
	}, projectsMap)
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringP("project", "p", "", "Project for the time entry")
}

// findProjectIdForEntry resolves the project for the entry, returning both its
// id and the name it was resolved to (the config key when detected from the
// current path), so the caller can look up project-specific settings.
func findProjectIdForEntry(projectName string, client StartService, workspaceID int) (int, string, error) {
	if projectName == "" {
		currentPath, err := os.Getwd()
		if err != nil {
			return 0, "", fmt.Errorf("failed to get current working directory: %w", err)
		}

		projectName, err = findProjectNameFromConfig(currentPath)
		if err != nil {
			return 0, "", fmt.Errorf("failed to find project name from config: %w", err)
		}
	}

	if projectName == "" {
		return 0, "", fmt.Errorf("no project name provided and no matching project found in config for current path")
	}

	projectId, err := client.GetProjectIdByName(workspaceID, projectName)
	if err != nil || projectId == 0 {
		return 0, "", fmt.Errorf("failed to get project ID for '%s': %w", projectName, err)
	}

	return projectId, projectName, nil
}

func findProjectNameFromConfig(currentPath string) (string, error) {
	var projects map[string]ProjectConfig
	err := viper.UnmarshalKey("projects", &projects)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal projects from config: %w", err)
	}

	for name, p := range projects {
		if len(p.Paths) == 0 {
			continue
		}

		for _, path := range p.Paths {
			if path == currentPath || strings.HasPrefix(currentPath, path) {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("no matching project found for current path '%s'", currentPath)
}

func getDescription(args []string, projectName string) string {
	var description string

	if len(args) > 0 {
		description = args[0]
	}

	if description == "" {
		description = detectDescriptionFromCurrentPath(projectName)
	}

	return description
}

func detectDescriptionFromCurrentPath(projectName string) string {
	currentPath, err := os.Getwd()
	if err != nil {
		return ""
	}

	return getTicketNumberFromPath(filepath.Base(currentPath), ticketPattern(projectName))
}

// ticketPattern returns the expression used to pull a ticket number out of a
// directory name. A project's own `ticket_pattern` wins over the global
// `start.ticket_pattern`, which wins over defaultTicketPattern. A pattern that
// does not compile is reported and the default is used instead.
func ticketPattern(projectName string) *regexp.Regexp {
	pattern, key := configuredTicketPattern(projectName)
	if pattern == "" {
		return defaultTicketRe
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid %s %q, using the default instead: %v\n", key, pattern, err)
		return defaultTicketRe
	}

	return re
}

// configuredTicketPattern returns the configured pattern and the config key it
// came from, or an empty pattern when nothing is configured.
func configuredTicketPattern(projectName string) (pattern, key string) {
	if projectName != "" {
		projectKey := "projects." + projectName + ".ticket_pattern"
		if p := viper.GetString(projectKey); p != "" {
			return p, projectKey
		}
	}

	return viper.GetString("start.ticket_pattern"), "start.ticket_pattern"
}

// getTicketNumberFromPath extracts a ticket number from a directory name.
// It returns "" unless the name holds exactly one distinct candidate: a name
// like `proj-2024-fix-123` is ambiguous, and an empty description beats a
// confidently wrong one, since it also feeds `7pace sync` as a work item id.
func getTicketNumberFromPath(s string, re *regexp.Regexp) string {
	var found string

	for _, candidate := range ticketCandidates(s, re) {
		if found == "" {
			found = candidate
			continue
		}

		if candidate != found {
			return ""
		}
	}

	return found
}

// ticketCandidates returns every match of re in s, using the first capture
// group when the pattern has one. Scanning resumes at the end of the captured
// text rather than the end of the whole match, so a separator consumed as a
// boundary does not hide the next candidate (`a-1-2` yields both 1 and 2).
func ticketCandidates(s string, re *regexp.Regexp) []string {
	var candidates []string

	for pos := 0; pos <= len(s); {
		loc := re.FindStringSubmatchIndex(s[pos:])
		if loc == nil {
			break
		}

		start, end := loc[0], loc[1]
		if len(loc) >= 4 && loc[2] >= 0 {
			start, end = loc[2], loc[3]
		}

		if end > start {
			candidates = append(candidates, s[pos+start:pos+end])
		}

		if end <= 0 {
			// Empty match: step forward to avoid looping forever.
			end = loc[1]
			if end <= 0 {
				end = 1
			}
		}
		pos += end
	}

	return candidates
}
