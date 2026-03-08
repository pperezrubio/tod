package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

// fetchBuilds queries the OneDev builds API and returns parsed build data.
func fetchBuilds(projectPath string, count int, query string) ([]map[string]interface{}, error) {
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
		return nil, err
	}

	var builds []map[string]interface{}
	if err := json.Unmarshal(body, &builds); err != nil {
		return nil, err
	}

	return builds, nil
}

// printBuilds renders the build list to stdout and returns whether all builds
// have reached a terminal state (SUCCESSFUL, FAILED, CANCELLED, TIMED_OUT)
// and whether any build failed.
func printBuilds(builds []map[string]interface{}, projectPath string) (allTerminal bool, anyFailed bool) {
	if len(builds) == 0 {
		fmt.Printf("No builds found for project '%s'.\n", projectPath)
		return true, false
	}

	allTerminal = true
	anyFailed = false

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

		upper := strings.ToUpper(status)
		if upper != "SUCCESSFUL" && upper != "FAILED" && upper != "CANCELLED" && upper != "TIMED_OUT" {
			allTerminal = false
		}
		if upper == "FAILED" || upper == "TIMED_OUT" {
			anyFailed = true
		}
	}

	return allTerminal, anyFailed
}

// isTerminalStatus returns true if the build status is a final/terminal state.
func isTerminalStatus(status string) bool {
	switch strings.ToUpper(status) {
	case "SUCCESSFUL", "FAILED", "CANCELLED", "TIMED_OUT":
		return true
	default:
		return false
	}
}

func (command BuildsCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	count, _ := cobraCmd.Flags().GetInt("count")
	query, _ := cobraCmd.Flags().GetString("query")
	watch, _ := cobraCmd.Flags().GetBool("watch")
	interval, _ := cobraCmd.Flags().GetInt("interval")
	projectPath := resolveProjectForBuilds(cobraCmd, logger)

	if !watch {
		// One-shot mode: fetch and print once
		builds, err := fetchBuilds(projectPath, count, query)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to query builds:", err)
			os.Exit(1)
		}
		printBuilds(builds, projectPath)
		return
	}

	// Watch mode: poll until all builds reach terminal state
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	pollInterval := time.Duration(interval) * time.Second
	iteration := 0

	for {
		// Clear screen on subsequent iterations (not the first)
		if iteration > 0 {
			fmt.Print("\033[2J\033[H") // ANSI: clear screen + move cursor to top
		}

		builds, err := fetchBuilds(projectPath, count, query)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to query builds:", err)
			os.Exit(1)
		}

		allTerminal, anyFailed := printBuilds(builds, projectPath)

		if allTerminal {
			if anyFailed {
				fmt.Println(wrapWithRed("\nAll builds finished. Some failed."))
				os.Exit(1)
			}
			fmt.Println(wrapWithGreen("\nAll builds finished successfully."))
			return
		}

		fmt.Printf("\nWatching... (poll every %ds, Ctrl+C to stop)\n", interval)
		iteration++

		select {
		case <-sigCh:
			fmt.Println("\nStopped watching.")
			return
		case <-time.After(pollInterval):
			// continue polling
		}
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
