package commands

import "github.com/spf13/cobra"

var accessRequestCmd = &cobra.Command{
	Use:     "access-request",
	Aliases: []string{"access"},
	Short:   "Ask to join private groups, and approve or decline requests",
}

var accessRequestStatusCmd = &cobra.Command{
	Use:   "status [GROUP_TOKEN]",
	Short: "Show where your own request to join stands",
	Long: `Show your own request to join a group: none, pending, declined, or viewable
(you can already open it). Works for a private group you cannot open yet.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/access_request", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Access request status")
	},
}

var accessRequestCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Ask a private group's admins to let you in",
	Long: `Ask to join a private group; the admins are notified. Idempotent: asking
while a request is pending, or within 7 days of a decline, changes nothing and
returns the current status.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/access_request", nil, "")
		if err != nil {
			return err
		}
		return printOK(data, "Access requested")
	},
}

var accessRequestListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List pending requests to join a group",
	Long:    "List the group's pending access requests, oldest first. Admin-only.",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/access_requests", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Pending access requests",
			crumb("approve", "wariwari access-request approve ID "+group, "Let a requester in as an editor"),
			crumb("decline", "wariwari access-request decline ID "+group, "Turn a request down"))
	},
}

var accessRequestApproveCmd = &cobra.Command{
	Use:   "approve ACCESS_REQUEST_ID [GROUP_TOKEN]",
	Short: "Approve a request and add the requester as an editor",
	Long:  "Approve a pending request. Admin-only. The requester joins as an editor and is notified. Returns the refreshed group snapshot.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/access_requests/"+args[0]+"/approval", nil, "")
		if err != nil {
			return err
		}
		return printOK(data, "Access request approved")
	},
}

var accessRequestDeclineCmd = &cobra.Command{
	Use:   "decline ACCESS_REQUEST_ID [GROUP_TOKEN]",
	Short: "Decline a request",
	Long:  "Decline a pending request. Admin-only. The requester is not notified and cannot ask again for 7 days.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		if _, err := api.Post(cmd.Context(), "/groups/"+group+"/access_requests/"+args[0]+"/decline", nil, ""); err != nil {
			return err
		}
		return printOK(map[string]any{"id": args[0], "declined": true}, "Access request declined")
	},
}

func init() {
	rootCmd.AddCommand(accessRequestCmd)
	accessRequestCmd.AddCommand(accessRequestStatusCmd, accessRequestCreateCmd, accessRequestListCmd, accessRequestApproveCmd, accessRequestDeclineCmd)
}
