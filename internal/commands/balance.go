package commands

import "github.com/spf13/cobra"

var balanceCmd = &cobra.Command{
	Use:   "balance [GROUP_TOKEN]",
	Short: "Show balances and suggested payments",
	Long: `Show who owes what: netByCurrency (currency code -> participant token ->
signed cents, positive means owed), suggestedPayments, and the spending
totals, taken from the group snapshot.

Settle a suggested payment with "wariwari payment create"; see how one
member's balance adds up with "wariwari participant balance".`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
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
