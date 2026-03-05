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

	"github.com/spf13/cobra"
)

type BranchesCommand struct{}
type BranchesListCommand struct{}
type BranchesCreateCommand struct{}
type BranchesDeleteCommand struct{}

func branchesAPICall(method, apiURL, body string) ([]byte, error) {
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

func resolveProjectForBranches(cobraCmd *cobra.Command, logger *log.Logger) string {
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

func branchesGetProjectId(projectPath string) (int, error) {
	apiURL := config.ServerUrl + "/~api/projects/ids/" + url.PathEscape(projectPath)
	body, err := branchesAPICall("GET", apiURL, "")
	if err != nil {
		return 0, fmt.Errorf("project '%s' not found: %v", projectPath, err)
	}
	var id int
	if err := json.Unmarshal(body, &id); err != nil {
		return 0, fmt.Errorf("failed to parse project ID: %v", err)
	}
	return id, nil
}

func (command BranchesCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	listCmd := BranchesListCommand{}
	listCmd.Execute(cobraCmd, args, logger)
}

func (command BranchesListCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolveProjectForBranches(cobraCmd, logger)
	projectId, err := branchesGetProjectId(projectPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Fetch branch list
	apiURL := fmt.Sprintf("%s/~api/repositories/%d/branches", config.ServerUrl, projectId)
	body, err := branchesAPICall("GET", apiURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to list branches:", err)
		os.Exit(1)
	}
	var branches []string
	if err := json.Unmarshal(body, &branches); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse branches:", err)
		os.Exit(1)
	}

	// Fetch default branch — API returns plain text (no JSON quotes)
	defaultURL := fmt.Sprintf("%s/~api/repositories/%d/default-branch", config.ServerUrl, projectId)
	defaultBody, _ := branchesAPICall("GET", defaultURL, "")
	defaultBranch := strings.TrimSpace(string(defaultBody))

	if len(branches) == 0 {
		fmt.Println("No branches found")
		return
	}

	fmt.Printf("Branches for project '%s':\n\n", projectPath)
	for _, branch := range branches {
		marker := "  "
		if branch == defaultBranch {
			marker = "* "
		}
		fmt.Printf("  %s%s\n", marker, branch)
	}
}

func (command BranchesCreateCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	branchName := args[0]
	fromRevision, _ := cobraCmd.Flags().GetString("from")
	if fromRevision == "" {
		fromRevision = "main"
	}
	projectPath := resolveProjectForBranches(cobraCmd, logger)
	projectId, err := branchesGetProjectId(projectPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"branchName": branchName,
		"revision":   fromRevision,
	}
	payloadBytes, _ := json.Marshal(payload)

	apiURL := fmt.Sprintf("%s/~api/repositories/%d/branches", config.ServerUrl, projectId)
	_, err = branchesAPICall("POST", apiURL, string(payloadBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create branch:", err)
		os.Exit(1)
	}
	fmt.Printf("Branch '%s' created from '%s'\n", branchName, fromRevision)
}

func (command BranchesDeleteCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	branchName := args[0]
	projectPath := resolveProjectForBranches(cobraCmd, logger)
	projectId, err := branchesGetProjectId(projectPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// IMPORTANT: Do NOT url.PathEscape(branchName) — slashes must pass through as-is
	// The server uses greedy regex {branch:.*} which handles slashes natively
	apiURL := fmt.Sprintf("%s/~api/repositories/%d/branches/%s", config.ServerUrl, projectId, branchName)
	_, err = branchesAPICall("DELETE", apiURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to delete branch:", err)
		os.Exit(1)
	}
	fmt.Printf("Branch '%s' deleted\n", branchName)
}
