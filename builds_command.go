package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type BuildsCommand struct {
}

func resolveProjectForBuilds(cobraCmd *cobra.Command, logger *log.Logger) string {
	projectPath, _ := cobraCmd.Flags().GetString("project")
	if projectPath == "" {
		workingDir, _ := os.Getwd()
		_, inferredProject, err := inferProject(workingDir, logger)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to infer project:", err)
			os.Exit(1)
		}
		projectPath = inferredProject
	}
	return projectPath
}

func (command BuildsCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	count, _ := cobraCmd.Flags().GetInt("count")
	query, _ := cobraCmd.Flags().GetString("query")
	projectPath := resolveProjectForBuilds(cobraCmd, logger)

	// Build query: if user provides a custom query, scope it to the project;
	// otherwise default to showing all builds for the project.
	var buildQuery string
	if query != "" {
		buildQuery = fmt.Sprintf(`"Project" is "%s" and (%s)`, projectPath, query)
	} else {
		buildQuery = fmt.Sprintf(`"Project" is "%s"`, projectPath)
	}

	apiURL := config.ServerUrl + "/~api/builds?offset=0&count=" + fmt.Sprintf("%d", count) +
		"&query=" + url.QueryEscape(buildQuery)

	body, err := makeAPICallSimple("GET", apiURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to query builds:", err)
		os.Exit(1)
	}

	var builds []map[string]interface{}
	if err := json.Unmarshal(body, &builds); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse builds:", err)
		os.Exit(1)
	}

	if len(builds) == 0 {
		fmt.Printf("No builds found for project '%s'.\n", projectPath)
		return
	}

	for _, build := range builds {
		number := int(build["number"].(float64))
		status, _ := build["status"].(string)
		jobName, _ := build["jobName"].(string)
		commitHash, _ := build["commitHash"].(string)

		statusColored := colorizeStatus(status)
		hash := commitHash
		if len(hash) > 8 {
			hash = hash[:8]
		}

		fmt.Printf("#%-4d %-12s %-30s %s\n", number, statusColored, jobName, hash)
	}
}

func colorizeStatus(status string) string {
	switch strings.ToUpper(status) {
	case "SUCCESSFUL":
		return wrapWithGreen(status)
	case "FAILED", "CANCELLED", "TIMED_OUT":
		return wrapWithRed(status)
	case "RUNNING":
		return wrapWithColor(status, "33") // yellow
	default:
		return status
	}
}
