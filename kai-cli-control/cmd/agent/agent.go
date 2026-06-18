package agent

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"agents", "a"},
		Short:   "Manage platform agents",
	}
	cmd.AddCommand(NewListCmd())
	return cmd
}
