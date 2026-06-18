package pipeline

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pipeline",
		Aliases: []string{"pipe", "p"},
		Short:   "Manage pipeline runs",
	}
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewShowCmd())
	cmd.AddCommand(NewCreateCmd())
	cmd.AddCommand(NewCancelCmd())
	cmd.AddCommand(NewApproveCmd())
	cmd.AddCommand(NewLogsCmd())
	return cmd
}
