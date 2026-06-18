package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/cmd/agent"
	"kaiplatform.com/cli/cmd/pipeline"
)

var rootCmd = &cobra.Command{
	Use:   "kaictl",
	Short: "CLI for kai-platform — manage pipelines, agents, and more",
	Long: `kaictl is a command-line tool for interacting with the kai-platform API.

It allows you to create and manage pipeline runs, view agent status,
stream real-time events, and query audit logs — all without coupling
to the platform itself.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("host", "", "Platform host URL (overrides config)")
	rootCmd.PersistentFlags().String("token", "", "Auth token (overrides config)")
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")

	rootCmd.AddCommand(NewLoginCmd())
	rootCmd.AddCommand(NewStatusCmd())
	rootCmd.AddCommand(NewStatsCmd())
	rootCmd.AddCommand(NewAuditCmd())
	rootCmd.AddCommand(NewEventsCmd())
	rootCmd.AddCommand(pipeline.NewCmd())
	rootCmd.AddCommand(agent.NewCmd())
}
