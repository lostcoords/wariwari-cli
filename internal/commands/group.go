package commands

import (
	"github.com/spf13/cobra"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage groups",
}

var groupCursor string
var groupBody string
var groupBodyFile string
var groupIdempotencyKey string

var groupListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List visible groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups", queryWithPaging(groupCursor))
		if err != nil {
			return err
		}
		return printOK(data, "Groups", crumb("show", "wariwari group show TOKEN", "Show a group"))
	},
}

var groupShowCmd = &cobra.Command{
	Use:     "show [TOKEN]",
	Aliases: []string{"view"},
	Short:   "Show a group",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group, nil)
		if err != nil {
			return err
		}
		return printOK(data, "Group", crumb("snapshot", "wariwari group snapshot "+group, "Fetch full group snapshot"))
	},
}

var groupSnapshotCmd = &cobra.Command{
	Use:   "snapshot [TOKEN]",
	Short: "Fetch a full group snapshot",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/snapshot", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Group snapshot")
	},
}

var groupActivityCmd = &cobra.Command{
	Use:   "activity [TOKEN]",
	Short: "List group activity",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/activity", queryWithPaging(groupCursor))
		if err != nil {
			return err
		}
		return printOK(data, "Group activity")
	},
}

var groupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group from JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		body, err := commandBody()
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups", body, groupIdempotencyKey)
		if err != nil {
			return err
		}
		return printOK(data, "Group created")
	},
}

var groupUpdateCmd = &cobra.Command{
	Use:   "update [TOKEN]",
	Short: "Update a group from JSON",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		body, err := commandBody()
		if err != nil {
			return err
		}
		data, err := api.Put(cmd.Context(), "/groups/"+group, body)
		if err != nil {
			return err
		}
		return printOK(data, "Group updated")
	},
}

func commandBody() (map[string]any, error) {
	if groupBodyFile != "" {
		return parseObjectFile(groupBodyFile)
	}
	return parseObject(groupBody)
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func init() {
	rootCmd.AddCommand(groupCmd)
	groupCmd.AddCommand(groupListCmd, groupShowCmd, groupSnapshotCmd, groupActivityCmd, groupCreateCmd, groupUpdateCmd)
	groupListCmd.Flags().StringVar(&groupCursor, "cursor", "", "Pagination cursor")
	groupActivityCmd.Flags().StringVar(&groupCursor, "cursor", "", "Pagination cursor")
	groupCreateCmd.Flags().StringVar(&groupBody, "data", "", "JSON request body")
	groupCreateCmd.Flags().StringVar(&groupBodyFile, "data-file", "", "Read JSON request body from file")
	groupCreateCmd.Flags().StringVar(&groupIdempotencyKey, "idempotency-key", "", "Idempotency-Key header")
	groupUpdateCmd.Flags().StringVar(&groupBody, "data", "", "JSON request body")
	groupUpdateCmd.Flags().StringVar(&groupBodyFile, "data-file", "", "Read JSON request body from file")
}
