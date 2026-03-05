package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type IssuesCommand struct{}

func colorizeIssueState(state string) string {
	switch state {
	case "Open":
		return wrapWithGreen(state)
	case "Closed":
		return wrapWithRed(state)
	default:
		return state
	}
}

// issuesAPICall is a convenience wrapper for issues API calls.
func issuesAPICall(method, apiURL, body string) ([]byte, error) {
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

// issuesGetProjectId resolves a project path to its numeric ID.
func issuesGetProjectId(projectPath string) (int, error) {
	apiURL := config.ServerUrl + "/~api/projects/ids/" + url.PathEscape(projectPath)
	body, err := issuesAPICall("GET", apiURL, "")
	if err != nil {
		return 0, fmt.Errorf("project '%s' not found: %v", projectPath, err)
	}
	var id int
	if err := json.Unmarshal(body, &id); err != nil {
		return 0, fmt.Errorf("failed to parse project ID: %v", err)
	}
	return id, nil
}

func resolveProjectForIssues(cobraCmd *cobra.Command, logger *log.Logger) string {
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

// IssuesListCommand lists issues for a project.
type IssuesListCommand struct{}

func (command IssuesListCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolveProjectForIssues(cobraCmd, logger)
	count, _ := cobraCmd.Flags().GetInt("count")
	query, _ := cobraCmd.Flags().GetString("query")
	state, _ := cobraCmd.Flags().GetString("state")

	var issueQuery string
	if query != "" {
		issueQuery = query
	} else {
		issueQuery = `"Project" is "` + projectPath + `"`
		switch state {
		case "open":
			issueQuery += ` and "State" is "Open"`
		case "closed":
			issueQuery += ` and "State" is "Closed"`
			// "all": no state filter
		}
	}

	params := url.Values{
		"query":  {issueQuery},
		"offset": {"0"},
		"count":  {strconv.Itoa(count)},
	}
	apiURL := config.ServerUrl + "/~api/issues?" + params.Encode()

	body, err := issuesAPICall("GET", apiURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to list issues:", err)
		os.Exit(1)
	}

	var issues []map[string]interface{}
	if err := json.Unmarshal(body, &issues); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse issues response:", err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		fmt.Printf("No issues found for project '%s'\n", projectPath)
		return
	}

	fmt.Printf("Issues for project '%s':\n\n", projectPath)
	for _, issue := range issues {
		number := int(issue["number"].(float64))
		title, _ := issue["title"].(string)
		issueState := ""
		if s, ok := issue["state"].(string); ok {
			issueState = s
		}
		fmt.Printf("  #%-6d  %-20s  %s\n", number, colorizeIssueState(issueState), title)
	}
}

// IssuesCreateCommand creates a new issue.
type IssuesCreateCommand struct{}

func (command IssuesCreateCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolveProjectForIssues(cobraCmd, logger)
	title, _ := cobraCmd.Flags().GetString("title")
	description, _ := cobraCmd.Flags().GetString("description")

	if title == "" {
		fmt.Fprintln(os.Stderr, "Error: --title is required")
		os.Exit(1)
	}

	projectId, err := issuesGetProjectId(projectPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get project ID:", err)
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"project": map[string]interface{}{
			"id": projectId,
		},
		"title":       title,
		"description": description,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to marshal request body:", err)
		os.Exit(1)
	}

	apiURL := config.ServerUrl + "/~api/issues"
	body, err := issuesAPICall("POST", apiURL, string(payloadBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create issue:", err)
		os.Exit(1)
	}

	var created map[string]interface{}
	if err := json.Unmarshal(body, &created); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse create response:", err)
		os.Exit(1)
	}

	number := int(created["number"].(float64))
	fmt.Printf("Created issue #%d in %s\n", number, projectPath)
}

// IssuesEditCommand edits an existing issue.
type IssuesEditCommand struct{}

func (command IssuesEditCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: issue number is required")
		os.Exit(1)
	}

	projectPath := resolveProjectForIssues(cobraCmd, logger)
	newTitle, _ := cobraCmd.Flags().GetString("title")
	newDescription, _ := cobraCmd.Flags().GetString("description")
	issueRef := args[0]

	// Find issue ID by number
	issueQuery := `"Project" is "` + projectPath + `" and "Number" is "` + issueRef + `"`
	params := url.Values{
		"query":  {issueQuery},
		"offset": {"0"},
		"count":  {"1"},
	}
	listURL := config.ServerUrl + "/~api/issues?" + params.Encode()

	listBody, err := issuesAPICall("GET", listURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to find issue:", err)
		os.Exit(1)
	}

	var issues []map[string]interface{}
	if err := json.Unmarshal(listBody, &issues); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse issues response:", err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		fmt.Fprintf(os.Stderr, "Issue #%s not found in project '%s'\n", issueRef, projectPath)
		os.Exit(1)
	}

	issue := issues[0]
	issueId := int(issue["id"].(float64))

	// Build patch payload with only changed fields
	patchPayload := map[string]interface{}{}
	if newTitle != "" {
		patchPayload["title"] = newTitle
	}
	if newDescription != "" {
		patchPayload["description"] = newDescription
	}

	if len(patchPayload) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one of --title or --description must be specified")
		os.Exit(1)
	}

	patchBytes, err := json.Marshal(patchPayload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to marshal patch body:", err)
		os.Exit(1)
	}

	patchURL := fmt.Sprintf("%s/~api/issues/%d", config.ServerUrl, issueId)

	// Use raw HTTP call to handle 204 No Content response
	req, err := http.NewRequest("PATCH", patchURL, bytes.NewReader(patchBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create PATCH request:", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to send PATCH request:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Failed to update issue: HTTP %d — %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	fmt.Printf("Updated issue #%s\n", issueRef)
}

// IssuesCloseCommand closes an issue via the MCP helper endpoint.
type IssuesCloseCommand struct{}

func (command IssuesCloseCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: issue number is required")
		os.Exit(1)
	}

	projectPath := resolveProjectForIssues(cobraCmd, logger)
	issueRef := args[0]

	payload := `{"instruction":"close"}`

	params := url.Values{
		"currentProject": {projectPath},
		"issueReference": {issueRef},
	}
	apiURL := config.ServerUrl + "/~api/mcp-helper/change-issue-state?" + params.Encode()

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader([]byte(payload)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create request:", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to close issue:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Failed to close issue: HTTP %d — %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	fmt.Printf("Closed issue #%s\n", issueRef)
}
