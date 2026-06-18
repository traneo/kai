package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pipeline runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			runs, err := cl.ListPipelines()
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(runs)
			}
			if len(runs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no pipelines found")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %-10s  %-6s  %-7s  %-9s  %-9s  %-10s  %s\n",
				"ID", "PROJECT", "STATUS", "STEPS", "PASSED", "FAILED", "BLOCKED", "QUEUED")
			for _, r := range runs {
				fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %-10s  %-6s  %-7d  %-9d  %-9d  %-10t  %t\n",
					r.ID, r.Project, r.Status, r.Steps, r.Passed, r.Failed, r.HasBlocked, r.HasQueued)
			}
			return nil
		},
	}
}
