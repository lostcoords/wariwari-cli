package commands

import "github.com/spf13/cobra"

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/me", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Authenticated user")
	},
}

func init() {
	rootCmd.AddCommand(meCmd)
}
