package agent

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
		Short:   "List connected agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			agents, err := cl.ListAgents()
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(agents)
			}
			if len(agents) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no agents connected")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %-12s  %-10s  missions=%-4s  uptime=%-10s\n",
				"ID", "STATE", "ADDR", "MISSIONS", "UPTIME")
			for _, a := range agents {
				healthy := "ok"
				if !a.Healthy {
					healthy = "UNHEALTHY"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-20s  %-12s  %-10s  %-12d  %s\n",
					a.ID, a.State+"("+healthy+")", a.Addr, a.MissionsCompleted, fmtDuration(a.UptimeMs))
			}
			return nil
		},
	}
}

func fmtDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := ms / 1000
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s = s % 60
	return fmt.Sprintf("%dm%ds", m, s)
}
