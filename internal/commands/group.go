package commands

import (
	"os"

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
var groupOutput string
var groupFxFrom string
var groupStatsDateRange string
var groupStatsParticipant string
var groupStatsCategory string
var groupStatsDate string
var groupStatsMonth string
var groupStatsWeek string

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
		nextCommand := "wariwari group list"
		if groupArchived {
			nextCommand += " --archived"
		}
		nextCommand = commandTakeFlag(nextCommand, groupTake)
		return printOK(data, "Groups",
			crumb("show", "wariwari group show TOKEN", "Fetch the full group snapshot"),
			nextPageCrumb(data, nextCommand+" --cursor"))
	},
}

var groupShowCmd = &cobra.Command{
	Use:     "show [TOKEN]",
	Aliases: []string{"view", "snapshot"},
	Short:   "Fetch the full group snapshot",
	Long: `Fetch a group snapshot: metadata, shareUrl (the invite link),
groupCategories, participants, transactions, netByCurrency, suggestedPayments
and spending totals.

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
		nextCommand := commandTakeFlag("wariwari group activity "+group, groupTake)
		return printOK(data, "Group activity",
			nextPageCrumb(data, nextCommand+" --cursor"))
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

Pass --idempotency-key to make a retry safe for 90 days; reusing a key with a
different body returns 409 IDEMPOTENCY_KEY_IN_USE.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		body, err := bodyFrom(groupBody, groupBodyFile)
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
"current". Changing currency rebases existing transactions in the background.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		body, err := bodyFrom(groupBody, groupBodyFile)
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

var groupDeleteCmd = &cobra.Command{
	Use:   "delete [TOKEN]",
	Short: "Soft-delete a group",
	Long: `Soft-delete the group for every member. --lock-version is required: pass
the group's lockVersion from the snapshot. A stale value returns 409 CONFLICT.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		if _, err := api.DeleteRevision(cmd.Context(), "/groups/"+group, lockVersion); err != nil {
			return err
		}
		return printOK(map[string]any{"token": group, "deleted": true}, "Group deleted")
	},
}

var groupArchiveCmd = &cobra.Command{
	Use:   "archive [TOKEN]",
	Short: "Archive a group for yourself",
	Long:  "Move the group into your archive (group list --archived). Only your view changes; other members are unaffected.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setArchived(cmd, args, true)
	},
}

var groupUnarchiveCmd = &cobra.Command{
	Use:   "unarchive [TOKEN]",
	Short: "Move a group back out of your archive",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setArchived(cmd, args, false)
	},
}

func setArchived(cmd *cobra.Command, args []string, archived bool) error {
	if err := requireAuth(); err != nil {
		return err
	}
	group, err := requireGroup(firstArg(args))
	if err != nil {
		return err
	}
	path := "/groups/" + group + "/archival"
	if archived {
		_, err = api.Post(cmd.Context(), path, nil, "")
	} else {
		_, err = api.Delete(cmd.Context(), path)
	}
	if err != nil {
		return err
	}
	summary := "Group unarchived"
	if archived {
		summary = "Group archived"
	}
	return printOK(map[string]any{"token": group, "archived": archived}, summary)
}

var groupStatsCmd = &cobra.Command{
	Use:   "stats [TOKEN]",
	Short: "Show spending statistics",
	Long: `Show spending statistics in the group currency's minor units: totals,
categoryBreakdown, perPersonCents, a timeline, and memberTotals.

  --date-range   all_time (default), this_month, last_month, last_3_months,
                 last_6_months, last_12_weeks, this_year
  --participant  participant token; expenses they paid or owe a share of
  --category     group category token
  --month        one month, YYYY-MM
  --week         the Monday-start week containing this ISO date
  --date         one ISO date`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/stats", map[string]string{
			"dateRange":   groupStatsDateRange,
			"participant": groupStatsParticipant,
			"category":    groupStatsCategory,
			"date":        groupStatsDate,
			"month":       groupStatsMonth,
			"week":        groupStatsWeek,
		})
		if err != nil {
			return err
		}
		return printOK(data, "Group stats")
	},
}

