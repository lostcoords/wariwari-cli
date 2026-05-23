package commands

import "github.com/spf13/cobra"

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage group categories",
}

var categoryBody string
var categoryBodyFile string

var categoryListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List categories",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/categories", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Categories")
	},
}

var categoryUpsertCmd = &cobra.Command{
	Use:   "upsert CATEGORY_TOKEN [GROUP_TOKEN]",
	Short: "Create or update a category from JSON",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		body, err := categoryCommandBody()
		if err != nil {
			return err
		}
		data, err := api.Put(cmd.Context(), "/groups/"+group+"/categories/"+args[0], body)
		if err != nil {
			return err
		}
		return printOK(data, "Category saved")
	},
}

var categoryDeleteCmd = &cobra.Command{
	Use:   "delete CATEGORY_TOKEN [GROUP_TOKEN]",
	Short: "Delete a category",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(optionalArg(args, 1))
		if err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/groups/"+group+"/categories/"+args[0])
		if err != nil {
			return err
		}
		return printOK(data, "Category deleted")
	},
}

func categoryCommandBody() (map[string]any, error) {
	if categoryBodyFile != "" {
		return parseObjectFile(categoryBodyFile)
	}
	return parseObject(categoryBody)
}

func init() {
	rootCmd.AddCommand(categoryCmd)
	categoryCmd.AddCommand(categoryListCmd, categoryUpsertCmd, categoryDeleteCmd)
	categoryUpsertCmd.Flags().StringVar(&categoryBody, "data", "", "JSON request body")
	categoryUpsertCmd.Flags().StringVar(&categoryBodyFile, "data-file", "", "Read JSON request body from file")
}
