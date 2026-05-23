package commands

import "github.com/spf13/cobra"

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage WariWari API keys",
}

var tokenName string
var tokenCursor string

var tokenListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/api_keys", queryWithPaging(tokenCursor))
		if err != nil {
			return err
		}
		return printOK(data, "API keys")
	},
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		body := map[string]any{"name": tokenName}
		data, err := api.Post(cmd.Context(), "/api_keys", body, "")
		if err != nil {
			return err
		}
		return printOK(data, "API key created")
	},
}

var tokenDeleteCmd = &cobra.Command{
	Use:     "delete ID",
	Aliases: []string{"rm"},
	Short:   "Revoke an API key",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/api_keys/"+args[0])
		if err != nil {
			return err
		}
		return printOK(data, "API key revoked")
	},
}

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenListCmd, tokenCreateCmd, tokenDeleteCmd)
	tokenListCmd.Flags().StringVar(&tokenCursor, "cursor", "", "Pagination cursor")
	tokenCreateCmd.Flags().StringVar(&tokenName, "name", "CLI", "API key name")
}
