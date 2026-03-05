package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const version = "2.1.1"

type CompatibleVersions struct {
	MinVersion string `json:"minVersion"`
	MaxVersion string `json:"maxVersion"`
}

var config *Config

var rootCmd = &cobra.Command{
	Use:   "tod",
	Short: "TOD (TheOneDev) is a command line tool for OneDev 13+",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration from file
		var err error
		config, err = LoadConfig()
		if err != nil {
			return err
		}

		// Configuration is loaded from file only

		if err := config.Validate(); err != nil {
			return err
		}

		if err := checkVersion(config.ServerUrl, config.AccessToken); err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		return nil
	},
}

var runLocalJobCmd = &cobra.Command{
	Use:   "run-local [job-name]",
	Short: "Run a CI/CD job against local changes",
	Long: `Run a CI/CD job against your local changes without committing/pushing.
This command stashes your local changes, pushes them to a temporal ref,
and streams the job execution logs back to your terminal.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Run local job command
		runLocalJobCommand := RunLocalJobCommand{}
		// Create a logger that prints to stdout
		logger := log.New(os.Stdout, "[RUN-LOCAL] ", log.LstdFlags)
		runLocalJobCommand.Execute(cmd, args, logger)
		return nil
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Long:  `List all projects on the OneDev server.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectsCommand := ProjectsCommand{}
		logger := log.New(os.Stdout, "[PROJECTS] ", log.LstdFlags)
		projectsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var buildsCmd = &cobra.Command{
	Use:   "builds",
	Short: "List recent builds",
	Long:  `List recent builds with their status, job name, and commit hash.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		buildsCommand := BuildsCommand{}
		logger := log.New(os.Stdout, "[BUILDS] ", log.LstdFlags)
		buildsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List job executors",
	Long:  `List configured job executors (agents) on the OneDev server.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		agentsCommand := AgentsCommand{}
		logger := log.New(os.Stdout, "[AGENTS] ", log.LstdFlags)
		agentsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs [build-number]",
	Short: "Stream build logs",
	Long:  `Stream build logs in real-time for a given build number.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logsCommand := LogsCommand{}
		logger := log.New(os.Stdout, "[LOGS] ", log.LstdFlags)
		logsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "List job secrets for a project",
	Long:  `List job secrets configured in a project's build settings.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		secretsCommand := SecretsCommand{}
		logger := log.New(os.Stdout, "[SECRETS] ", log.LstdFlags)
		secretsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show project settings",
	Long:  `Show project settings sections or a specific section in detail.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		settingsCommand := SettingsCommand{}
		logger := log.New(os.Stdout, "[SETTINGS] ", log.LstdFlags)
		settingsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List users",
	Long:  `List users on the OneDev server.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		usersCommand := UsersCommand{}
		logger := log.New(os.Stdout, "[USERS] ", log.LstdFlags)
		usersCommand.Execute(cmd, args, logger)
		return nil
	},
}

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "List webhooks for a project",
	Long:  `List webhooks configured in a project's settings.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		webhooksCommand := WebhooksCommand{}
		logger := log.New(os.Stdout, "[WEBHOOKS] ", log.LstdFlags)
		webhooksCommand.Execute(cmd, args, logger)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Show, get, or set configuration values.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip global config validation for config command
		// config set needs to work without a valid config
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all config values",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ConfigShowCommand(cmd, args)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ConfigGetCommand(cmd, args)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key=value]",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ConfigSetCommand(cmd, args)
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ConfigPathCommand(cmd, args)
	},
}

var createProjectCmd = &cobra.Command{
	Use:   "create-project [name]",
	Short: "Create a new project",
	Long:  "Create a new project on the OneDev server.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		createProjectCommand := CreateProjectCommand{}
		logger := log.New(os.Stdout, "[CREATE-PROJECT] ", log.LstdFlags)
		createProjectCommand.Execute(cmd, args, logger)
		return nil
	},
}

var prsCmd = &cobra.Command{
	Use:   "prs",
	Short: "Manage pull requests",
	Long:  `List, create, and merge pull requests in a OneDev project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prsCommand := PrsCommand{}
		logger := log.New(os.Stdout, "[PRS] ", log.LstdFlags)
		prsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var prsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Long:  `List pull requests for a OneDev project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prsCommand := PrsCommand{}
		logger := log.New(os.Stdout, "[PRS] ", log.LstdFlags)
		prsCommand.Execute(cmd, args, logger)
		return nil
	},
}

var prsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pull request",
	Long:  `Create a new pull request in a OneDev project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prsCommand := PrsCommand{}
		logger := log.New(os.Stdout, "[PRS] ", log.LstdFlags)
		prsCommand.ExecuteCreate(cmd, args, logger)
		return nil
	},
}

