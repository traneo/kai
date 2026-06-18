package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <id>",
		Short: "Cancel a running pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			result, err := cl.CancelPipeline(args[0])
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pipeline %s cancelled\n", args[0])
			return nil
		},
	}
}
