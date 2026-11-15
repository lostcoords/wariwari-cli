package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	participantMergeInto string
	participantPosition  int
	participantUp        bool
	participantDown      bool
)

var participantClaimCmd = &cobra.Command{
	Use:   "claim PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Link a guest participant to your account",
	Long: `Say "this participant is me". If you are already linked to another
participant in the group, the guest is merged into yours.`,
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

var participantUnlinkCmd = &cobra.Command{
	Use:   "unlink PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Unlink your account from a participant",
	Long: `Stop being this participant; it stays in the group as a guest. Refused if
it would leave a private group without an admin.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/groups/"+group+"/participants/"+args[0]+"/link", write(nil))
		if err != nil {
			return err
		}
		return printOK(data, "Participant unlinked")
	},
}

var participantMergeCmd = &cobra.Command{
	Use:   "merge PARTICIPANT_TOKEN [GROUP_TOKEN] --into WINNER_TOKEN",
	Short: "Merge a duplicate participant into another",
	Long: `Fold PARTICIPANT_TOKEN into --into: every transaction naming it moves to the
winner (and gets a new lockVersion). The API requires the merged participant's
lockVersion; pass --lock-version or let the CLI read it from the snapshot.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, map[string]any{"into": participantMergeInto}, snapshotRecord(group, "participants", args[0]))
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
	Use:     "move PARTICIPANT_TOKEN [GROUP_TOKEN] (--position N | --up | --down)",
	Aliases: []string{"reorder"},
	Short:   "Reorder a participant",
	Long: `Move a participant to an absolute --position (1-based) or one step --up or
--down. Reordering changes no lockVersion. The API still checks the moved
participant's lockVersion; pass --lock-version or let the CLI read it.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := moveBody(cmd)
		if err != nil {
			return err
		}
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, body, snapshotRecord(group, "participants", args[0]))
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/participants/"+args[0]+"/position", w)
		if err != nil {
			return err
		}
		return printOK(data, "Participant moved")
	},
}

func moveBody(cmd *cobra.Command) (map[string]any, error) {
	switch {
	case cmd.Flags().Changed("position") && !participantUp && !participantDown:
		return map[string]any{"position": participantPosition}, nil
	case participantUp && !participantDown && !cmd.Flags().Changed("position"):
		return map[string]any{"direction": "up"}, nil
	case participantDown && !participantUp && !cmd.Flags().Changed("position"):
		return map[string]any{"direction": "down"}, nil
	}
	return nil, fmt.Errorf("pass exactly one of --position N, --up or --down")
}

func init() {
	participantCmd.AddCommand(participantClaimCmd, participantUnlinkCmd, participantMergeCmd, participantMoveCmd)
	addIdempotencyFlag(participantClaimCmd, participantMergeCmd, participantMoveCmd)
	addLockVersionFlag(participantMergeCmd, participantMoveCmd)
	participantMergeCmd.Flags().StringVar(&participantMergeInto, "into", "", "Token of the participant to keep")
	_ = participantMergeCmd.MarkFlagRequired("into")
	participantMoveCmd.Flags().IntVar(&participantPosition, "position", 0, "Absolute 1-based position")
	participantMoveCmd.Flags().BoolVar(&participantUp, "up", false, "Move one step up")
	participantMoveCmd.Flags().BoolVar(&participantDown, "down", false, "Move one step down")
}
