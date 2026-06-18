package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewApproveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "approve <id> <step>",
		Aliases: []string{"reject"},
		Short:   "Approve or reject a blocked step",
		Long: `Approve (default) or reject a step that is waiting for human approval.
Use --reject and optionally --message to provide a reason.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			reject, _ := cmd.Flags().GetBool("reject")
			msg, _ := cmd.Flags().GetString("message")
			action := "approve"
			if reject {
				action = "reject"
			}
			result, err := cl.ApproveStep(args[0], args[1], action, msg)
			if err != nil {
				return err
			}
			if cmdutil.JSONFlag(cmd) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			status, _ := result["status"].(string)
			fmt.Fprintf(cmd.OutOrStdout(), "step %s: %s\n", args[1], status)
			return nil
		},
	}
	cmd.Flags().Bool("reject", false, "Reject instead of approve")
	cmd.Flags().StringP("message", "m", "", "Reason for approval/rejection")
	return cmd
}
