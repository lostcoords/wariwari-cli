package commands

import "github.com/spf13/cobra"

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage group categories",
}

var categoryListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List categories",
	Long: `List a group's categories, read out of the group snapshot's groupCategories —
there is no categories list endpoint.`,
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
		return printOK(map[string]any{"groupCategories": snapshotList(snapshot, "groupCategories")}, "Categories")
	},
}

var categoryCreateCmd = &cobra.Command{
	Use:   "create [GROUP_TOKEN]",
	Short: "Create a category from JSON",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, err := commandData()
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/categories", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Category created")
	},
}

var categoryExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Print example request payloads",
}

var categoryExampleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Example payload for creating a category",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"token":    "4QfYk8nE2pRtVwXz7aBcDeFg",
			"name":     "Food",
			"emoji":    "🍜",
			"position": 1,
		}, "Category create example")
	},
}

var categoryExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a category",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"name":        "Food",
			"emoji":       "🍜",
			"position":    1,
			"lockVersion": 0,
		}, "Category update example")
	},
}

var categoryUpdateCmd = &cobra.Command{
	Use:   "update CATEGORY_TOKEN [GROUP_TOKEN]",
	Short: "Update a category from JSON",
	Long: `Update a category. lockVersion is required — send the value from the current
snapshot. A stale value returns 409 CONFLICT with the fresh category under
"current".`,
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
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/categories/"+args[0], write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Category updated")
	},
}

var categoryDeleteCmd = &cobra.Command{
	Use:   "delete CATEGORY_TOKEN [GROUP_TOKEN]",
	Short: "Delete a category",
	Long:  guardedDeleteHelp,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, nil, snapshotRecord(group, "groupCategories", args[0]))
		if err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/groups/"+group+"/categories/"+args[0], w)
		if err != nil {
			return err
		}
		return printOK(data, "Category deleted")
	},
}

func init() {
	addLockVersionFlag(categoryDeleteCmd)
	addDataFlags(categoryCreateCmd, categoryUpdateCmd)
	addIdempotencyFlag(categoryCreateCmd, categoryUpdateCmd, categoryDeleteCmd)
	rootCmd.AddCommand(categoryCmd)
	categoryCmd.AddCommand(categoryListCmd, categoryCreateCmd, categoryUpdateCmd, categoryDeleteCmd, categoryExampleCmd)
	categoryExampleCmd.AddCommand(categoryExampleCreateCmd, categoryExampleUpdateCmd)
}
