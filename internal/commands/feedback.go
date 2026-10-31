package commands

import (
	"strings"

	"github.com/spf13/cobra"
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback MESSAGE...",
	Short: "Send a message to the WariWari team",
	Long: `Send feedback to the WariWari team: at most 4000 characters and 5 links.
Replies go to your account's email address. Limited to 5 messages a minute.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		body := map[string]any{"body": strings.Join(args, " ")}
		if _, err := api.Post(cmd.Context(), "/feedback", body, ""); err != nil {
			return err
		}
		return printOK(map[string]any{"sent": true}, "Feedback sent")
	},
}

func init() {
	rootCmd.AddCommand(feedbackCmd)
}
