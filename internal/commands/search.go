package commands

import (
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [QUERY...]",
	Short: "Search your groups, people and expenses",
	Long: `Search every group you belong to. Matches group names, member names,
expense descriptions and categories. A query that is just an amount (7000,
¥7,000, 12.50) or an ISO date (2026-05-22) also matches expenses with that
amount or date. With no query, lists your most recently active groups.

Returns up to 6 groups, 6 people and 8 transactions, each with its group.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/search", map[string]string{"q": strings.Join(args, " ")})
		if err != nil {
			return err
		}
		return printOK(data, "Search results")
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
