package commands

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var commandsCmd = &cobra.Command{
	Use:     "commands [filter]",
	Aliases: []string{"cmds"},
	Short:   "List available commands",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := ""
		if len(args) > 0 {
			filter = strings.ToLower(args[0])
		}

		var rows []map[string]any
		walkCommands(rootCmd, func(c *cobra.Command) {
			if c.Hidden {
				return
			}
			use := c.CommandPath()
			if filter != "" && !strings.Contains(strings.ToLower(use), filter) {
				return
			}
			rows = append(rows, map[string]any{
				"command": use,
				"short":   c.Short,
			})
			for _, alias := range c.Aliases {
				aliasPath := aliasCommandPath(c, alias)
				if filter != "" && !strings.Contains(strings.ToLower(aliasPath), filter) {
					continue
				}
				rows = append(rows, map[string]any{
					"command": aliasPath,
					"short":   c.Short,
					"alias":   true,
				})
			}
		})
		sort.Slice(rows, func(i, j int) bool {
			return rows[i]["command"].(string) < rows[j]["command"].(string)
		})
		return printOK(rows, "Command catalog")
	},
}

func aliasCommandPath(cmd *cobra.Command, alias string) string {
	parent := cmd.Parent()
	if parent == nil {
		return alias
	}
	return parent.CommandPath() + " " + alias
}

func walkCommands(cmd *cobra.Command, visit func(*cobra.Command)) {
	visit(cmd)
	for _, child := range cmd.Commands() {
		walkCommands(child, visit)
	}
}

func init() {
	rootCmd.AddCommand(commandsCmd)
}
