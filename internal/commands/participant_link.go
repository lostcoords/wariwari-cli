package commands

import "github.com/spf13/cobra"

var (
	participantMergeInto string
	participantPosition  int
)

var participantClaimCmd = &cobra.Command{
	Use:   "claim PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Link a guest participant to your account",
	Long: `Say "this participant is me". If you are already linked to another
participant in the group, the guest is merged into yours. Returns the
participant you are linked to. To stop being a member, "wariwari group leave".`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/participants/"+args[0]+"/claim", write(nil))
		if err != nil {
			return err
		}
		return printOK(data, "Participant claimed")
	},
}

var participantMergeCmd = &cobra.Command{
	Use:   "merge PARTICIPANT_TOKEN [GROUP_TOKEN] --into WINNER_TOKEN",
	Short: "Merge a duplicate participant into another",
	Long: `Fold PARTICIPANT_TOKEN into --into: every transaction naming it moves to the
winner (and gets a new lockVersion). Returns the winner. The API requires the
merged participant's lockVersion; pass --lock-version or let the CLI read it
from the snapshot.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, map[string]any{"intoParticipantToken": participantMergeInto}, snapshotRecord(group, "participants", args[0]))
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/participants/"+args[0]+"/merge", w)
		if err != nil {
			return err
		}
		return printOK(data, "Participant merged")
	},
}

var participantMoveCmd = &cobra.Command{
	Use:     "move PARTICIPANT_TOKEN [GROUP_TOKEN] --position N",
	Aliases: []string{"reorder"},
	Short:   "Reorder a participant",
	Long: `Move a participant to an absolute 1-based --position; one past the end of
the list leaves the order unchanged. Reordering changes no lockVersion and
takes none. Returns the participant.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/participants/"+args[0]+"/position",
			write(map[string]any{"position": participantPosition}))
		if err != nil {
			return err
		}
		return printOK(data, "Participant moved")
	},
}

func init() {
	participantCmd.AddCommand(participantClaimCmd, participantMergeCmd, participantMoveCmd)
	addIdempotencyFlag(participantClaimCmd, participantMergeCmd, participantMoveCmd)
	addLockVersionFlag(participantMergeCmd)
	participantMergeCmd.Flags().StringVar(&participantMergeInto, "into", "", "Token of the participant to keep")
	_ = participantMergeCmd.MarkFlagRequired("into")
	participantMoveCmd.Flags().IntVar(&participantPosition, "position", 0, "Absolute 1-based position")
	_ = participantMoveCmd.MarkFlagRequired("position")
}
