package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/cmdutil"
)

func NewEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events",
		Short: "Stream real-time platform events (SSE)",
		Long: `Connects to the platform's SSE endpoint and streams events
to stdout until interrupted (Ctrl+C).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cmdutil.Client(cmd)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			isJSON := cmdutil.JSONFlag(cmd)

			fmt.Fprintln(os.Stderr, "connected — waiting for events (Ctrl+C to stop)")

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt)

			go func() {
				<-stop
				fmt.Fprintln(os.Stderr, "\ndisconnected")
				os.Exit(0)
			}()

			return cl.StreamEvents(func(evt map[string]any) {
				if isJSON {
					json.NewEncoder(out).Encode(evt)
				} else {
					typ, _ := evt["type"].(string)
					runID, _ := evt["run_id"].(string)
					msg, _ := evt["message"].(string)
					if msg != "" {
						fmt.Fprintf(out, "[%s] %s: %s\n", typ, runID, msg)
					} else {
						fmt.Fprintf(out, "[%s] %s\n", typ, runID)
					}
				}
			})
		},
	}
}
