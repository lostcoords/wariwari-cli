package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var invitationCmd = &cobra.Command{
	Use:   "invitation",
	Short: "Manage invite links for view-only and private groups",
}

var invitationRole string
var invitationExpiresIn string

// invitationNow is the clock --expires-in counts from; tests replace it.
var invitationNow = time.Now

// expiresAt turns an --expires-in window (90m, 24h, 7d) into the date-time the
// server takes. Days are calendar days of 24 hours, as the server counts them.
func expiresAt(window string) (string, error) {
	var d time.Duration
	if days, ok := strings.CutSuffix(window, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil {
			return "", fmt.Errorf("--expires-in %q: use a number of days like 7d, or hours like 24h", window)
		}
		d = time.Duration(n) * 24 * time.Hour
	} else {
		parsed, err := time.ParseDuration(window)
		if err != nil {
			return "", fmt.Errorf("--expires-in %q: use a number of days like 7d, or hours like 24h", window)
		}
		d = parsed
	}
	if d <= 0 {
		return "", fmt.Errorf("--expires-in %q: the window must be in the future", window)
	}
	return invitationNow().Add(d).Format(time.RFC3339), nil
}

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
--expires-in is how long the link works, in days or hours (7d, 24h, 30d);
omit it for an invitation that never expires.

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
		if invitationExpiresIn != "" {
			at, err := expiresAt(invitationExpiresIn)
			if err != nil {
				return err
			}
			body["expiresAt"] = at
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
	invitationCreateCmd.Flags().StringVar(&invitationExpiresIn, "expires-in", "", "How long the link works: 7d, 24h, 30d")
}
