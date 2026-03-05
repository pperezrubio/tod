package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type IterationsCommand struct{}
type IterationsListCommand struct{}
type IterationsCreateCommand struct{}
type IterationsCloseCommand struct{}

func iterationsAPICall(method, apiURL, body string) ([]byte, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, apiURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return makeAPICall(req)
}

func resolveProjectForIterations(cobraCmd *cobra.Command, logger *log.Logger) string {
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

func iterationsGetProjectId(projectPath string) (int, error) {
	apiURL := config.ServerUrl + "/~api/projects/ids/" + url.PathEscape(projectPath)
	body, err := iterationsAPICall("GET", apiURL, "")
	if err != nil {
		return 0, fmt.Errorf("project '%s' not found: %v", projectPath, err)
	}
	var id int
	if err := json.Unmarshal(body, &id); err != nil {
		return 0, fmt.Errorf("failed to parse project ID: %v", err)
	}
	return id, nil
}

// parseDateToEpochDay converts "YYYY-MM-DD" to epoch day number (days since 1970-01-01)
func parseDateToEpochDay(dateStr string) (int, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid date format '%s' (expected YYYY-MM-DD): %v", dateStr, err)
	}
	return int(t.Unix() / 86400), nil
}

// epochDayToDate converts epoch day number back to "YYYY-MM-DD" string
func epochDayToDate(day int) string {
	t := time.Unix(int64(day)*86400, 0).UTC()
	return t.Format("2006-01-02")
}

func (command IterationsCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	listCmd := IterationsListCommand{}
	listCmd.Execute(cobraCmd, args, logger)
}

func (command IterationsListCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolveProjectForIterations(cobraCmd, logger)
	projectId, err := iterationsGetProjectId(projectPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	apiURL := fmt.Sprintf("%s/~api/projects/%d/iterations?offset=0&count=100", config.ServerUrl, projectId)
	body, err := iterationsAPICall("GET", apiURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to list iterations:", err)
		os.Exit(1)
	}
	var iterations []map[string]interface{}
	if err := json.Unmarshal(body, &iterations); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse iterations:", err)
		os.Exit(1)
	}
	if len(iterations) == 0 {
		fmt.Printf("No iterations found for project '%s'\n", projectPath)
		return
	}

	fmt.Printf("Iterations for project '%s':\n\n", projectPath)
	fmt.Printf("  %-6s  %-20s  %-12s  %-12s  %s\n", "ID", "Name", "Start", "Due", "Status")
	fmt.Printf("  %-6s  %-20s  %-12s  %-12s  %s\n", "------", "--------------------", "------------", "------------", "------")
	for _, it := range iterations {
		id := int(it["id"].(float64))
		name, _ := it["name"].(string)
		closed, _ := it["closed"].(bool)
		status := "Open"
		if closed {
			status = "Closed"
		}
		startStr := "-"
		if startDay, ok := it["startDay"]; ok && startDay != nil {
			startStr = epochDayToDate(int(startDay.(float64)))
		}
		dueStr := "-"
		if dueDay, ok := it["dueDay"]; ok && dueDay != nil {
			dueStr = epochDayToDate(int(dueDay.(float64)))
		}
		fmt.Printf("  %-6d  %-20s  %-12s  %-12s  %s\n", id, name, startStr, dueStr, status)
	}
}

func (command IterationsCreateCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	name := args[0]
	projectPath := resolveProjectForIterations(cobraCmd, logger)
	projectId, err := iterationsGetProjectId(projectPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	startStr, _ := cobraCmd.Flags().GetString("start")
	dueStr, _ := cobraCmd.Flags().GetString("due")
	description, _ := cobraCmd.Flags().GetString("description")

	payload := map[string]interface{}{
		"projectId": projectId,
		"name":      name,
		"closed":    false,
	}
	if description != "" {
		payload["description"] = description
	}
	if startStr != "" {
		day, err := parseDateToEpochDay(startStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		payload["startDay"] = day
	}
	if dueStr != "" {
		day, err := parseDateToEpochDay(dueStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		payload["dueDay"] = day
	}

	payloadBytes, _ := json.Marshal(payload)
	apiURL := config.ServerUrl + "/~api/iterations"
	resp, err := iterationsAPICall("POST", apiURL, string(payloadBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create iteration:", err)
		os.Exit(1)
	}
	var iterationId int64
	json.Unmarshal(resp, &iterationId)
	fmt.Printf("Iteration '%s' created (ID: %d)\n", name, iterationId)
}

func (command IterationsCloseCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	iterationIdStr := args[0]
	// Parse iteration ID
	var iterationId int
	if _, err := fmt.Sscanf(iterationIdStr, "%d", &iterationId); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid iteration ID '%s': must be a number\n", iterationIdStr)
		os.Exit(1)
	}

	// Fetch current iteration (must re-send full object on update)
	apiURL := fmt.Sprintf("%s/~api/iterations/%d", config.ServerUrl, iterationId)
	body, err := iterationsAPICall("GET", apiURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to find iteration:", err)
		os.Exit(1)
	}
	var iteration map[string]interface{}
	if err := json.Unmarshal(body, &iteration); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse iteration:", err)
		os.Exit(1)
	}

	name, _ := iteration["name"].(string)

	// Set closed = true and re-POST full object
	iteration["closed"] = true
	payloadBytes, _ := json.Marshal(iteration)
	_, err = iterationsAPICall("POST", apiURL, string(payloadBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to close iteration:", err)
		os.Exit(1)
	}
	fmt.Printf("Iteration '%s' closed\n", name)
}
