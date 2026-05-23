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
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/participants", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Participants")
	},
}

var participantUpsertCmd = &cobra.Command{
	Use:   "upsert PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Create or update a participant from JSON",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		body, err := participantCommandBody()
		if err != nil {
			return err
		}
		data, err := api.Put(cmd.Context(), "/groups/"+group+"/participants/"+args[0], body)
		if err != nil {
			return err
		}
		return printOK(data, "Participant saved")
	},
}

var participantDeleteCmd = &cobra.Command{
	Use:   "delete PARTICIPANT_TOKEN [GROUP_TOKEN]",
	Short: "Soft-delete a participant",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/groups/"+group+"/participants/"+args[0])
		if err != nil {
			return err
		}
		return printOK(data, "Participant deleted")
	},
}

func participantCommandBody() (map[string]any, error) {
	if participantBodyFile != "" {
		return parseObjectFile(participantBodyFile)
	}
	return parseObject(participantBody)
}

func optionalArg(args []string, index int) string {
	if len(args) <= index {
		return ""
	}
	return args[index]
}

func init() {
	rootCmd.AddCommand(participantCmd)
	participantCmd.AddCommand(participantListCmd, participantUpsertCmd, participantDeleteCmd)
	participantUpsertCmd.Flags().StringVar(&participantBody, "data", "", "JSON request body")
	participantUpsertCmd.Flags().StringVar(&participantBodyFile, "data-file", "", "Read JSON request body from file")
}
