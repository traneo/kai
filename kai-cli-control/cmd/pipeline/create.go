package pipeline

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "create <file.yaml>",
		Aliases: []string{"new", "run"},
		Short:   "Create a pipeline run from a YAML file",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			yaml, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			result, err := cl.CreatePipeline(string(yaml))
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			id, _ := result["id"].(string)
			status, _ := result["status"].(string)
			project, _ := result["project"].(string)
			fmt.Fprintf(cmd.OutOrStdout(), "created pipeline %s (%s) — status: %s\n", id, project, status)
			return nil
		},
	}
}
