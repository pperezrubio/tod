package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

type PrsCommand struct{}

func colorizePRStatus(status string) string {
	switch status {
	case "OPEN":
		return wrapWithGreen(status)
	case "MERGED":
		return wrapWithColor(status, "36")
	case "DISCARDED":
		return wrapWithRed(status)
	default:
		return status
	}
}

func prsAPIGet(apiURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	return makeAPICall(req)
}

func prsAPIPost(apiURL string, payload []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return makeAPICall(req)
}

func resolvePRProject(cobraCmd *cobra.Command, logger *log.Logger) string {
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

func (command PrsCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolvePRProject(cobraCmd, logger)
	count, _ := cobraCmd.Flags().GetInt("count")
	query, _ := cobraCmd.Flags().GetString("query")
	status, _ := cobraCmd.Flags().GetString("status")

	var fullQuery string
	if query != "" {
		fullQuery = query
	} else {
		fullQuery = `"Target Project" is "` + projectPath + `"`
		switch status {
		case "open":
			fullQuery += " and open"
		case "merged":
			fullQuery += " and merged"
		case "discarded":
			fullQuery += " and discarded"
		}
	}

	queryParams := url.Values{
		"query":  {fullQuery},
		"offset": {"0"},
		"count":  {strconv.Itoa(count)},
	}
	apiURL := config.ServerUrl + "/~api/pulls?" + queryParams.Encode()

	body, err := prsAPIGet(apiURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to list pull requests:", err)
		os.Exit(1)
	}

	var prs []map[string]interface{}
	if err := json.Unmarshal(body, &prs); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse response:", err)
		os.Exit(1)
	}

	if len(prs) == 0 {
		fmt.Println("No pull requests found.")
		return
	}

	for _, pr := range prs {
		number := int(pr["number"].(float64))
		title, _ := pr["title"].(string)
		prStatus, _ := pr["status"].(string)
		sourceBranch, _ := pr["sourceBranch"].(string)
		targetBranch, _ := pr["targetBranch"].(string)
		fmt.Printf("#%-5d  %-20s  %s  (%s -> %s)\n", number, colorizePRStatus(prStatus), title, sourceBranch, targetBranch)
	}
}

func (command PrsCommand) ExecuteCreate(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolvePRProject(cobraCmd, logger)
	title, _ := cobraCmd.Flags().GetString("title")
	source, _ := cobraCmd.Flags().GetString("source")
	target, _ := cobraCmd.Flags().GetString("target")
	description, _ := cobraCmd.Flags().GetString("description")

	if title == "" {
		fmt.Fprintln(os.Stderr, "Flag --title is required")
		os.Exit(1)
	}
	if source == "" {
		fmt.Fprintln(os.Stderr, "Flag --source is required")
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"title": title, "sourceBranch": source, "targetBranch": target, "description": description,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to marshal request body:", err)
		os.Exit(1)
	}

	apiURL := config.ServerUrl + "/~api/mcp-helper/create-pull-request?currentProject=" + url.QueryEscape(projectPath)
	body, err := prsAPIPost(apiURL, payloadBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create pull request:", err)
		os.Exit(1)
	}

	var pr map[string]interface{}
	if err := json.Unmarshal(body, &pr); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse response:", err)
		os.Exit(1)
	}
	number := int(pr["number"].(float64))
	prTitle, _ := pr["title"].(string)
	fmt.Printf("Created pull request #%d: %s\n", number, prTitle)
}

