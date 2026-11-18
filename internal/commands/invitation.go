package commands

import "github.com/spf13/cobra"

var invitationCmd = &cobra.Command{
	Use:   "invitation",
	Short: "Manage invite links for view-only and private groups",
}

var invitationRole string
var invitationExpiresAt string

var invitationListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List invitations that can still be used",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/invitations", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Invitations")
	},
}

var invitationCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Create an invitation",
	Long: `Create an invitation to a view-only or private group and print its url.
Admin-only. --role is the
membershipRole whoever accepts gets (read_only, editor, admin; default editor).
--expires-at is when the link stops working, an ISO 8601 date-time in the
future (2026-12-31T23:59:00+09:00); omit it for an invitation that never
expires.

For a public group, share the snapshot's shareUrl instead.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body := map[string]any{}
		if invitationRole != "" {
			body["membershipRole"] = invitationRole
		}
		if invitationExpiresAt != "" {
			body["expiresAt"] = invitationExpiresAt
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/invitations", write(body))
		if err != nil {
			return err
		}
		invitation, _ := data.(map[string]any)
		url, _ := invitation["url"].(string)
		token, _ := invitation["token"].(string)
		return printOK(data, "Invitation created; share "+url,
			crumb("revoke", "wariwari invitation revoke "+token+" "+group, "Stop the link working"))
	},
}

var invitationRevokeCmd = &cobra.Command{
	Use:   "revoke INVITATION_TOKEN [GROUP_TOKEN]",
	Short: "Revoke an invitation",
	Long:  "Revoke an outstanding invitation so its link stops working. Admin-only.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/invitations/"+args[0], write(nil)); err != nil {
			return err
		}
		return printOK(map[string]any{"token": args[0], "revoked": true}, "Invitation revoked")
	},
}

var invitationAcceptCmd = &cobra.Command{
	Use:   "accept INVITATION_TOKEN GROUP_TOKEN",
	Short: "Accept an invitation and join the group",
	Long:  "Accept an invitation and join the group. Returns the participant you are now linked to.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/invitations/"+args[0]+"/acceptance", write(nil))
		if err != nil {
			return err
		}
		return printOK(data, "Invitation accepted")
	},
}

func init() {
	rootCmd.AddCommand(invitationCmd)
	invitationCmd.AddCommand(invitationListCmd, invitationCreateCmd, invitationRevokeCmd, invitationAcceptCmd)
	addIdempotencyFlag(invitationCreateCmd, invitationRevokeCmd, invitationAcceptCmd)
	invitationCreateCmd.Flags().StringVar(&invitationRole, "role", "", "Role for whoever accepts: read_only, editor, admin")
	invitationCreateCmd.Flags().StringVar(&invitationExpiresAt, "expires-at", "", "When the link stops working, ISO 8601 with an offset")
}