var groupFxRateCmd = &cobra.Command{
	Use:   "fx-rate [TOKEN] --from CURRENCY",
	Short: "Show the rate from a currency to the group currency",
	Long: `Show the exchange rate from --from (ISO-4217) to the group currency. rate
is a decimal string. Use it to convert a foreign-currency amount before
recording it.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/fx_rate", map[string]string{"from": groupFxFrom})
		if err != nil {
			return err
		}
		return printOK(data, "Exchange rate")
	},
}

var groupExportCmd = &cobra.Command{
	Use:   "export [TOKEN]",
	Short: "Export the group's transactions as CSV",
	Long: `Download every transaction as a wariwari.v1 CSV, the same file the web app
exports and the importers accept. Writes the CSV to stdout, or to --output.
Needs settings access to the group.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		csv, err := api.GetRaw(cmd.Context(), "/groups/"+group+"/csv_export", "text/csv")
		if err != nil {
			return err
		}
		if groupOutput == "" {
			_, err = out.Write(csv)
			return err
		}
		if err := os.WriteFile(groupOutput, csv, 0o644); err != nil {
			return err
		}
		return printOK(map[string]any{"path": groupOutput, "bytes": len(csv)}, "CSV export written")
	},
}

var groupSettingsCmd = &cobra.Command{
	Use:   "settings [TOKEN]",
	Short: "Show group settings",
	Long: `Show the editable settings: currency, visibility, categories (with a used
flag), and participants with their membershipRole. Needs settings access.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/settings", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Group settings",
			crumb("update", "wariwari group settings update "+group+" --data '{\"lockVersion\":N}'", "Change currency, visibility, categories or roles"))
	},
}

var groupSettingsUpdateCmd = &cobra.Command{
	Use:   "update [TOKEN]",
	Short: "Update group settings from JSON",
	Long: `Update currency, visibility, category edits, and participant roles.
lockVersion (the group's) is required.

  currency       ISO-4217; existing transactions are rebased in the background
  visibility     public_editable, public_view_only, private_registered
  categories     [{"token", "name", "emoji", "position"}], existing categories by token
  participants   [{"token", "membershipRole"}], role is read_only, editor or
                 admin; only participants linked to an account have a role

Returns the refreshed group snapshot.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		group, err := requireGroup(firstArg(args))
		if err != nil {
			return err
		}
		body, err := bodyFrom(groupBody, groupBodyFile)
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/settings", body)
		if err != nil {
			return err
		}
		return printOK(data, "Group settings updated")
	},
}

func init() {
	rootCmd.AddCommand(groupCmd)
	addLockVersionFlag(groupDeleteCmd)
	groupCmd.AddCommand(groupListCmd, groupShowCmd, groupActivityCmd, groupCreateCmd, groupUpdateCmd, groupDeleteCmd,
		groupArchiveCmd, groupUnarchiveCmd, groupStatsCmd, groupFxRateCmd, groupExportCmd, groupSettingsCmd, groupExampleCmd)
	groupSettingsCmd.AddCommand(groupSettingsUpdateCmd)
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
	groupSettingsUpdateCmd.Flags().StringVar(&groupBody, "data", "", "JSON request body")
	groupSettingsUpdateCmd.Flags().StringVar(&groupBodyFile, "data-file", "", "Read JSON request body from file")
	groupStatsCmd.Flags().StringVar(&groupStatsDateRange, "date-range", "", "Statistics window, e.g. this_month")
	groupStatsCmd.Flags().StringVar(&groupStatsParticipant, "participant", "", "Only expenses this participant paid or owes a share of")
	groupStatsCmd.Flags().StringVar(&groupStatsCategory, "category", "", "Only expenses filed under this group category token")
	groupStatsCmd.Flags().StringVar(&groupStatsDate, "date", "", "Only this ISO date")
	groupStatsCmd.Flags().StringVar(&groupStatsMonth, "month", "", "Only this month, YYYY-MM")
	groupStatsCmd.Flags().StringVar(&groupStatsWeek, "week", "", "Only the Monday-start week containing this ISO date")
	groupFxRateCmd.Flags().StringVar(&groupFxFrom, "from", "", "Source currency, ISO-4217 (required)")
	_ = groupFxRateCmd.MarkFlagRequired("from")
	groupExportCmd.Flags().StringVar(&groupOutput, "output", "", "Write the CSV to this file instead of stdout")
}
