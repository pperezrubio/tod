package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

type LogsCommand struct {
}

func (command LogsCommand) Execute(cobraCmd *cobra.Command, args []string, logger *log.Logger) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Exactly one build number is required")
		os.Exit(1)
	}

	buildNumber, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Build number must be an integer:", args[0])
		os.Exit(1)
	}

	workingDir, _ := cobraCmd.Flags().GetString("working-dir")
	if workingDir == "" {
		workingDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get working directory:", err)
			os.Exit(1)
		}
	}

	// Infer project from git remote (same as run-local, run, etc.)
	_, project, inferErr := inferProject(workingDir, logger)

	// Search builds scoped to project when we can infer it
	var searchURL string
	if inferErr == nil && project != "" {
		query := fmt.Sprintf(`"Project" is "%s"`, project)
		searchURL = config.ServerUrl + "/~api/builds?offset=0&count=200&query=" + url.QueryEscape(query)
		logger.Printf("Searching builds for project '%s'\n", project)
	} else {
		// Fallback: search all recent builds (ambiguous when multiple projects share build numbers)
		logger.Printf("Could not infer project (%v), searching all recent builds\n", inferErr)
		searchURL = config.ServerUrl + "/~api/builds?offset=0&count=200"
	}

	body, err := makeAPICallSimple("GET", searchURL, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to query builds:", err)
		os.Exit(1)
	}

	var builds []map[string]interface{}
	if err := json.Unmarshal(body, &builds); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse builds:", err)
		os.Exit(1)
	}

	var buildId int
	found := false
	for _, build := range builds {
		num := int(build["number"].(float64))
		if num == buildNumber {
			buildId = int(build["id"].(float64))
			found = true
			break
		}
	}

	if !found {
		if inferErr == nil && project != "" {
			fmt.Fprintf(os.Stderr, "Build #%d not found in project '%s'\n", buildNumber, project)
		} else {
			fmt.Fprintf(os.Stderr, "Build #%d not found\n", buildNumber)
		}
		os.Exit(1)
	}

	if project != "" {
		fmt.Printf("Streaming log for build #%d (project: %s)...\n", buildNumber, project)
	} else {
		fmt.Printf("Streaming log for build #%d...\n", buildNumber)
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	err = streamBuildLog(buildId, buildNumber, signalChannel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