func (command PrsCommand) ExecuteMerge(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Exactly one pull request number is required")
		os.Exit(1)
	}
	prNumber := args[0]
	projectPath := resolvePRProject(cobraCmd, logger)
	strategy, _ := cobraCmd.Flags().GetString("strategy")
	deleteBranch, _ := cobraCmd.Flags().GetBool("delete-branch")

	var mergeStrategy string
	switch strategy {
	case "merge-commit":
		mergeStrategy = "CREATE_MERGE_COMMIT"
	case "squash-merge":
		mergeStrategy = "SQUASH_SOURCE_BRANCH_COMMITS"
	case "rebase-merge":
		mergeStrategy = "REBASE_SOURCE_BRANCH_COMMITS"
	default:
		fmt.Fprintln(os.Stderr, "Invalid --strategy value. Must be one of: merge-commit, squash-merge, rebase-merge")
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"action":             "MERGE",
		"mergeStrategy":      mergeStrategy,
		"deleteSourceBranch": deleteBranch,
		"commitMessage":      fmt.Sprintf("Merge pull request #%s", prNumber),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to marshal request body:", err)
		os.Exit(1)
	}

	queryParams := url.Values{
		"currentProject":       {projectPath},
		"pullRequestReference": {prNumber},
	}
	apiURL := config.ServerUrl + "/~api/mcp-helper/process-pull-request?" + queryParams.Encode()
	_, err = prsAPIPost(apiURL, payloadBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to merge pull request:", err)
		os.Exit(1)
	}
	fmt.Printf("Merged pull request #%s\n", prNumber)
}

type PrsApproveCommand struct{}
type PrsRequestChangesCommand struct{}

func prsGetCurrentUserId() (int, error) {
	apiURL := config.ServerUrl + "/~api/users?offset=0&count=1"
	body, err := prsAPIGet(apiURL)
	if err != nil {
		return 0, err
	}
	var users []map[string]interface{}
	if err := json.Unmarshal(body, &users); err != nil || len(users) == 0 {
		return 0, fmt.Errorf("failed to get current user")
	}
	id := int(users[0]["id"].(float64))
	return id, nil
}

func prsGetPRId(projectPath, prNumber string) (int, error) {
	query := fmt.Sprintf(`"Number" is "%s#%s"`, projectPath, prNumber)
	apiURL := config.ServerUrl + "/~api/pulls?query=" + url.QueryEscape(query) + "&offset=0&count=1"
	body, err := prsAPIGet(apiURL)
	if err != nil {
		return 0, fmt.Errorf("PR #%s not found: %v", prNumber, err)
	}
	var prs []map[string]interface{}
	if err := json.Unmarshal(body, &prs); err != nil || len(prs) == 0 {
		return 0, fmt.Errorf("PR #%s not found", prNumber)
	}
	return int(prs[0]["id"].(float64)), nil
}

func (command PrsApproveCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	prNumber := args[0]
	projectPath := resolvePRProject(cobraCmd, logger)

	prId, err := prsGetPRId(projectPath, prNumber)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	userId, err := prsGetCurrentUserId()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get current user:", err)
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"request": map[string]interface{}{"id": prId},
		"user":    map[string]interface{}{"id": userId},
		"status":  "APPROVED",
	}
	payloadBytes, _ := json.Marshal(payload)

	apiURL := config.ServerUrl + "/~api/pull-request-reviews"
	_, err = prsAPIPost(apiURL, payloadBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to approve PR:", err)
		os.Exit(1)
	}
	fmt.Printf("Approved pull request #%s\n", prNumber)
}

func (command PrsRequestChangesCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	prNumber := args[0]
	projectPath := resolvePRProject(cobraCmd, logger)

	prId, err := prsGetPRId(projectPath, prNumber)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	userId, err := prsGetCurrentUserId()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get current user:", err)
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"request": map[string]interface{}{"id": prId},
		"user":    map[string]interface{}{"id": userId},
		"status":  "REQUESTED_FOR_CHANGES",
	}
	payloadBytes, _ := json.Marshal(payload)

	apiURL := config.ServerUrl + "/~api/pull-request-reviews"
	_, err = prsAPIPost(apiURL, payloadBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to request changes on PR:", err)
		os.Exit(1)
	}
	fmt.Printf("Requested changes on pull request #%s\n", prNumber)
}
