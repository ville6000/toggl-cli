package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/data"
	"github.com/ville6000/toggl-cli/internal/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ProjectConfig holds project path mappings from config.
type ProjectConfig struct {
	Paths []string `mapstructure:"paths"`
}

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
		projectId, err := findProjectIdForEntry(projectName, client, workspaceId)
		if err != nil {
			return fmt.Errorf("failed to find project ID: %w", err)
		}

		description := getDescription(args)
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

func findProjectIdForEntry(projectName string, client StartService, workspaceID int) (int, error) {
	var err error

	if projectName == "" {
		currentPath, err := os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("failed to get current working directory: %w", err)
		}

		projectName, err = findProjectNameFromConfig(currentPath)
		if err != nil {
			return 0, fmt.Errorf("failed to find project name from config: %w", err)
		}
	}

	if projectName == "" {
		return 0, fmt.Errorf("no project name provided and no matching project found in config for current path")
	}

	projectId, err := client.GetProjectIdByName(workspaceID, projectName)
	if err != nil || projectId == 0 {
		return 0, fmt.Errorf("failed to get project ID for '%s': %w", projectName, err)
	}

	return projectId, nil
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

func getDescription(args []string) string {
	var description string

	if len(args) > 0 {
		description = args[0]
	}

	if description == "" {
		description = detectDescriptionFromCurrentPath()
	}

	return description
}

func detectDescriptionFromCurrentPath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		return ""
	}

	parts := strings.Split(currentPath, string(os.PathSeparator))

	if len(parts) == 0 {
		return ""
	}

	lastPart := parts[len(parts)-1]
	ticketNumber := getTicketNumberFromPath(lastPart)

	if ticketNumber != "" {
		return ticketNumber
	}

	return ""
}

func getTicketNumberFromPath(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
