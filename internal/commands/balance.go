package commands

import "github.com/spf13/cobra"

var balanceCmd = &cobra.Command{
	Use:   "balance [GROUP_TOKEN]",
	Short: "Show balances and suggested payments",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/balances", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Group balances")
	},
}

func init() {
	rootCmd.AddCommand(balanceCmd)
}
