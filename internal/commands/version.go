package commands

import "github.com/spf13/cobra"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show WariWari CLI version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{"version": version}, version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
