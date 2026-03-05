package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ArtifactsCommand is the root artifacts command struct.
type ArtifactsCommand struct{}

// ArtifactsListCommand lists build artifacts.
type ArtifactsListCommand struct{}

// ArtifactsDownloadCommand downloads a build artifact.
type ArtifactsDownloadCommand struct{}

// artifactsAPICall is a convenience wrapper for artifacts API calls.
// Returns (body, statusCode, error).
func artifactsAPICallRaw(method, apiURL, body string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, apiURL, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request to %s: %v", apiURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response from %s: %v", apiURL, err)
	}

	return respBody, resp.StatusCode, nil
}

// artifactsAPICall returns body or error (treats non-200/204 as error).
func artifactsAPICall(method, apiURL, body string) ([]byte, error) {
	respBody, statusCode, err := artifactsAPICallRaw(method, apiURL, body)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		return nil, fmt.Errorf("HTTP %d error for endpoint %s: %s", statusCode, apiURL, string(respBody))
	}
	return respBody, nil
}

// resolveProjectForArtifacts resolves the project path from flag or git remote inference.
func resolveProjectForArtifacts(cobraCmd *cobra.Command, logger *log.Logger) string {
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

// resolveBuildId resolves a build number (e.g. "2") to the internal DB build ID.
func resolveBuildId(projectPath string, buildRef string) (int, error) {
	query := fmt.Sprintf(`"Number" is "%s#%s"`, projectPath, buildRef)
	params := url.Values{
		"query":  {query},
		"offset": {"0"},
		"count":  {"1"},
	}
	apiURL := config.ServerUrl + "/~api/builds?" + params.Encode()

	body, err := artifactsAPICall("GET", apiURL, "")
	if err != nil {
		return 0, fmt.Errorf("failed to query builds: %v", err)
	}

	var builds []map[string]interface{}
	if err := json.Unmarshal(body, &builds); err != nil {
		return 0, fmt.Errorf("failed to parse builds response: %v", err)
	}

	if len(builds) == 0 {
		return 0, fmt.Errorf("build #%s not found for project '%s'", buildRef, projectPath)
	}

	idFloat, ok := builds[0]["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected id format in builds response")
	}

	return int(idFloat), nil
}

// humanizeSize formats a byte count into a human-readable string.
func humanizeSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// artifactEntry represents a single file or directory artifact.
type artifactEntry struct {
	path      string
	isDir     bool
	sizeBytes int64
	mediaType string
}

func flattenInfoNode(node map[string]interface{}, entries *[]artifactEntry) {
	path, _ := node["path"].(string)

	if children, ok := node["children"]; ok {
		// DirectoryInfo
		entry := artifactEntry{
			path:  path + "/",
			isDir: true,
		}
		*entries = append(*entries, entry)

		if childList, ok := children.([]interface{}); ok {
			for _, child := range childList {
				if childMap, ok := child.(map[string]interface{}); ok {
					flattenInfoNode(childMap, entries)
				}
			}
		}
	} else {
		// FileInfo
		var sizeBytes int64
		if lengthFloat, ok := node["length"].(float64); ok {
			sizeBytes = int64(lengthFloat)
		}
		mediaType, _ := node["mediaType"].(string)
		entry := artifactEntry{
			path:      path,
			isDir:     false,
			sizeBytes: sizeBytes,
			mediaType: mediaType,
		}
		*entries = append(*entries, entry)
	}
}

// Execute implements the artifacts list subcommand.
func (command ArtifactsListCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolveProjectForArtifacts(cobraCmd, logger)

	buildFlag, _ := cobraCmd.Flags().GetString("build")
	if buildFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --build (-b) is required")
		os.Exit(1)
	}

	buildId, err := resolveBuildId(projectPath, buildFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to resolve build ID:", err)
		os.Exit(1)
	}

	apiURL := fmt.Sprintf("%s/~api/artifacts/%d/infos", config.ServerUrl, buildId)

	respBody, statusCode, err := artifactsAPICallRaw("GET", apiURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to list artifacts:", err)
		os.Exit(1)
	}

	// 204 No Content or empty response = no artifacts
	if statusCode == http.StatusNoContent || len(respBody) == 0 {
		fmt.Printf("No artifacts found for build #%s\n", buildFlag)
		return
	}

	if statusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Failed to list artifacts: HTTP %d: %s\n", statusCode, string(respBody))
		os.Exit(1)
	}

	// Response can be a single FileInfo/DirectoryInfo object OR an array.
	var entries []artifactEntry

	var arrayResponse []map[string]interface{}
	if err := json.Unmarshal(respBody, &arrayResponse); err == nil {
		for _, node := range arrayResponse {
			flattenInfoNode(node, &entries)
		}
	} else {
		var singleResponse map[string]interface{}
		if err := json.Unmarshal(respBody, &singleResponse); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to parse artifacts response:", err)
			os.Exit(1)
		}
		if len(singleResponse) > 0 {
			flattenInfoNode(singleResponse, &entries)
		}
	}

	if len(entries) == 0 {
		fmt.Printf("No artifacts found for build #%s\n", buildFlag)
		return
	}

	fmt.Printf("Artifacts for build #%s (project '%s'):\n\n", buildFlag, projectPath)
	fmt.Printf("  %-50s %-12s %s\n", "PATH", "SIZE", "TYPE")
	for _, entry := range entries {
		if entry.isDir {
			fmt.Printf("  %-50s %-12s\n", entry.path, "<dir>")
		} else {
			fmt.Printf("  %-50s %-12s %s\n", entry.path, humanizeSize(entry.sizeBytes), entry.mediaType)
		}
	}
}

// Execute implements the artifacts download subcommand.
func (command ArtifactsDownloadCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	projectPath := resolveProjectForArtifacts(cobraCmd, logger)

	buildFlag, _ := cobraCmd.Flags().GetString("build")
	if buildFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --build (-b) is required")
		os.Exit(1)
	}

	outputFlag, _ := cobraCmd.Flags().GetString("output")

	artifactPath := args[0]

	buildId, err := resolveBuildId(projectPath, buildFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to resolve build ID:", err)
		os.Exit(1)
	}

	// Construct download URL with artifact path escaped correctly
	downloadURL := fmt.Sprintf("%s/~api/artifacts/%d/contents/%s",
		config.ServerUrl, buildId, url.PathEscape(artifactPath))

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create download request:", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to download artifact:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "HTTP %d error downloading artifact: %s\n", resp.StatusCode, string(bodyBytes))
		os.Exit(1)
	}

	// Determine output file name
	outFile := outputFlag
	if outFile == "" {
		outFile = filepath.Base(artifactPath)
	}

	f, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create output file:", err)
		os.Exit(1)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write artifact to file:", err)
		os.Exit(1)
	}

	fmt.Printf("Downloaded %s (%d bytes)\n", outFile, written)
}
