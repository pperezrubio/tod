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

	// Resolve project: --project flag takes priority, then infer from git remote
	projectPath := resolveProjectForBuilds(cobraCmd, logger)

	// Search builds scoped to the resolved project
	query := fmt.Sprintf(`"Project" is "%s"`, projectPath)
	searchURL := config.ServerUrl + "/~api/builds?offset=0&count=200&query=" + url.QueryEscape(query)

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
		fmt.Fprintf(os.Stderr, "Build #%d not found in project '%s'\n", buildNumber, projectPath)
		os.Exit(1)
	}

	fmt.Printf("Streaming log for build #%d (project: %s)...\n", buildNumber, projectPath)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	err = streamBuildLog(buildId, buildNumber, signalChannel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
