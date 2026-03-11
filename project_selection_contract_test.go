package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunCommandRegistersWorkingDirFlag(t *testing.T) {
	if runJobCmd.Flags().Lookup("working-dir") == nil {
		t.Fatalf("run command must register --working-dir flag")
	}
}

func TestProjectFlagRegisteredForSecretsSettingsWebhooks(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "secrets", cmd: secretsCmd},
		{name: "settings", cmd: settingsCmd},
		{name: "webhooks", cmd: webhooksCmd},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cmd.Flags().Lookup("project") == nil {
				t.Fatalf("%s command must register --project flag", tc.name)
			}
		})
	}
}

func TestMCPRunJobUsesEffectiveProjectSelection(t *testing.T) {
	content := mustReadMCPCommand(t)

	if !strings.Contains(content, "runJob(effectiveProject, jobMap)") {
		t.Fatalf("MCP runJob handler must invoke runJob using effectiveProject")
	}
}

func TestMCPQueryHandlerDoesNotSendCompetingProjectSelectors(t *testing.T) {
	content := mustReadMCPCommand(t)
	section := extractFunctionSection(t, content, "func (command *MCPCommand) handleQueryEntitiesTool")

	if strings.Contains(section, "\"project\":        {project},") && strings.Contains(section, "\"currentProject\": {currentProject},") {
		t.Fatalf("MCP query handler must not send both project and currentProject as competing selectors")
	}
}

func TestInferProjectReturnsDeterministicErrorWhenWorkingDirIsNotGitRepo(t *testing.T) {
	workingDir := t.TempDir()
	logger := log.New(io.Discard, "", 0)

	_, _, err := inferProject(workingDir, logger)
	if err == nil {
		t.Fatalf("inferProject must fail when working directory is not a git repository")
	}

	msg := err.Error()
	if !strings.Contains(msg, "working directory is not inside a git repository") {
		t.Fatalf("unexpected error message: %s", msg)
	}
}

func mustReadMCPCommand(t *testing.T) string {
	t.Helper()

	path := filepath.Join("mcp_command.go")
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	return string(bytes)
}

func extractFunctionSection(t *testing.T, content string, signature string) string {
	t.Helper()

	start := strings.Index(content, signature)
	if start == -1 {
		t.Fatalf("failed to find function signature: %s", signature)
	}

	remaining := content[start+len(signature):]
	next := strings.Index(remaining, "\nfunc ")
	if next == -1 {
		return content[start:]
	}

	return content[start : start+len(signature)+next]
}
