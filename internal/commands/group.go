package commands

import (
	"github.com/spf13/cobra"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage groups",
}

var groupCursor string
var groupTake int
var groupArchived bool
var groupBody string
var groupBodyFile string
var groupIdempotencyKey string

var groupListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List visible groups",
	Long: `List the groups the authenticated user participates in.

Keyset paginated: the response carries pagination.hasMore and pagination.cursor.
Pass that cursor back with --cursor to fetch the next page.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		query := pagingQuery(groupTake, groupCursor)
		if groupArchived {
			query["archived"] = "true"
		}
		data, err := api.Get(cmd.Context(), "/groups", query)
		if err != nil {
			return err
		}
		return printOK(data, "Groups",
			crumb("show", "wariwari group show TOKEN", "Fetch the full group snapshot"),
			nextPageCrumb(data, "wariwari group list --cursor"))
	},
}

var groupShowCmd = &cobra.Command{
	Use:     "show [TOKEN]",
	Aliases: []string{"view", "snapshot"},
	Short:   "Fetch the full group snapshot",
	Long: `Fetch a group snapshot: metadata, groupCategories, participants,
transactions, netByCurrency, suggestedPayments and spending totals.

This one payload is the whole group read surface — there are no separate
participants, categories or balances endpoints.`,
	Args: cobra.MaximumNArgs(1),
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
		return printOK(data, "Group snapshot",
			crumb("balance", "wariwari balance "+group, "Show balances and suggested payments"),
			crumb("transactions", "wariwari transaction list "+group, "Page through transactions"))
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
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/activity", pagingQuery(groupTake, groupCursor))
		if err != nil {
			return err
		}
		return printOK(data, "Group activity",
			nextPageCrumb(data, "wariwari group activity "+group+" --cursor"))
	},
}

var groupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group from JSON",
	Long: `Create a group from a JSON body. Requires name, icon, and at least one
participant.

Group creation is session-credential only: a developer API key (wari_...) gets
403 ACCESS_DENIED. Create the group in the app, then point the CLI at it with
--group or "wariwari config set-group TOKEN".

Pass --idempotency-key to make a retry safe for 24 hours; reusing a key with a
different body returns 409 IDEMPOTENCY_KEY_IN_USE.`,
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

var groupExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Print example request payloads",
}

var groupExampleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Example payload for creating a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"name":     "Trip",
			"icon":     "🧳",
			"currency": "JPY",
			"participants": []map[string]any{
				{"name": "Ada"},
				{"name": "Grace"},
			},
		}, "Group create example")
	},
}

var groupExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"name":        "Trip 2026",
			"icon":        "🧳",
			"lockVersion": 1,
		}, "Group update example")
	},
}

var groupUpdateCmd = &cobra.Command{
	Use:   "update [TOKEN]",
	Short: "Update a group from JSON",
	Long: `Update group metadata. lockVersion is required — send the value from the
current snapshot. A stale value returns 409 CONFLICT with the fresh group under
"current". currency can only change while the group has no transactions.`,
	Args: cobra.MaximumNArgs(1),
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
		data, err := api.Patch(cmd.Context(), "/groups/"+group, body)
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

func init() {
	rootCmd.AddCommand(groupCmd)
	groupCmd.AddCommand(groupListCmd, groupShowCmd, groupActivityCmd, groupCreateCmd, groupUpdateCmd, groupExampleCmd)
	groupExampleCmd.AddCommand(groupExampleCreateCmd, groupExampleUpdateCmd)
	groupListCmd.Flags().StringVar(&groupCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	groupListCmd.Flags().IntVar(&groupTake, "take", 0, "Results per page (max 100)")
	groupListCmd.Flags().BoolVar(&groupArchived, "archived", false, "List archived groups instead of the active set")
	groupActivityCmd.Flags().StringVar(&groupCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	groupActivityCmd.Flags().IntVar(&groupTake, "take", 0, "Results per page (max 100)")
	groupCreateCmd.Flags().StringVar(&groupBody, "data", "", "JSON request body")
	groupCreateCmd.Flags().StringVar(&groupBodyFile, "data-file", "", "Read JSON request body from file")
	groupCreateCmd.Flags().StringVar(&groupIdempotencyKey, "idempotency-key", "", "Idempotency-Key header")
	groupUpdateCmd.Flags().StringVar(&groupBody, "data", "", "JSON request body")
	groupUpdateCmd.Flags().StringVar(&groupBodyFile, "data-file", "", "Read JSON request body from file")
}
