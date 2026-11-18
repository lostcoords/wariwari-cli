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
	Long: `List a group's categories, read out of the group snapshot's categories —
there is no categories list endpoint. An expense names one by categoryToken.`,
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
		return printOK(map[string]any{"categories": snapshotList(snapshot, "categories")}, "Categories")
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
			"name":     "Food",
			"emoji":    "🍜",
			"position": 1,
		}, "Category update example")
	},
}

var categoryUpdateCmd = &cobra.Command{
	Use:   "update CATEGORY_TOKEN [GROUP_TOKEN]",
	Short: "Update a category from JSON",
	Long: `Update a category: send only what changes of name, emoji and position.
Returns the category.` + guardedUpdateHelp,
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
		w, err := revisedWrite(cmd, body, snapshotRecord(group, "categories", args[0]))
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/categories/"+args[0], w)
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
		w, err := revisedWrite(cmd, nil, snapshotRecord(group, "categories", args[0]))
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/categories/"+args[0], w); err != nil {
			return err
		}
		return printOK(map[string]any{"token": args[0], "deleted": true}, "Category deleted")
	},
}

func init() {
	addLockVersionFlag(categoryUpdateCmd, categoryDeleteCmd)
	addDataFlags(categoryCreateCmd, categoryUpdateCmd)
	addIdempotencyFlag(categoryCreateCmd, categoryUpdateCmd, categoryDeleteCmd)
	rootCmd.AddCommand(categoryCmd)
	categoryCmd.AddCommand(categoryListCmd, categoryCreateCmd, categoryUpdateCmd, categoryDeleteCmd, categoryExampleCmd)
	categoryExampleCmd.AddCommand(categoryExampleCreateCmd, categoryExampleUpdateCmd)
}
