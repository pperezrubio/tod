package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

type CreateProjectCommand struct{}

func (command CreateProjectCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	name := args[0]
	description, _ := cobraCmd.Flags().GetString("description")
	parentPath, _ := cobraCmd.Flags().GetString("parent")

	payload := map[string]interface{}{
		"name":                name,
		"description":         description,
		"codeManagement":      true,
		"issueManagement":     true,
		"gitPackConfig":       map[string]interface{}{},
		"codeAnalysisSetting": map[string]interface{}{},
	}

	if parentPath != "" {
		parentId, err := resolveProjectId(parentPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to resolve parent project ID:", err)
			os.Exit(1)
		}
		payload["parent"] = map[string]interface{}{
			"id": parentId,
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to marshal request body:", err)
		os.Exit(1)
	}

	apiURL := config.ServerUrl + "/~api/projects"

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create request:", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to send request:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read response body:", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "Failed to create project: HTTP %d — %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// OneDev returns 200 with just the project ID as a number,
	// or 201 with the full project object
	var projectId int
	if err := json.Unmarshal(body, &projectId); err == nil {
		fmt.Printf("Created project '%s' (ID: %d)\n", name, projectId)
		return
	}

	var created map[string]interface{}
	if err := json.Unmarshal(body, &created); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse response:", err)
		os.Exit(1)
	}

	id := int(created["id"].(float64))
	createdName, _ := created["name"].(string)
	fmt.Printf("Created project '%s' (ID: %d)\n", createdName, id)
}

// resolveProjectId resolves a project path to its numeric ID via the REST API.
func resolveProjectId(projectPath string) (int, error) {
	apiURL := config.ServerUrl + "/~api/projects"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)

	q := req.URL.Query()
	q.Set("query", `"Path" is "`+projectPath+`"`)
	q.Set("offset", "0")
	q.Set("count", "1")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d — %s", resp.StatusCode, string(body))
	}

	var projects []map[string]interface{}
	if err := json.Unmarshal(body, &projects); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(projects) == 0 {
		return 0, fmt.Errorf("project '%s' not found", projectPath)
	}

	id := int(projects[0]["id"].(float64))
	return id, nil
}
