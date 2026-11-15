package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var transactionBulkEditCmd = &cobra.Command{
	Use:   "bulk-edit [GROUP_TOKEN]",
	Short: "Re-assign payer, category, split or date across many transactions",
	Long: `Apply one change to many transactions. The body names transactionTokens and
any of paidByToken, groupCategoryToken, splitType with splitParticipantTokens
(plus splitPercentages or splitShares), date, timeOfDay.

lockVersions maps each token to the lockVersion you read; a stale one returns
409 CONFLICT. Tokens missing from lockVersions are read first and sent at
their current lockVersion. See "wariwari transaction example bulk-edit".

Bulk editing is behind a feature flag; while it is off for your account the
API answers 404 NOT_FOUND.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, err := bulkBody(cmd.Context(), group)
		if err != nil {
			return err
		}
		data, err := api.Post(cmd.Context(), "/groups/"+group+"/transaction_bulk_edit", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Transactions updated")
	},
}

var transactionBulkDeleteCmd = &cobra.Command{
	Use:   "bulk-delete [GROUP_TOKEN]",
	Short: "Soft-delete many transactions",
	Long: `Soft-delete many transactions from {"transactionTokens": [...]}.

lockVersions maps each token to the lockVersion you read; tokens missing from
it are read first and sent at their current lockVersion.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		body, err := bulkBody(cmd.Context(), group)
		if err != nil {
			return err
		}
		data, err := api.Delete(cmd.Context(), "/groups/"+group+"/transaction_bulk_edit", write(body))
		if err != nil {
			return err
		}
		return printOK(data, "Transactions deleted")
	},
}

var transactionExampleBulkEditCmd = &cobra.Command{
	Use:   "bulk-edit",
	Short: "Example payload for bulk-editing transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"transactionTokens":  []string{"9AHTHuAGT7U2CpSfktrfUAYA", "3bRtYuKmNpQwErTyUiOpAsDf"},
			"lockVersions":       map[string]int{"9AHTHuAGT7U2CpSfktrfUAYA": 2, "3bRtYuKmNpQwErTyUiOpAsDf": 0},
			"groupCategoryToken": "4QfYk8nE2pRtVwXz7aBcDeFg",
			"splitType":          "shares",
			"splitParticipantTokens": []string{
				"7gKMjL9pPWsKaHQ4AbwS5kVj", "8mPNkR4qQXtLbJR5CcxT6yWa",
			},
			"splitShares": map[string]int{"7gKMjL9pPWsKaHQ4AbwS5kVj": 2, "8mPNkR4qQXtLbJR5CcxT6yWa": 1},
		}, "Transaction bulk-edit example")
	},
}

// bulkBody reads the JSON body and fills lockVersions for any listed token
// the caller left out, from that transaction's current record.
func bulkBody(ctx context.Context, group string) (map[string]any, error) {
	body, err := commandData()
	if err != nil {
		return nil, err
	}
	tokens, _ := body["transactionTokens"].([]any)
	if len(tokens) == 0 {
		return nil, fmt.Errorf(`transactionTokens is required, e.g. --data '{"transactionTokens":["TOKEN"]}'`)
	}
	versions, _ := body["lockVersions"].(map[string]any)
	if versions == nil {
		versions = map[string]any{}
	}
	for _, raw := range tokens {
		token, _ := raw.(string)
		if _, ok := versions[token]; ok || token == "" {
			continue
		}
		record, err := transactionRecord(group, token)(ctx)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", token, err)
		}
		revision, err := lockVersion(record)
		if err != nil {
			return nil, err
		}
		versions[token] = revision
	}
	body["lockVersions"] = versions
	return body, nil
}
