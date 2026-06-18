package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show platform status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			s, err := cl.GetStatus()
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(s)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Version    %s\n", s.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Agents     %d (idle: %d, busy: %d)\n", s.Agents, s.IdleAgents, s.BusyAgents)
			fmt.Fprintf(cmd.OutOrStdout(), "Queue      %d missions\n", s.QueueDepth)
			fmt.Fprintf(cmd.OutOrStdout(), "Pipelines  %d total\n", s.Pipelines)
			fmt.Fprintf(cmd.OutOrStdout(), "Uptime     %s\n", s.Uptime)
			return nil
		},
	}
}
