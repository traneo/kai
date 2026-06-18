package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show aggregated usage statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			showRuns, _ := cmd.Flags().GetBool("runs")
			s, err := cl.GetStats(showRuns)
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(s)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Agents     %d total, %d idle, %d busy\n",
				s.Agents["total"], s.Agents["idle"], s.Agents["busy"])
			fmt.Fprintf(cmd.OutOrStdout(), "Queue      %d\n", s.QueueDepth)
			fmt.Fprintf(cmd.OutOrStdout(), "Pipelines  %d\n", s.Pipelines)
			fmt.Fprintf(cmd.OutOrStdout(), "Steps      %d\n", s.Steps)
			if t, ok := s.Tokens["total"]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "Tokens     %.0f total\n", toFloat64(t))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Duration   %d ms\n", s.DurationMs)
			if showRuns && len(s.Runs) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nPer-run tokens:")
				for _, r := range s.Runs {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s): %d\n", r.RunID, r.Project, r.TotalTokens)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("runs", false, "Include per-run breakdown")
	return cmd
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}
