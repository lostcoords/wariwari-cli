package commands

import "github.com/spf13/cobra"

var paymentCmd = &cobra.Command{
	Use:   "payment",
	Short: "Record settle-up payments between participants",
}

var paymentBody string
var paymentBodyFile string

var paymentCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Record a payment from JSON",
	Long: `Record a settlement: fromToken paid toToken amountCents. currency defaults
to the group currency; description, date and timeOfDay are optional. Take the
amounts from the snapshot's suggestedPayments to settle up.

Include an optional client-generated 24-character base58 token to make retries
safe. Returns the refreshed group snapshot.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		body, err := bodyFrom(paymentBody, paymentBodyFile)
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/payments", body, "")
		if err != nil {
			return err
		}
		return printOK(data, "Payment recorded")
	},
}

var paymentUpdateCmd = &cobra.Command{
	Use:   "update PAYMENT_TOKEN [GROUP_TOKEN]",
	Short: "Update a payment from JSON",
	Long: `Update a payment. The body replaces the payment: send fromToken, toToken,
amountCents and the payment's lockVersion (all required), plus currency,
description, date and timeOfDay to keep them. An omitted date resets to today.

A payment is a transaction of kind payment: read it with "transaction show" and
delete it with "transaction delete". Returns the refreshed group snapshot.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		body, err := bodyFrom(paymentBody, paymentBodyFile)
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/payments/"+args[0], body)
		if err != nil {
			return err
		}
		return printOK(data, "Payment updated")
	},
}

var paymentExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Print example request payloads",
}

var paymentExampleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Example payload for recording a payment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"token":       "5xKpQ2mNbV8cRtY3wZ9aHfJd",
			"fromToken":   "8mPNkR4qQXtLbJR5CcxT6yWa",
			"toToken":     "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"amountCents": 4000,
			"currency":    "JPY",
			"date":        "2026-05-24",
		}, "Payment create example")
	},
}

var paymentExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a payment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"fromToken":   "8mPNkR4qQXtLbJR5CcxT6yWa",
			"toToken":     "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"amountCents": 4500,
			"lockVersion": 0,
		}, "Payment update example")
	},
}

func init() {
	rootCmd.AddCommand(paymentCmd)
	paymentCmd.AddCommand(paymentCreateCmd, paymentUpdateCmd, paymentExampleCmd)
	paymentExampleCmd.AddCommand(paymentExampleCreateCmd, paymentExampleUpdateCmd)
	for _, cmd := range []*cobra.Command{paymentCreateCmd, paymentUpdateCmd} {
		cmd.Flags().StringVar(&paymentBody, "data", "", "JSON request body")
		cmd.Flags().StringVar(&paymentBodyFile, "data-file", "", "Read JSON request body from file")
	}
}
