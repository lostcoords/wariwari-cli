package commands

import "github.com/spf13/cobra"

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show or update your account profile",
}

var accountBody string
var accountBodyFile string

var accountShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the authenticated account",
	RunE:  runAccountShow,
}

// `me` predates `account show`; it reads the same GET /v1/account payload.
var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated account (same as account show)",
	RunE:  runAccountShow,
}

func runAccountShow(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}
	data, err := api.Get(cmd.Context(), "/account", nil)
	if err != nil {
		return err
	}
	return printOK(data, "Account")
}

var accountUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update profile fields from JSON",
	Long: `Update profile fields. The body is flat (no "user" wrapper) and partial:
send only what changes. Fields: name, avatarColor, removeAvatar,
preferredCurrency, locale (en, es, ja), marketingOptIn, and
notificationPreferences (newExpense, paymentReceived, invitationAccepted,
accessRequested, accessApproved). Returns the updated account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		body, err := bodyFrom(accountBody, accountBodyFile)
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/account", body)
		if err != nil {
			return err
		}
		return printOK(data, "Account updated")
	},
}

func init() {
	rootCmd.AddCommand(accountCmd, meCmd)
	accountCmd.AddCommand(accountShowCmd, accountUpdateCmd)
	accountUpdateCmd.Flags().StringVar(&accountBody, "data", "", "JSON request body")
	accountUpdateCmd.Flags().StringVar(&accountBodyFile, "data-file", "", "Read JSON request body from file")
}
