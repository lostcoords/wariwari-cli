package commands

import "github.com/spf13/cobra"

var participantCmd = &cobra.Command{
	Use:   "participant",
	Short: "Manage group participants",
}

var participantListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List participants",
	Long: `List a group's participants, read out of the group snapshot — there is no
participants list endpoint.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		snapshot, err := groupSnapshot(cmd.Context(), group)
		if err != nil {
			return err
		}
		return printOK(map[string]any{"participants": snapshotList(snapshot, "participants")}, "Participants")
	},
}

var participantCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Create a participant from JSON",
	Long: `Create a participant from JSON: name, optional color (one of the palette
hex values in the API spec) and an optional client-generated 24-character
base58 token. Reusing a token returns 409 DUPLICATE_TOKEN; retry a create
safely with --idempotency-key instead.`,
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
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/participants", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Participant created")
	},
}

var participantExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Print example request payloads",
}

var participantExampleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Example payload for creating a participant",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"token": "qm1mVhS7nyyF99vfPYwfF3nN",
			"name":  "Ada",
			"color": "#1F3A5F",
		}, "Participant create example")
	},
}

var participantExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a participant",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"name":        "Ada Lovelace",
			"color":       "#1F3A5F",
			"lockVersion": 0,
		}, "Participant update example")
	},
}

var participantUpdateCmd = &cobra.Command{
	Use:   "update PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Update a participant from JSON",
	Long: `Update a participant. lockVersion is required — send the value from the
current snapshot. A stale value returns 409 CONFLICT with the fresh participant
under "current".`,
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
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/participants/"+args[0], write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Participant updated")
	},
}

var participantDeleteCmd = &cobra.Command{
	Use:   "delete PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Soft-delete a participant",
	Long:  guardedDeleteHelp,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, nil, snapshotRecord(group, "participants", args[0]))
		if err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/groups/"+group+"/participants/"+args[0], w)
		if err != nil {
			return err
		}
		return printOK(data, "Participant deleted")
	},
}

var participantBalanceCmd = &cobra.Command{
	Use:   "balance PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Show how a participant's balance adds up",
	Long: `Show one participant's balance ledger in the group currency: balanceCents
(positive means they are owed) and four lines, paid_for_others, paid_for_you,
sent and received, each listing the transactions behind it by token.
otherCurrencies is true when they also have transactions kept in another
currency, which are balanced separately and left out here.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/participants/"+args[0]+"/balance", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Participant balance",
			crumb("transactions", "wariwari transaction list "+group+" --participant "+args[0], "List the transactions behind it"))
	},
}

func init() {
	addLockVersionFlag(participantDeleteCmd)
	addDataFlags(participantCreateCmd, participantUpdateCmd)
	addIdempotencyFlag(participantCreateCmd, participantUpdateCmd, participantDeleteCmd)
	rootCmd.AddCommand(participantCmd)
	participantCmd.AddCommand(participantListCmd, participantBalanceCmd, participantCreateCmd, participantUpdateCmd, participantDeleteCmd, participantExampleCmd)
	participantExampleCmd.AddCommand(participantExampleCreateCmd, participantExampleUpdateCmd)
}
