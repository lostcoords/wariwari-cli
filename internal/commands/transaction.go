package commands

import "github.com/spf13/cobra"

var transactionCmd = &cobra.Command{
	Use:   "transaction",
	Short: "Manage transactions",
}

var transactionBody string
var transactionBodyFile string
var transactionCursor string
var transactionTake int
var transactionFrom string
var transactionTo string
var transactionCategory string
var transactionParticipant string

var transactionListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List a group's transactions",
	Long: `List a group's transactions, newest first, keyset paginated.

Filters combine and narrow rather than assert: a well-formed token that matches
nothing returns an empty page, while a malformed date is 400 INVALID_REQUEST.

  --from/--to      ISO dates (2026-01-31), inclusive
  --category       group category token (from the snapshot's groupCategories)
  --participant    participant token; matches transactions that participant
                   paid for or owes a share of`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		query := pagingQuery(transactionTake, transactionCursor)
		query["from"] = transactionFrom
		query["to"] = transactionTo
		query["category"] = transactionCategory
		query["participant"] = transactionParticipant

		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions", query)
		if err != nil {
			return err
		}
		return printOK(data, "Transactions",
			nextPageCrumb(data, "wariwari transaction list "+group+" --cursor"))
	},
}

var transactionShowCmd = &cobra.Command{
	Use:     "show TRANSACTION_TOKEN [GROUP_TOKEN]",
	Aliases: []string{"view"},
	Short:   "Show a transaction",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions/"+args[0], nil)
		if err != nil {
			return err
		}
		return printOK(data, "Transaction",
			crumb("update", "wariwari transaction update "+args[0]+" "+group, "Edit it — send this record's lockVersion"))
	},
}

var transactionCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Create a transaction from JSON",
	Long: `Create an expense or settlement payment. Returns the refreshed group
snapshot.

Required: description, amountCents, currency, splitType, date, paidByToken,
splits. splitType is one of equal, percentage, shares, amount.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		body, err := transactionCommandBody()
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/transactions", body, "")
		if err != nil {
			return err
		}
		return printOK(data, "Transaction created")
	},
}

var transactionExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Print example request payloads",
}

var transactionExampleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Example payload for creating a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"kind":        "expense",
			"description": "Team dinner",
			"amountCents": 12000,
			"currency":    "JPY",
			"splitType":   "equal",
			"date":        "2026-05-23",
			"paidByToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"splits": []map[string]any{
				{"participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj", "amountCents": 4000},
				{"participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa", "amountCents": 4000},
				{"participantToken": "9nQPmS5rRYuMcKS6DdyU7zXb", "amountCents": 4000},
			},
		}, "Transaction create example")
	},
}

var transactionExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"description": "Team dinner at ramen shop",
			"amountCents": 13200,
			"splitType":   "equal",
			"paidByToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"splits": []map[string]any{
				{"participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj", "amountCents": 4400},
				{"participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa", "amountCents": 4400},
				{"participantToken": "9nQPmS5rRYuMcKS6DdyU7zXb", "amountCents": 4400},
			},
			"lockVersion": 3,
		}, "Transaction update example")
	},
}

var transactionUpdateCmd = &cobra.Command{
	Use:   "update TRANSACTION_TOKEN [GROUP_TOKEN]",
	Short: "Update a transaction from JSON",
	Long: `Update a transaction. lockVersion is required — send the value from the
current record. A stale value returns 409 CONFLICT with the fresh transaction
under "current". Sending splits replaces all existing splits.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		body, err := transactionCommandBody()
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/transactions/"+args[0], body)
		if err != nil {
			return err
		}
		return printOK(data, "Transaction updated")
	},
}

var transactionDeleteCmd = &cobra.Command{
	Use:   "delete TRANSACTION_TOKEN [GROUP_TOKEN]",
	Short: "Soft-delete a transaction",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/groups/"+group+"/transactions/"+args[0])
		if err != nil {
			return err
		}
		return printOK(data, "Transaction deleted")
	},
}

var transactionActivityCmd = &cobra.Command{
	Use:   "activity TRANSACTION_TOKEN [GROUP_TOKEN]",
	Short: "Show transaction activity",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions/"+args[0]+"/activity", pagingQuery(transactionTake, transactionCursor))
		if err != nil {
			return err
		}
		return printOK(data, "Transaction activity",
			nextPageCrumb(data, "wariwari transaction activity "+args[0]+" "+group+" --cursor"))
	},
}

func transactionCommandBody() (map[string]any, error) {
	if transactionBodyFile != "" {
		return parseObjectFile(transactionBodyFile)
	}
	return parseObject(transactionBody)
}

func init() {
	rootCmd.AddCommand(transactionCmd)
	transactionCmd.AddCommand(transactionListCmd, transactionShowCmd, transactionCreateCmd, transactionUpdateCmd, transactionDeleteCmd, transactionActivityCmd, transactionExampleCmd)
	transactionExampleCmd.AddCommand(transactionExampleCreateCmd, transactionExampleUpdateCmd)
	transactionListCmd.Flags().StringVar(&transactionCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	transactionListCmd.Flags().IntVar(&transactionTake, "take", 0, "Results per page (max 100)")
	transactionListCmd.Flags().StringVar(&transactionFrom, "from", "", "Only transactions dated on or after this ISO date")
	transactionListCmd.Flags().StringVar(&transactionTo, "to", "", "Only transactions dated on or before this ISO date")
	transactionListCmd.Flags().StringVar(&transactionCategory, "category", "", "Only transactions filed under this group category token")
	transactionListCmd.Flags().StringVar(&transactionParticipant, "participant", "", "Only transactions this participant paid or owes a share of")
	transactionActivityCmd.Flags().StringVar(&transactionCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	transactionActivityCmd.Flags().IntVar(&transactionTake, "take", 0, "Results per page (max 100)")
	transactionCreateCmd.Flags().StringVar(&transactionBody, "data", "", "JSON request body")
	transactionCreateCmd.Flags().StringVar(&transactionBodyFile, "data-file", "", "Read JSON request body from file")
	transactionUpdateCmd.Flags().StringVar(&transactionBody, "data", "", "JSON request body")
	transactionUpdateCmd.Flags().StringVar(&transactionBodyFile, "data-file", "", "Read JSON request body from file")
}
