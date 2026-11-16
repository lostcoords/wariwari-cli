package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var recurringCmd = &cobra.Command{
	Use:   "recurring",
	Short: "Manage expenses posted on a schedule (Pro)",
}

var (
	recurringEvery    string
	recurringInterval int
	recurringStarts   string
)

var recurringListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List a group's recurring expenses",
	Long: `List a group's recurring expenses, paused ones included. A schedule is
paused while the member who set it up has no Pro or has left the group:
occurrences are skipped, not saved for later.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/recurring_expenses", nil)
		if err != nil {
			return err
		}
		return printOK(data, "Recurring expenses",
			crumb("create", "wariwari recurring create "+group+" --data-file recurring.json --every monthly --starts YYYY-MM-DD", "Schedule another"))
	},
}

var recurringShowCmd = &cobra.Command{
	Use:     "show RECURRING_TOKEN [GROUP_TOKEN]",
	Aliases: []string{"view"},
	Short:   "Show a recurring expense",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/recurring_expenses/"+args[0], nil)
		if err != nil {
			return err
		}
		return printOK(data, "Recurring expense")
	},
}

var recurringCreateCmd = &cobra.Command{
	Use:     "create [GROUP_TOKEN]",
	Aliases: []string{"add"},
	Short:   "Schedule a recurring expense",
	Long: `Schedule an expense that wariwari posts on every occurrence. The JSON body
takes the expense fields of "transaction create" (description, amountCents,
currency, paidByToken, splitType, splits, and optionally category or
groupCategoryToken), without date, timeOfDay or kind. The schedule comes from
flags, which override the same fields in the body:

  --every weekly|monthly|yearly   cadence (required unless the body has it)
  --interval N                    every N weeks, months or years (default 1)
  --starts YYYY-MM-DD             first occurrence (required unless the body has it)

Needs Pro on your account; otherwise 403 PRO_REQUIRED. Retry safely with
--idempotency-key and the same body.`,
	Example: `  wariwari recurring create GROUP_TOKEN --data-file rent.json --every monthly --starts 2026-12-01
  wariwari recurring example create --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, err := recurringBody(cmd)
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/recurring_expenses", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Recurring expense created")
	},
}

var recurringUpdateCmd = &cobra.Command{
	Use:   "update RECURRING_TOKEN [GROUP_TOKEN]",
	Short: "Change a recurring expense",
	Long: `Change a schedule: send only the fields that change, as JSON, flags, or both.
Sending splits replaces them all. Expenses it already posted stay as they are.
Needs Pro on your account; otherwise 403 PRO_REQUIRED.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		body, err := recurringBody(cmd)
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/recurring_expenses/"+args[0], write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Recurring expense updated")
	},
}

var recurringDeleteCmd = &cobra.Command{
	Use:     "delete RECURRING_TOKEN [GROUP_TOKEN]",
	Aliases: []string{"rm"},
	Short:   "Stop a recurring expense",
	Long:    `Stop the schedule. Expenses it already posted stay in the group.`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/recurring_expenses/"+args[0], write(nil)); err != nil {
			return err
		}
		return printOK(map[string]any{"token": args[0], "deleted": true}, "Recurring expense deleted")
	},
}

var recurringExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Print example request payloads",
}

var recurringExampleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Example payload for scheduling a recurring expense",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"description": "Rent",
			"amountCents": 180000,
			"currency":    "JPY",
			"paidByToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
			"splitType":   "equal",
			"splits": []map[string]any{
				{"participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj"},
				{"participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa"},
			},
			"cadence":  "monthly",
			"interval": 1,
			"startsOn": "2026-12-01",
		}, "Recurring expense create example")
	},
}

// recurringBody reads the JSON body and lays the schedule flags over it.
func recurringBody(cmd *cobra.Command) (map[string]any, error) {
	body, err := commandData()
	if err != nil {
		return nil, err
	}
	if cmd.Flags().Changed("every") {
		switch recurringEvery {
		case "weekly", "monthly", "yearly":
			body["cadence"] = recurringEvery
		default:
			return nil, fmt.Errorf("--every must be weekly, monthly or yearly")
		}
	}
	if cmd.Flags().Changed("interval") {
		body["interval"] = recurringInterval
	}
	if cmd.Flags().Changed("starts") {
		body["startsOn"] = recurringStarts
	}
	return body, nil
}

func init() {
	rootCmd.AddCommand(recurringCmd)
	recurringCmd.AddCommand(recurringListCmd, recurringShowCmd, recurringCreateCmd, recurringUpdateCmd, recurringDeleteCmd, recurringExampleCmd)
	recurringExampleCmd.AddCommand(recurringExampleCreateCmd)
	addDataFlags(recurringCreateCmd, recurringUpdateCmd)
	addIdempotencyFlag(recurringCreateCmd, recurringUpdateCmd, recurringDeleteCmd)
	for _, cmd := range []*cobra.Command{recurringCreateCmd, recurringUpdateCmd} {
		cmd.Flags().StringVar(&recurringEvery, "every", "", "Cadence: weekly, monthly or yearly")
		cmd.Flags().IntVar(&recurringInterval, "interval", 1, "Every N weeks, months or years")
		cmd.Flags().StringVar(&recurringStarts, "starts", "", "First occurrence, YYYY-MM-DD")
	}
}