var prsMergeCmd = &cobra.Command{
	Use:   "merge [pull-request-number]",
	Short: "Merge a pull request",
	Long:  `Merge a pull request in a OneDev project.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prsCommand := PrsCommand{}
		logger := log.New(os.Stdout, "[PRS] ", log.LstdFlags)
		prsCommand.ExecuteMerge(cmd, args, logger)
		return nil
	},
}

var prsApproveCmd = &cobra.Command{
	Use:   "approve <pr-number>",
	Short: "Approve a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := PrsApproveCommand{}
		logger := log.New(os.Stdout, "[PRS] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

var prsRequestChangesCmd = &cobra.Command{
	Use:   "request-changes <pr-number>",
	Short: "Request changes on a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := PrsRequestChangesCommand{}
		logger := log.New(os.Stdout, "[PRS] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server",
	Long:  `Start the Model Context Protocol server for tool integration.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip global config validation for MCP command
		// MCP will handle its own config validation after logger initialization
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// MCP command
		mcpCommand := MCPCommand{}
		mcpCommand.Execute(cmd, args)
		return nil
	},
}

var runJobCmd = &cobra.Command{
	Use:   "run [job-name]",
	Short: "Run a CI/CD job against a specific branch or tag",
	Long: `Run a CI/CD job against a specific branch or tag in the repository.
Either --branch or --tag must be specified, but not both.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Run job command
		runJobCommand := RunJobCommand{}
		// Create a logger that prints to stdout
		logger := log.New(os.Stdout, "[RUN] ", log.LstdFlags)
		runJobCommand.Execute(cmd, args, logger)
		return nil
	},
}

var checkoutPullRequestCmd = &cobra.Command{
	Use:   "checkout [pull-request-reference]",
	Short: "Checkout a pull request",
	Long:  `Checkout a pull request by its reference.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Checkout command
		checkoutPullRequestCommand := CheckoutPullRequestCommand{}
		// Create a logger that prints to stdout
		logger := log.New(os.Stdout, "[CHECKOUT] ", log.LstdFlags)
		checkoutPullRequestCommand.Execute(cmd, args, logger)
		return nil
	},
}

var checkBuildSpecCmd = &cobra.Command{
	Use:   "check-build-spec",
	Short: "Check build spec",
	Long:  "Check build spec for its validity, as well as updating it to latest version if needed",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check build spec command
		checkBuildSpecCommand := CheckBuildSpecCommand{}
		// Create a logger that prints to stdout
		logger := log.New(os.Stdout, "[CHECK] ", log.LstdFlags)
		checkBuildSpecCommand.Execute(cmd, args, logger)
		return nil
	},
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage project issues",
	Long:  `List, create, edit, and close OneDev project issues.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: list issues
		issuesListCommand := IssuesListCommand{}
		logger := log.New(os.Stdout, "[ISSUES] ", log.LstdFlags)
		issuesListCommand.Execute(cmd, args, logger)
		return nil
	},
}

var issuesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues for a project",
	Long:  `List issues for a project with optional filtering by state and query.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issuesListCommand := IssuesListCommand{}
		logger := log.New(os.Stdout, "[ISSUES] ", log.LstdFlags)
		issuesListCommand.Execute(cmd, args, logger)
		return nil
	},
}

var issuesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Long:  `Create a new issue in a project.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issuesCreateCommand := IssuesCreateCommand{}
		logger := log.New(os.Stdout, "[ISSUES] ", log.LstdFlags)
		issuesCreateCommand.Execute(cmd, args, logger)
		return nil
	},
}

var issuesEditCmd = &cobra.Command{
	Use:   "edit <number>",
	Short: "Edit an existing issue",
	Long:  `Edit an existing issue's title or description.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issuesEditCommand := IssuesEditCommand{}
		logger := log.New(os.Stdout, "[ISSUES] ", log.LstdFlags)
		issuesEditCommand.Execute(cmd, args, logger)
		return nil
	},
}

var issuesCloseCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close an issue",
	Long:  `Close an issue by changing its state.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issuesCloseCommand := IssuesCloseCommand{}
		logger := log.New(os.Stdout, "[ISSUES] ", log.LstdFlags)
		issuesCloseCommand.Execute(cmd, args, logger)
		return nil
	},
}

var issuesCommentsCmd = &cobra.Command{
	Use:   "comments <number>",
	Short: "List comments for an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := IssuesCommentsCommand{}
		logger := log.New(os.Stdout, "[ISSUES] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

var issuesCommentCmd = &cobra.Command{
	Use:   "comment <number> <body>",
	Short: "Add a comment to an issue",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := IssuesCommentCommand{}
		logger := log.New(os.Stdout, "[ISSUES] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

var artifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "Manage build artifacts",
	Long:  `List and download OneDev build artifacts.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		artifactsListCommand := ArtifactsListCommand{}
		logger := log.New(os.Stdout, "[ARTIFACTS] ", log.LstdFlags)
		artifactsListCommand.Execute(cmd, args, logger)
		return nil
	},
}

var artifactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List artifacts for a build",
	Long:  `List all artifacts produced by a specific build.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		artifactsListCommand := ArtifactsListCommand{}
		logger := log.New(os.Stdout, "[ARTIFACTS] ", log.LstdFlags)
		artifactsListCommand.Execute(cmd, args, logger)
		return nil
	},
}

var artifactsDownloadCmd = &cobra.Command{
	Use:   "download <path>",
	Short: "Download an artifact from a build",
	Long:  `Download a specific artifact file from a build by its path.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		artifactsDownloadCommand := ArtifactsDownloadCommand{}
		logger := log.New(os.Stdout, "[ARTIFACTS] ", log.LstdFlags)
		artifactsDownloadCommand.Execute(cmd, args, logger)
		return nil
	},
}

