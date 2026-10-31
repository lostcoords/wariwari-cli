package commands

import "github.com/spf13/cobra"

var invitationCmd = &cobra.Command{
	Use:   "invitation",
	Short: "Manage invite links for private groups",
}

var invitationRole string
var invitationExpiresIn string

var invitationCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Create an invitation",
	Long: `Create an invitation to a private group. Admin-only. --role is the
membershipRole whoever accepts gets (read_only, editor, admin; default editor).
--expires-in is 24h, 7d or 30d; omit it for an invitation that never expires.

For a public group, share the snapshot's shareUrl instead.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		body := map[string]any{}
		if invitationRole != "" {
			body["membershipRole"] = invitationRole
		}
		if invitationExpiresIn != "" {
			body["expiresIn"] = invitationExpiresIn
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/invitations", body, "")
		if err != nil {
			return err
		}
		return printOK(data, "Invitation created")
	},
}

var invitationRevokeCmd = &cobra.Command{
	Use:   "revoke INVITATION_TOKEN [GROUP_TOKEN]",
	Short: "Revoke an invitation",
	Long:  "Revoke an outstanding invitation so its link stops working. Admin-only.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/invitations/"+args[0]); err != nil {
			return err
		}
		return printOK(map[string]any{"token": args[0], "revoked": true}, "Invitation revoked")
	},
}

func init() {
	rootCmd.AddCommand(invitationCmd)
	invitationCmd.AddCommand(invitationCreateCmd, invitationRevokeCmd)
	invitationCreateCmd.Flags().StringVar(&invitationRole, "role", "", "Role for whoever accepts: read_only, editor, admin")
	invitationCreateCmd.Flags().StringVar(&invitationExpiresIn, "expires-in", "", "Expiry window: 24h, 7d, 30d")
}
