package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logs <id> <step>",
		Aliases: []string{"conversation"},
		Short:   "Show step conversation logs",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			entries, err := cl.GetConversation(args[0], args[1], limit)
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no logs for this step")
				return nil
			}
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", e.Source, e.Message)
			}
			return nil
		},
	}
	cmd.Flags().Int("limit", 200, "Max log entries")
	return cmd
}