var branchesCmd = &cobra.Command{
	Use:   "branches",
	Short: "Manage repository branches",
	Long:  `List, create, and delete OneDev repository branches.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		command := BranchesCommand{}
		logger := log.New(os.Stdout, "[BRANCHES] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

var branchesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List branches for a project",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		command := BranchesListCommand{}
		logger := log.New(os.Stdout, "[BRANCHES] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

var branchesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := BranchesCreateCommand{}
		logger := log.New(os.Stdout, "[BRANCHES] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

var branchesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := BranchesDeleteCommand{}
		logger := log.New(os.Stdout, "[BRANCHES] ", log.LstdFlags)
		command.Execute(cmd, args, logger)
		return nil
	},
}

func init() {
	// Run-local command specific flags
	runLocalJobCmd.Flags().String("working-dir", "", "Specify working directory to run job against (defaults to current directory)")
	runLocalJobCmd.Flags().StringArrayP("param", "p", nil, "Specify job parameters in form of key=value (can be used multiple times)")

	// Run job command specific flags
	runJobCmd.Flags().String("branch", "", "Specify branch to run job against (either --branch or --tag is required)")
	runJobCmd.Flags().String("tag", "", "Specify tag to run job against (either --branch or --tag is required)")
	runJobCmd.Flags().StringArrayP("param", "p", nil, "Specify job parameters in form of key=value (can be used multiple times)")

	// Checkout command specific flags
	checkoutPullRequestCmd.Flags().String("working-dir", "", "Specify working directory to checkout pull request against (defaults to current directory)")

	// Check build spec command specific flags
	checkBuildSpecCmd.Flags().String("working-dir", "", "Specify working directory containing build spec file (defaults to current directory)")

	// MCP command specific flags
	mcpCmd.Flags().String("log-file", "", "Specify log file path for debug logging")

	// Projects command flags
	projectsCmd.Flags().IntP("count", "n", 50, "Number of projects to show")

	// Builds command flags
	buildsCmd.Flags().IntP("count", "n", 10, "Number of builds to show")
	buildsCmd.Flags().StringP("query", "q", "", "OneDev build query (e.g. '\"Job\" is \"Release\"')")

	// Config subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)

	// Issues command flags (shared for default list action)
	issuesCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	issuesCmd.Flags().IntP("count", "n", 50, "Maximum number of issues to return")
	issuesCmd.Flags().StringP("query", "q", "", "OneDev issue query (overrides --state filter)")
	issuesCmd.Flags().StringP("state", "s", "open", "Filter by state: open, closed, all")

	// Issues list command flags
	issuesListCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	issuesListCmd.Flags().IntP("count", "n", 50, "Maximum number of issues to return")
	issuesListCmd.Flags().StringP("query", "q", "", "OneDev issue query (overrides --state filter)")
	issuesListCmd.Flags().StringP("state", "s", "open", "Filter by state: open, closed, all")

	// Issues create command flags
	issuesCreateCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	issuesCreateCmd.Flags().StringP("title", "t", "", "Issue title (required)")
	issuesCreateCmd.Flags().StringP("description", "d", "", "Issue description")

	// Issues edit command flags
	issuesEditCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	issuesEditCmd.Flags().StringP("title", "t", "", "New title for the issue")
	issuesEditCmd.Flags().StringP("description", "d", "", "New description for the issue")

	// Issues close command flags
	issuesCloseCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")

	// Issues comments command flags
	issuesCommentsCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")

	// Issues comment command flags
	issuesCommentCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")

	// Add subcommands to issuesCmd
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesEditCmd)
	issuesCmd.AddCommand(issuesCloseCmd)
	issuesCmd.AddCommand(issuesCommentsCmd)
	issuesCmd.AddCommand(issuesCommentCmd)

	// PRS command flags (default list action)
	prsCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	prsCmd.Flags().IntP("count", "n", 20, "Maximum number of pull requests to return")
	prsCmd.Flags().StringP("query", "q", "", "OneDev PR query (overrides --status filter)")
	prsCmd.Flags().StringP("status", "s", "open", "Filter by status: open, merged, discarded, all")

	// PRS list command flags
	prsListCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	prsListCmd.Flags().IntP("count", "n", 20, "Maximum number of pull requests to return")
	prsListCmd.Flags().StringP("query", "q", "", "OneDev PR query (overrides --status filter)")
	prsListCmd.Flags().StringP("status", "s", "open", "Filter by status: open, merged, discarded, all")

	// PRS create command flags
	prsCreateCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	prsCreateCmd.Flags().String("title", "", "Pull request title (required)")
	prsCreateCmd.Flags().String("source", "", "Source branch (required)")
	prsCreateCmd.Flags().String("target", "main", "Target branch (default: main)")
	prsCreateCmd.Flags().String("description", "", "Pull request description")

	// PRS merge command flags
	prsMergeCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	prsMergeCmd.Flags().String("strategy", "merge-commit", "Merge strategy: merge-commit, squash-merge, rebase-merge")
	prsMergeCmd.Flags().Bool("delete-branch", true, "Delete source branch after merge")

	// PRS approve command flags
	prsApproveCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")

	// PRS request-changes command flags
	prsRequestChangesCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")

	// Add subcommands to prsCmd
	prsCmd.AddCommand(prsListCmd)
	prsCmd.AddCommand(prsCreateCmd)
	prsCmd.AddCommand(prsMergeCmd)
	prsCmd.AddCommand(prsApproveCmd)
	prsCmd.AddCommand(prsRequestChangesCmd)

	// Artifacts command flags (shared for default list action)
	artifactsCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	artifactsCmd.Flags().StringP("build", "b", "", "Build number (required)")

	// Artifacts list command flags
	artifactsListCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	artifactsListCmd.Flags().StringP("build", "b", "", "Build number (required)")

	// Artifacts download command flags
	artifactsDownloadCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	artifactsDownloadCmd.Flags().StringP("build", "b", "", "Build number (required)")
	artifactsDownloadCmd.Flags().StringP("output", "o", "", "Output file path (default: artifact filename)")

	// Add subcommands to artifactsCmd
	artifactsCmd.AddCommand(artifactsListCmd)
	artifactsCmd.AddCommand(artifactsDownloadCmd)

	// Add commands to root
	rootCmd.AddCommand(runLocalJobCmd)
	rootCmd.AddCommand(runJobCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(checkoutPullRequestCmd)
	rootCmd.AddCommand(checkBuildSpecCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(buildsCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(settingsCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(webhooksCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(createProjectCmd)
	rootCmd.AddCommand(issuesCmd)
	rootCmd.AddCommand(prsCmd)
	rootCmd.AddCommand(artifactsCmd)

	// Branches command flags
	branchesCmd.Flags().StringP("project", "p", "", "Project path (inferred from git remote if not specified)")
	branchesListCmd.Flags().StringP("project", "p", "", "Project path")
	branchesCreateCmd.Flags().StringP("project", "p", "", "Project path")
	branchesCreateCmd.Flags().String("from", "main", "Base revision (branch, tag, or commit hash)")
	branchesDeleteCmd.Flags().StringP("project", "p", "", "Project path")

	// Add subcommands to branchesCmd
	branchesCmd.AddCommand(branchesListCmd)
	branchesCmd.AddCommand(branchesCreateCmd)
	branchesCmd.AddCommand(branchesDeleteCmd)

	// Add branches to root
	rootCmd.AddCommand(branchesCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
