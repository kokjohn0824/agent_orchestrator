// Package cli provides the command-line interface
package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/config"
	orcherrors "github.com/anthropic/agent-orchestrator/internal/errors"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/spf13/cobra"
)

var (
	// Version information (set at build time)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	// Global flags
	cfgFile     string
	dryRun      bool
	verbose     bool
	debug       bool
	quiet       bool
	outputFormat string

	// Global config
	cfg *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "agent-orchestrator",
	Short: i18n.CmdRootShort,
	Long:  i18n.CmdRootLong,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for some commands
		if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf(i18n.ErrLoadConfigFailed, err)
		}

		// Override with flags
		if dryRun {
			cfg.DryRun = true
		}
		if verbose {
			cfg.Verbose = true
		}
		if debug {
			cfg.Debug = true
			cfg.Verbose = true // debug implies verbose
		}
		if quiet {
			cfg.Quiet = true
			cfg.Verbose = false
		}
		if outputFormat != "" {
			cfg.AgentOutputFormat = outputFormat
		}

		return cfg.Validate()
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Persistent flags (available to all commands)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", i18n.FlagConfig)
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, i18n.FlagDryRun)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, i18n.FlagVerbose)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, i18n.FlagDebug)
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, i18n.FlagQuiet)
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", i18n.FlagOutput)

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(retryCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(configCmd)

	// Ticket management commands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(dropCmd)
}

// versionCmd shows version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.CmdVersionShort,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Agent Orchestrator %s\n", Version)
		fmt.Printf("  Commit: %s\n", Commit)
		fmt.Printf("  Built:  %s\n", BuildDate)
	},
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return cfg
}

// CreateAgentCaller creates and configures an agent caller with the current config.
// It sets up DryRun and Verbose modes, and checks if the agent is available.
// Returns an error if the agent is not available (unless in DryRun mode).
func CreateAgentCaller() (*agent.Caller, error) {
	caller := agent.NewCaller(
		cfg.AgentCommand,
		cfg.AgentForce,
		cfg.AgentOutputFormat,
		cfg.LogsDir,
	)
	caller.SetDryRun(cfg.DryRun)
	caller.SetVerbose(cfg.Verbose)
	caller.DisableDetailedLog = cfg.DisableDetailedLog

	if !caller.IsAvailable() && !cfg.DryRun {
		return nil, orcherrors.ErrAgentNotAvailable()
	}

	return caller, nil
}
