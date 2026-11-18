package commands

import "github.com/spf13/cobra"

var groupIncludeMembers bool

var groupLeaveCmd = &cobra.Command{
	Use:   "leave [TOKEN]",
	Short: "Leave a group: unlink your account from your participant",
	Long: `Stop being a member. Your participant stays in the group as a guest with
its history, and the group leaves your list and your archive. The last admin
of a private or view-only group gets 400 VALIDATION_FAILED.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/membership", write(nil)); err != nil {
			return err
		}
		return printOK(map[string]any{"token": group, "left": true}, "Left group")
	},
}

var groupDuplicateCmd = &cobra.Command{
	Use:   "duplicate [TOKEN]",
	Short: "Start a new group with this one's categories",
	Long: `Create a new group with this group's categories, and with
--include-members its participants too. Returns the new group.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body := map[string]any{"includeMembers": groupIncludeMembers}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/duplication", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Group duplicated")
	},
}

var groupJoinCmd = &cobra.Command{
	Use:   "join [TOKEN]",
	Short: "Join a public group you have the link to",
	Long: `Become a registered member of a public-editable group and get your
participant back; joining a group you already belong to returns it too. For a
private group, ask with "wariwari access-request create" or accept an
invitation.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/membership", write(nil))
		if err != nil {
			return err
		}
		return printOK(data, "Joined group")
	},
}

var groupNotificationsCmd = &cobra.Command{
	Use:   "notifications",
	Short: "Your push notifications for a group",
}

var groupNotificationsShowCmd = &cobra.Command{
	Use:   "show [TOKEN]",
	Short: "Show whether you muted push notifications for a group",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/notification_settings", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Notification settings")
	},
}

var groupNotificationsMuteCmd = notificationToggle("mute", true)
var groupNotificationsUnmuteCmd = notificationToggle("unmute", false)

func notificationToggle(use string, muted bool) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [TOKEN]",
		Short: map[bool]string{true: "Mute", false: "Unmute"}[muted] + " push notifications for a group",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			group, err := groupArg(args, 0)
			if err != nil {
				return err
			}
			data, err := api.Put(cmd.Context(), "/groups/"+group+"/notification_settings",
				write(map[string]any{"muted": muted}))
			if err != nil {
				return err
			}
			return printOK(data, "Notification settings updated")
		},
	}
}

func init() {
	groupCmd.AddCommand(groupLeaveCmd, groupDuplicateCmd, groupJoinCmd, groupNotificationsCmd)
	groupNotificationsCmd.AddCommand(groupNotificationsShowCmd, groupNotificationsMuteCmd, groupNotificationsUnmuteCmd)
	addIdempotencyFlag(groupLeaveCmd, groupDuplicateCmd, groupJoinCmd, groupNotificationsMuteCmd, groupNotificationsUnmuteCmd)
	groupDuplicateCmd.Flags().BoolVar(&groupIncludeMembers, "include-members", false, "Copy the participants too")
}
