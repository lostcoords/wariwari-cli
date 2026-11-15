package commands

import (
	"strings"

	"github.com/spf13/cobra"
)

var feedbackEmail string

var feedbackCmd = &cobra.Command{
	Use:   "feedback MESSAGE...",
	Short: "Send a message to the WariWari team",
	Long: `Send feedback to the WariWari team: at most 4000 characters and 5 links.
Signed in, replies go to your account's email address; without a token, pass
--email to get one. Limited to 5 messages an hour.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := map[string]any{"body": strings.Join(args, " ")}
		if feedbackEmail != "" {
			body["email"] = feedbackEmail
		}
		if _, err := api.Post(cmd.Context(), "/feedback", write(body)); err != nil {
			return err
		}
		return printOK(map[string]any{"sent": true}, "Feedback sent")
	},
}

func init() {
	rootCmd.AddCommand(feedbackCmd)
	feedbackCmd.Flags().StringVar(&feedbackEmail, "email", "", "Reply-to address when not signed in")
}
