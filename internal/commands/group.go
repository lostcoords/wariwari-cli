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
var groupOutput string
var groupFxFrom string
var groupFxDate string
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
	Long: `Fetch a group snapshot: metadata, visibility, shareUrl (the invite link),
categories, participants, transactions, netByCurrency, suggestedPayments and
spending totals. This is the one read of a whole group; every write returns
only the record it changed.

Participants and categories have no list endpoints; they are read from this
payload. For one member's balance worked out transaction by transaction, see
"wariwari participant balance".`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
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
		group, err := groupArg(args, 0)
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
participant; each participant may carry a color. Optional: currency, a client
token, categories (token, emoji, name, position) to replace the defaults, and
creatorParticipantToken naming the participant that is you (you become its
admin). See "wariwari group example create". Returns the group without its
participants; read them with "wariwari group show".

A developer API key may create 20 groups an hour. Pass --idempotency-key (a
UUID) to make a retry safe for 90 days; reusing a key with a different body
returns 409 IDEMPOTENCY_KEY_IN_USE.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		body, err := commandData()
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups", write(body))
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
				{"token": "7gKMjL9pPWsKaHQ4AbwS5kVj", "name": "Ada", "color": "#1F3A5F"},
				{"token": "8mPNkR4qQXtLbJR5CcxT6yWa", "name": "Grace", "color": "#A94E3B"},
			},
			"creatorParticipantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"categories": []map[string]any{
				{"token": "4QfYk8nE2pRtVwXz7aBcDeFg", "emoji": "🍜", "name": "Food", "position": 1},
				{"token": "5RgZm9oF3qSuWxYa8bCdEfGh", "emoji": "🚃", "name": "Transport", "position": 2},
			},
		}, "Group create example")
	},
}

var groupExampleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Example payload for updating a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"name":       "Trip 2026",
			"icon":       "🧳",
			"currency":   "USD",
			"visibility": "public_view_only",
		}, "Group update example")
	},
}

var groupUpdateCmd = &cobra.Command{
	Use:   "update [TOKEN]",
	Short: "Update a group from JSON",
	Long: `Update a group: send only what changes of name, icon, currency and
visibility (public_editable, public_view_only, private_registered). Changing
currency rebases existing transactions in the background. Returns the group.` + guardedUpdateHelp,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, err := commandData()
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, body, groupRecord(group))
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group, w)
		if err != nil {
			return err
		}
		return printOK(data, "Group updated")
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:   "delete [TOKEN]",
	Short: "Soft-delete a group",
	Long: `Soft-delete the group for every member. The API checks the group's
lockVersion: pass --lock-version with the value you read to have a concurrent
edit rejected with 409 CONFLICT, or let the CLI read the current one first.
Deleting an already deleted group succeeds again.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		w, err := revisedWrite(cmd, nil, groupRecord(group))
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group, w); err != nil {
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
	group, err := groupArg(args, 0)
	if err != nil {
		return err
	}
	path := "/groups/" + group + "/archival"
	if archived {
		_, err = api.Post(cmd.Context(), path, write(nil))
	} else {
		_, err = api.Delete(cmd.Context(), path, write(nil))
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
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/stats", map[string]string{
			"dateRange":        groupStatsDateRange,
			"participantToken": groupStatsParticipant,
			"categoryToken":    groupStatsCategory,
			"date":             groupStatsDate,
			"month":            groupStatsMonth,
			"week":             groupStatsWeek,
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
recording it. --date picks the day (default today); provisional is true while
that day's rate is not yet published.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/fx_rate", map[string]string{"from": groupFxFrom, "date": groupFxDate})
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
		group, err := groupArg(args, 0)
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

func init() {
	rootCmd.AddCommand(groupCmd)
	addLockVersionFlag(groupUpdateCmd, groupDeleteCmd)
	addDataFlags(groupCreateCmd, groupUpdateCmd)
	addIdempotencyFlag(groupCreateCmd, groupUpdateCmd, groupDeleteCmd, groupArchiveCmd, groupUnarchiveCmd)
	groupCmd.AddCommand(groupListCmd, groupShowCmd, groupActivityCmd, groupCreateCmd, groupUpdateCmd, groupDeleteCmd,
		groupArchiveCmd, groupUnarchiveCmd, groupStatsCmd, groupFxRateCmd, groupExportCmd, groupExampleCmd)
	groupExampleCmd.AddCommand(groupExampleCreateCmd, groupExampleUpdateCmd)
	groupListCmd.Flags().StringVar(&groupCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	groupListCmd.Flags().IntVar(&groupTake, "take", 0, "Results per page (max 100)")
	groupListCmd.Flags().BoolVar(&groupArchived, "archived", false, "List archived groups instead of the active set")
	groupActivityCmd.Flags().StringVar(&groupCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	groupActivityCmd.Flags().IntVar(&groupTake, "take", 0, "Results per page (max 100)")
	groupStatsCmd.Flags().StringVar(&groupStatsDateRange, "date-range", "", "Statistics window, e.g. this_month")
	groupStatsCmd.Flags().StringVar(&groupStatsParticipant, "participant", "", "Only expenses this participant paid or owes a share of")
	groupStatsCmd.Flags().StringVar(&groupStatsCategory, "category", "", "Only expenses filed under this group category token")
	groupStatsCmd.Flags().StringVar(&groupStatsDate, "date", "", "Only this ISO date")
	groupStatsCmd.Flags().StringVar(&groupStatsMonth, "month", "", "Only this month, YYYY-MM")
	groupStatsCmd.Flags().StringVar(&groupStatsWeek, "week", "", "Only the Monday-start week containing this ISO date")
	groupFxRateCmd.Flags().StringVar(&groupFxFrom, "from", "", "Source currency, ISO-4217 (required)")
	groupFxRateCmd.Flags().StringVar(&groupFxDate, "date", "", "Day the rate is for, usually the expense's date (default today)")
	_ = groupFxRateCmd.MarkFlagRequired("from")
	groupExportCmd.Flags().StringVar(&groupOutput, "output", "", "Write the CSV to this file instead of stdout")
}
