package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Query audit log",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			runID, _ := cmd.Flags().GetString("run-id")
			events, err := cl.ListAudit(limit, runID)
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(events)
			}
			for _, e := range events {
				t, _ := time.Parse(time.RFC3339, e.Time)
				fmt.Fprintf(cmd.OutOrStdout(), "%-4d  %s  %-22s  %s  %s\n",
					e.ID, t.Format("15:04:05"), e.Type, e.RunID, e.Message)
			}
			return nil
		},
	}
	cmd.Flags().Int("limit", 50, "Max events")
	cmd.Flags().String("run-id", "", "Filter by run ID")
	return cmd
}
