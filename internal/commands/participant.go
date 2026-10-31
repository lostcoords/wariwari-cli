package commands

import "github.com/spf13/cobra"

var participantCmd = &cobra.Command{
	Use:   "participant",
	Short: "Manage group participants",
}

var participantBody string
var participantBodyFile string

var participantListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List participants",
	Long: `List a group's participants, read out of the group snapshot — there is no
participants list endpoint.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
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
	Long: `Create a participant from JSON. Developer API keys are supported.

Include an optional client-generated 24-character base58 token to make retries
an idempotent upsert. Reuse the same token and body when retrying.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		body, err := bodyFrom(participantBody, participantBodyFile)
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/participants", body, "")
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
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		body, err := bodyFrom(participantBody, participantBodyFile)
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/participants/"+args[0], body)
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
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		data, err := api.DeleteRevision(cmd.Context(), "/groups/"+group+"/participants/"+args[0], lockVersion)
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
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
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
	rootCmd.AddCommand(participantCmd)
	participantCmd.AddCommand(participantListCmd, participantBalanceCmd, participantCreateCmd, participantUpdateCmd, participantDeleteCmd, participantExampleCmd)
	participantExampleCmd.AddCommand(participantExampleCreateCmd, participantExampleUpdateCmd)
	participantCreateCmd.Flags().StringVar(&participantBody, "data", "", "JSON request body")
	participantCreateCmd.Flags().StringVar(&participantBodyFile, "data-file", "", "Read JSON request body from file")
	participantUpdateCmd.Flags().StringVar(&participantBody, "data", "", "JSON request body")
	participantUpdateCmd.Flags().StringVar(&participantBodyFile, "data-file", "", "Read JSON request body from file")
}
