package commands

import "github.com/spf13/cobra"

var transactionCmd = &cobra.Command{
	Use:     "transaction",
	Aliases: []string{"expense"},
	Short:   "Manage transactions",
}

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
  --category       group category token (from the snapshot's categories)
  --participant    participant token; matches transactions that participant
                   paid for or owes a share of`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		query := pagingQuery(transactionTake, transactionCursor)
		query["from"] = transactionFrom
		query["to"] = transactionTo
		query["categoryToken"] = transactionCategory
		query["participantToken"] = transactionParticipant

		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions", query)
		if err != nil {
			return err
		}
		nextCommand := "wariwari transaction list " + group
		nextCommand = commandFlag(nextCommand, "from", transactionFrom)
		nextCommand = commandFlag(nextCommand, "to", transactionTo)
		nextCommand = commandFlag(nextCommand, "category", transactionCategory)
		nextCommand = commandFlag(nextCommand, "participant", transactionParticipant)
		nextCommand = commandTakeFlag(nextCommand, transactionTake)
		return printOK(data, "Transactions",
			nextPageCrumb(data, nextCommand+" --cursor"))
	},
}

var transactionShowCmd = &cobra.Command{
	Use:     "show TRANSACTION_TOKEN [GROUP_TOKEN]",
	Aliases: []string{"view"},
	Short:   "Show a transaction",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions/"+args[0], nil)
		if err != nil {
			return err
		}
		return printOK(data, "Transaction",
			crumb("update", "wariwari transaction update "+args[0]+" "+group, "Edit it; pass this record's lockVersion as --lock-version"))
	},
}

var transactionCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Create a transaction from JSON",
	Long: `Create an expense and get it back. Developer API keys are supported.

Required: kind ("expense"), description, amountCents, currency, splitType,
date, paidByParticipantToken, splits. splitType is one of equal, percentage,
shares, amount; with equal the splits may omit amountCents and the server
divides evenly. Optional: categoryToken (a group category from the snapshot's
categories; there is no free-text category), timeOfDay (HH:MM), and
balanceCurrency (send the transaction currency to keep it in its own balance
bucket instead of the group currency).

A settlement is a transaction of kind "payment" with its own fields; record
one with "wariwari payment create".

Reusing a client-generated token returns 409 DUPLICATE_TOKEN; retry a create
safely with --idempotency-key and the same body.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, err := commandData()
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/transactions", write(body))
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
			"token":                  "9AHTHuAGT7U2CpSfktrfUAYA",
			"kind":                   "expense",
			"description":            "Team dinner",
			"categoryToken":          "4QfYk8nE2pRtVwXz7aBcDeFg",
			"amountCents":            12000,
			"currency":               "JPY",
			"balanceCurrency":        "JPY",
			"splitType":              "equal",
			"date":                   "2026-05-23",
			"timeOfDay":              "19:30",
			"paidByParticipantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"splits": []map[string]any{
				{"participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj"},
				{"participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa"},
				{"participantToken": "9nQPmS5rRYuMcKS6DdyU7zXb"},
			},
		}, "Transaction create example")
	},
}

var transactionExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"kind":                   "expense",
			"description":            "Team dinner at ramen shop",
			"categoryToken":          nil,
			"amountCents":            13200,
			"splitType":              "amount",
			"timeOfDay":              "20:00",
			"paidByParticipantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"splits": []map[string]any{
				{"participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj", "amountCents": 6600},
				{"participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa", "amountCents": 6600},
			},
		}, "Transaction update example")
	},
}

var transactionUpdateCmd = &cobra.Command{
	Use:   "update TRANSACTION_TOKEN [GROUP_TOKEN]",
	Short: "Update a transaction from JSON",
	Long: `Update an expense and get it back. kind is required and cannot change:
send "kind": "expense" plus only the fields that change. Sending splits
replaces all existing splits; send "categoryToken": null to clear the
category. Update a settlement with "wariwari payment update".` + guardedUpdateHelp,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		body, err := commandData()
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, body, transactionRecord(group, args[0]))
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/transactions/"+args[0], w)
		if err != nil {
			return err
		}
		return printOK(data, "Transaction updated")
	},
}

var transactionDeleteCmd = &cobra.Command{
	Use:   "delete TRANSACTION_TOKEN [GROUP_TOKEN]",
	Short: "Soft-delete a transaction",
	Long:  guardedDeleteHelp,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, nil, transactionRecord(group, args[0]))
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/transactions/"+args[0], w); err != nil {
			return err
		}
		return printOK(map[string]any{"token": args[0], "deleted": true}, "Transaction deleted")
	},
}

var transactionActivityCmd = &cobra.Command{
	Use:   "activity TRANSACTION_TOKEN [GROUP_TOKEN]",
	Short: "Show transaction activity",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions/"+args[0]+"/activity", pagingQuery(transactionTake, transactionCursor))
		if err != nil {
			return err
		}
		nextCommand := commandTakeFlag("wariwari transaction activity "+args[0]+" "+group, transactionTake)
		return printOK(data, "Transaction activity",
			nextPageCrumb(data, nextCommand+" --cursor"))
	},
}

func init() {
	addLockVersionFlag(transactionUpdateCmd, transactionDeleteCmd)
	addDataFlags(transactionCreateCmd, transactionUpdateCmd, transactionBulkEditCmd, transactionBulkDeleteCmd)
	addIdempotencyFlag(transactionCreateCmd, transactionUpdateCmd, transactionDeleteCmd, transactionBulkEditCmd, transactionBulkDeleteCmd)
	rootCmd.AddCommand(transactionCmd)
	transactionCmd.AddCommand(transactionListCmd, transactionShowCmd, transactionCreateCmd, transactionUpdateCmd, transactionDeleteCmd, transactionActivityCmd, transactionBulkEditCmd, transactionBulkDeleteCmd, transactionExampleCmd)
	transactionExampleCmd.AddCommand(transactionExampleCreateCmd, transactionExampleUpdateCmd, transactionExampleBulkEditCmd)
	transactionListCmd.Flags().StringVar(&transactionCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	transactionListCmd.Flags().IntVar(&transactionTake, "take", 0, "Results per page (max 100)")
	transactionListCmd.Flags().StringVar(&transactionFrom, "from", "", "Only transactions dated on or after this ISO date")
	transactionListCmd.Flags().StringVar(&transactionTo, "to", "", "Only transactions dated on or before this ISO date")
	transactionListCmd.Flags().StringVar(&transactionCategory, "category", "", "Only transactions filed under this group category token")
	transactionListCmd.Flags().StringVar(&transactionParticipant, "participant", "", "Only transactions this participant paid or owes a share of")
	transactionActivityCmd.Flags().StringVar(&transactionCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	transactionActivityCmd.Flags().IntVar(&transactionTake, "take", 0, "Results per page (max 100)")
}
