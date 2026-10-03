package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"

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
	ProjectIDByName(ctx context.Context, workspaceID int, projectName string) (int, error)
	CreateTimeEntry(ctx context.Context, workspaceID int, entry api.TimeEntry) (*api.TimeEntry, error)
	ProjectNames(ctx context.Context, workspaceID int, needed ...int) (map[int]string, error)
}

func newStartCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new time entry",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, workspaceID, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			projectName, err := cmd.Flags().GetString("project")
			if err != nil {
				return fmt.Errorf("failed to get project flag: %w", err)
			}

			client := newTogglClient(v, token)
			projectID, resolvedProject, err := findProjectIDForEntry(cmd.Context(), v, projectName, client, workspaceID)
			if err != nil {
				return fmt.Errorf("failed to find project ID: %w", err)
			}

			description := getDescription(v, cmd.ErrOrStderr(), args, resolvedProject)
			return runStart(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), client, description, workspaceID, projectID)
		},
	}

	cmd.Flags().StringP("project", "p", "", "Project for the time entry")

	return cmd
}

func runStart(ctx context.Context, out, errOut io.Writer, client StartService, description string, workspaceID, projectID int) error {
	timeEntry := api.NewTimeEntry(description, workspaceID, projectID, false)

	createdEntry, err := client.CreateTimeEntry(ctx, workspaceID, timeEntry)
	if err != nil {
		return fmt.Errorf("failed to create time entry: %w", err)
	}

	projectsMap, err := client.ProjectNames(ctx, workspaceID, createdEntry.ProjectID)
	if err != nil {
		// Non-fatal: the entry was already created. Show it without project name.
		fmt.Fprintln(errOut, "warning: failed to get projects, showing entry without project name:", err)
		projectsMap = nil
	}

	start, err := time.Parse(time.RFC3339Nano, createdEntry.Start)
	if err != nil {
		start, err = time.Parse(time.RFC3339, createdEntry.Start)
		if err != nil {
			return fmt.Errorf("failed to parse start time: %w", err)
		}
	}

	return outputCurrentEntry(out, &api.TimeEntryItem{
		ID:          createdEntry.ID,
		Description: createdEntry.Description,
		ProjectID:   createdEntry.ProjectID,
		Start:       start,
	}, projectsMap)
}

// findProjectIDForEntry resolves the project for the entry, returning both its
// id and the name it was resolved to (the config key when detected from the
// current path), so the caller can look up project-specific settings.
func findProjectIDForEntry(ctx context.Context, v *viper.Viper, projectName string, client StartService, workspaceID int) (int, string, error) {
	if projectName == "" {
		currentPath, err := os.Getwd()
		if err != nil {
			return 0, "", fmt.Errorf("failed to get current working directory: %w", err)
		}

		projectName, err = findProjectNameFromConfig(v, currentPath)
		if err != nil {
			return 0, "", fmt.Errorf("failed to find project name from config: %w", err)
		}
	}

	if projectName == "" {
		return 0, "", errors.New("no project name provided and no matching project found in config for current path")
	}

	projectID, err := client.ProjectIDByName(ctx, workspaceID, projectName)
	if err != nil || projectID == 0 {
		return 0, "", fmt.Errorf("failed to get project ID for '%s': %w", projectName, err)
	}

	return projectID, projectName, nil
}

func findProjectNameFromConfig(v *viper.Viper, currentPath string) (string, error) {
	var projects map[string]ProjectConfig
	err := v.UnmarshalKey("projects", &projects)
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

// getDescription returns the description given on the command line, falling
// back to a ticket number detected from the current directory's name. When
// nothing can be detected the user is told why the entry has no description.
func getDescription(v *viper.Viper, errOut io.Writer, args []string, projectName string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}

	description, dir := detectDescriptionFromCurrentPath(v, errOut, projectName)
	if description == "" {
		fmt.Fprintf(errOut, "warning: could not detect a ticket number from directory name %q "+
			"(no match, or more than one candidate), starting entry without description; "+
			"pass a description or set ticket_pattern in the config\n", dir)
	}

	return description
}

// detectDescriptionFromCurrentPath returns the ticket number detected from the
// current directory's name, along with that name.
func detectDescriptionFromCurrentPath(v *viper.Viper, errOut io.Writer, projectName string) (description, dir string) {
	currentPath, err := os.Getwd()
	if err != nil {
		return "", ""
	}

	dir = filepath.Base(currentPath)
	return getTicketNumberFromPath(dir, ticketPattern(v, errOut, projectName)), dir
}

// ticketPattern returns the expression used to pull a ticket number out of a
// directory name. A project's own `ticket_pattern` wins over the global
// `start.ticket_pattern`, which wins over defaultTicketPattern. A pattern that
// does not compile is reported and the default is used instead.
func ticketPattern(v *viper.Viper, errOut io.Writer, projectName string) *regexp.Regexp {
	pattern, key := configuredTicketPattern(v, projectName)
	if pattern == "" {
		return defaultTicketRe
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(errOut, "warning: invalid %s %q, using the default instead: %v\n", key, pattern, err)
		return defaultTicketRe
	}

	return re
}

// configuredTicketPattern returns the configured pattern and the config key it
// came from, or an empty pattern when nothing is configured.
func configuredTicketPattern(v *viper.Viper, projectName string) (pattern, key string) {
	if projectName != "" {
		projectKey := "projects." + projectName + ".ticket_pattern"
		if p := v.GetString(projectKey); p != "" {
			return p, projectKey
		}
	}

	return v.GetString("start.ticket_pattern"), "start.ticket_pattern"
}

// getTicketNumberFromPath extracts a ticket number from a directory name.
// It returns "" unless the name holds exactly one distinct candidate: a name
// like `proj-2024-fix-123` is ambiguous, and an empty description beats a
// confidently wrong one.
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
