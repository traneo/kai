package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"

	"kaiplatform.com/cli/internal/client"
	"kaiplatform.com/cli/internal/config"
)

func Client(cmd *cobra.Command) (*client.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	host, _ := cmd.Flags().GetString("host")
	if host == "" {
		host = cfg.Host
	}
	token, _ := cmd.Flags().GetString("token")
	if token == "" {
		token = cfg.Token
	}
	if host == "" {
		return nil, fmt.Errorf("no host configured; run 'kaictl login' or set --host")
	}
	return client.New(host, token), nil
}

func JSONFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}
