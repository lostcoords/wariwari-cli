package commands

import "github.com/spf13/cobra"

var paymentCmd = &cobra.Command{
	Use:   "payment",
	Short: "Record settle-up payments between participants",
	Long: `A payment is a transaction of kind "payment": money one participant sent
another. These commands write it through the transaction routes and set kind
for you. Read one with "transaction show", list them with "transaction list"
and delete one with "transaction delete".`,
}

var paymentCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Record a payment from JSON",
	Long: `Record a settlement: fromParticipantToken paid toParticipantToken
amountCents. currency defaults to the group currency; description, date and
timeOfDay are optional. Take the amounts from the snapshot's suggestedPayments
to settle up. Returns the payment.

Retry safely with --idempotency-key and the same body.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, err := paymentBody()
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/transactions", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Payment recorded")
	},
}

var paymentUpdateCmd = &cobra.Command{
	Use:   "update PAYMENT_TOKEN [GROUP_TOKEN]",
	Short: "Update a payment from JSON",
	Long: `Update a payment: send only what changes of fromParticipantToken,
toParticipantToken, amountCents, currency, description, date and timeOfDay.
A blank description resets it to the default "From → To" label. Returns the
payment.` + guardedUpdateHelp,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		body, err := paymentBody()
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
		return printOK(data, "Payment updated")
	},
}

// paymentBody reads the JSON body and names its kind, which the transaction
// routes require on every write.
func paymentBody() (map[string]any, error) {
	body, err := commandData()
	if err != nil {
		return nil, err
	}
	body["kind"] = "payment"
	return body, nil
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
			"token":                "5xKpQ2mNbV8cRtY3wZ9aHfJd",
			"fromParticipantToken": "8mPNkR4qQXtLbJR5CcxT6yWa",
			"toParticipantToken":   "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"amountCents":          4000,
			"currency":             "JPY",
			"date":                 "2026-05-24",
		}, "Payment create example")
	},
}

var paymentExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a payment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"amountCents": 4500,
		}, "Payment update example")
	},
}

func init() {
	rootCmd.AddCommand(paymentCmd)
	addLockVersionFlag(paymentUpdateCmd)
	addDataFlags(paymentCreateCmd, paymentUpdateCmd)
	addIdempotencyFlag(paymentCreateCmd, paymentUpdateCmd)
	paymentCmd.AddCommand(paymentCreateCmd, paymentUpdateCmd, paymentExampleCmd)
	paymentExampleCmd.AddCommand(paymentExampleCreateCmd, paymentExampleUpdateCmd)
}
