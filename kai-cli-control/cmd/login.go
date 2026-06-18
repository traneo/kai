package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/config"
)

func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <host> <token>",
		Short: "Save platform connection info",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Config{
				Host:  args[0],
				Token: args[1],
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("saved connection to ~/.kaictl/config.json")
			return nil
		},
	}
}
