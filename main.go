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
		var err error
		config, err = LoadConfig()
		if err != nil {
			return err
		}
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
		runLocalJobCommand := RunLocalJobCommand{}
		logger := log.New(os.Stdout, "[RUN-LOCAL] ", log.LstdFlags)
		runLocalJobCommand.Execute(cmd, args, logger)
		return nil
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server",
	Long:  `Start the Model Context Protocol server for tool integration.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
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
		runJobCommand := RunJobCommand{}
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
		checkoutPullRequestCommand := CheckoutPullRequestCommand{}
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
		checkBuildSpecCommand := CheckBuildSpecCommand{}
		logger := log.New(os.Stdout, "[CHECK] ", log.LstdFlags)
		checkBuildSpecCommand.Execute(cmd, args, logger)
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

func init() {
	runLocalJobCmd.Flags().String("working-dir", "", "Specify working directory to run job against (defaults to current directory)")
	runLocalJobCmd.Flags().StringArrayP("param", "p", nil, "Specify job parameters in form of key=value (can be used multiple times)")

	runJobCmd.Flags().String("branch", "", "Specify branch to run job against (either --branch or --tag is required)")
	runJobCmd.Flags().String("tag", "", "Specify tag to run job against (either --branch or --tag is required)")
	runJobCmd.Flags().StringArrayP("param", "p", nil, "Specify job parameters in form of key=value (can be used multiple times)")

	checkoutPullRequestCmd.Flags().String("working-dir", "", "Specify working directory to checkout pull request against (defaults to current directory)")
	checkBuildSpecCmd.Flags().String("working-dir", "", "Specify working directory containing build spec file (defaults to current directory)")
	mcpCmd.Flags().String("log-file", "", "Specify log file path for debug logging")

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

	prsCmd.AddCommand(prsListCmd)
	prsCmd.AddCommand(prsCreateCmd)
	prsCmd.AddCommand(prsMergeCmd)
	prsCmd.AddCommand(prsApproveCmd)
	prsCmd.AddCommand(prsRequestChangesCmd)

	rootCmd.AddCommand(runLocalJobCmd)
	rootCmd.AddCommand(runJobCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(checkoutPullRequestCmd)
	rootCmd.AddCommand(checkBuildSpecCmd)
	rootCmd.AddCommand(prsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
