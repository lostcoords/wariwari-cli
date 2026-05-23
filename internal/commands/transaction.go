package commands

import "github.com/spf13/cobra"

var transactionCmd = &cobra.Command{
	Use:   "transaction",
	Short: "Manage transactions",
}

var transactionBody string
var transactionBodyFile string
var transactionCursor string

var transactionListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List transactions",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions", queryWithPaging(transactionCursor))
		if err != nil {
			return err
		}
		return printOK(data, "Transactions")
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
		return printOK(data, "Transaction")
	},
}

var transactionUpsertCmd = &cobra.Command{
	Use:   "upsert TRANSACTION_TOKEN [GROUP_TOKEN]",
	Short: "Create or update a transaction from JSON",
	Args:  cobra.RangeArgs(1, 2),
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
		return printOK(data, "Transaction saved")
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
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/transactions/"+args[0]+"/activity", queryWithPaging(transactionCursor))
		if err != nil {
			return err
		}
		return printOK(data, "Transaction activity")
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
	transactionCmd.AddCommand(transactionListCmd, transactionShowCmd, transactionUpsertCmd, transactionDeleteCmd, transactionActivityCmd)
	transactionListCmd.Flags().StringVar(&transactionCursor, "cursor", "", "Pagination cursor")
	transactionActivityCmd.Flags().StringVar(&transactionCursor, "cursor", "", "Pagination cursor")
	transactionUpsertCmd.Flags().StringVar(&transactionBody, "data", "", "JSON request body")
	transactionUpsertCmd.Flags().StringVar(&transactionBodyFile, "data-file", "", "Read JSON request body from file")
}
