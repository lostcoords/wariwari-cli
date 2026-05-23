package commands

import "github.com/spf13/cobra"

var balanceCmd = &cobra.Command{
	Use:   "balance [GROUP_TOKEN]",
	Short: "Show balances and suggested payments",
	Long: `Show who owes what, derived from the group snapshot.

There is no balances endpoint: this reads GET /v1/groups/:group_token and pulls
out netByCurrency (currency code -> participant token -> signed cents),
suggestedPayments, and the spending totals.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		snapshot, err := groupSnapshot(cmd.Context(), group)
		if err != nil {
			return err
		}
		balances := map[string]any{
			"token":              snapshot["token"],
			"name":               snapshot["name"],
			"currency":           snapshot["currency"],
			"netByCurrency":      snapshot["netByCurrency"],
			"suggestedPayments":  snapshot["suggestedPayments"],
			"spendingByCurrency": snapshot["spendingByCurrency"],
			"totalBaseCents":     snapshot["totalBaseCents"],
			"participants":       snapshotList(snapshot, "participants"),
			"generatedAt":        snapshot["generatedAt"],
		}
		return printOK(balances, "Group balances",
			crumb("snapshot", "wariwari group show "+group, "Read the full group snapshot"))
	},
}

func init() {
	rootCmd.AddCommand(balanceCmd)
}
