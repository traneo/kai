package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show <id>",
		Aliases: []string{"get"},
		Short:   "Show pipeline detail",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			p, err := cl.GetPipeline(args[0])
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(p)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID        %s\n", p.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Project   %s\n", p.Project)
			fmt.Fprintf(cmd.OutOrStdout(), "Status    %s\n", p.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Created   %s\n", p.CreatedAt)
			fmt.Fprintf(cmd.OutOrStdout(), "Updated   %s\n", p.UpdatedAt)
			if p.OutputURL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Output    %s (sha: %s)\n", p.OutputURL, p.OutputSHA)
			}
			if p.Error != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Error     %s\n", p.Error)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nSteps:")
			for _, s := range p.Steps {
				statusIcon := statusIcon(s.Status)
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %-18s  %-10s  retry=%d/%d  agent=%s\n",
					statusIcon, s.ID, s.Status, s.Retries, s.MaxRetries, s.AssignedTo)
				if s.Prompt != "" {
					prompt := strings.ReplaceAll(s.Prompt, "\n", " ")
					if len(prompt) > 70 {
						prompt = prompt[:70] + "..."
					}
					fmt.Fprintf(cmd.OutOrStdout(), "     prompt: %s\n", prompt)
				}
				if len(s.DependsOn) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "     depends: %s\n", strings.Join(s.DependsOn, ", "))
				}
				if s.Error != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "     error: %s\n", s.Error)
				}
				for _, g := range s.GateResults {
					fmt.Fprintf(cmd.OutOrStdout(), "     gate %s: %s (%s) — %s\n", g.Gate, g.Status, g.Duration, g.Message)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}
}

func statusIcon(s string) string {
	switch s {
	case "passed":
		return "✓"
	case "failed":
		return "✗"
	case "running":
		return "▶"
	case "blocked":
		return "⊘"
	case "ready", "pending":
		return "○"
	case "cancelled":
		return "–"
	default:
		return "?"
	}
}
