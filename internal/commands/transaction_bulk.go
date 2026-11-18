package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

const bulkRevisionHelp = `Each item of transactions is {"token", "expectedRevision"}: the lockVersion
you read. Any stale one returns 409 CONFLICT and changes nothing. An item
without expectedRevision is read first and sent at its current lockVersion.

Bulk editing is behind a feature flag; while it is off the API answers 404
NOT_FOUND.`

var transactionBulkEditCmd = &cobra.Command{
	Use:   "bulk-edit [GROUP_TOKEN]",
	Short: "Re-assign payer, category, split or date across many transactions",
	Long: `Apply one change to many transactions, all of them or none. The body names
transactions and any of paidByParticipantToken, categoryToken, date, timeOfDay
and split: {"type": equal|percentage|shares, "members": [{"participantToken",
"weight"}], "newMemberNames": [...]}. Category and split apply to expenses
only. Returns the changed transactions and any participants the split added.
See "wariwari transaction example bulk-edit".

` + bulkRevisionHelp,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, _, err := bulkBody(cmd.Context(), group)
		if err != nil {
			return err
		}
		data, err := api.Patch(cmd.Context(), "/groups/"+group+"/transaction_batch", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Transactions updated")
	},
}

var transactionBulkDeleteCmd = &cobra.Command{
	Use:   "bulk-delete [GROUP_TOKEN]",
	Short: "Soft-delete many transactions",
	Long: `Soft-delete many transactions, all of them or none, from
{"transactions": [{"token": "..."}]}.

` + bulkRevisionHelp,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, tokens, err := bulkBody(cmd.Context(), group)
		if err != nil {
			return err
		}
		if _, err := api.Post(cmd.Context(), "/groups/"+group+"/transaction_deletions", write(body)); err != nil {
			return err
		}
		return printOK(map[string]any{"tokens": tokens, "deleted": true}, fmt.Sprintf("%d transactions deleted", len(tokens)))
	},
}

var transactionExampleBulkEditCmd = &cobra.Command{
	Use:   "bulk-edit",
	Short: "Example payload for bulk-editing transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"transactions": []map[string]any{
				{"token": "9AHTHuAGT7U2CpSfktrfUAYA", "expectedRevision": 2},
				{"token": "3bRtYuKmNpQwErTyUiOpAsDf", "expectedRevision": 0},
			},
			"categoryToken": "4QfYk8nE2pRtVwXz7aBcDeFg",
			"split": map[string]any{
				"type": "shares",
				"members": []map[string]any{
					{"participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj", "weight": 2},
					{"participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa", "weight": 1},
				},
			},
		}, "Transaction bulk-edit example")
	},
}

// bulkBody reads the JSON body and gives every listed transaction the caller
// left without an expectedRevision its current lockVersion. It also returns
// the tokens, in order.
func bulkBody(ctx context.Context, group string) (map[string]any, []string, error) {
	body, err := commandData()
	if err != nil {
		return nil, nil, err
	}
	items, _ := body["transactions"].([]any)
	if len(items) == 0 {
		return nil, nil, fmt.Errorf(`transactions is required, e.g. --data '{"transactions":[{"token":"TOKEN"}]}'`)
	}
	tokens := make([]string, 0, len(items))
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		token, _ := item["token"].(string)
		if token == "" {
			return nil, nil, fmt.Errorf(`each of transactions is {"token": "...", "expectedRevision": N}`)
		}
		tokens = append(tokens, token)
		if _, ok := item["expectedRevision"]; ok {
			continue
		}
		record, err := transactionRecord(group, token)(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", token, err)
		}
		if item["expectedRevision"], err = lockVersion(record); err != nil {
			return nil, nil, err
		}
	}
	return body, tokens, nil
}
