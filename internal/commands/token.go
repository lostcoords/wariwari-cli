package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// API-key management is deliberately web-only — there is no /v1 route to list,
// create or revoke a key. This command stays so the old `token list|create|
// delete` invocations fail with directions instead of "unknown command".
var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Explain where WariWari API keys are managed",
	Long: `WariWari API keys are managed on the web account page, not through the API.
There is no /v1 endpoint to list, create, or revoke a key.

Create a key at https://wariwari.jp/account (Account -> API keys). It is shown
once; save it with "wariwari auth login TOKEN". "wariwari auth login" with no
token creates one for this machine through the browser.

Note that a developer key (wari_...) cannot create groups — POST /v1/groups is
session-credential only and answers 403 ACCESS_DENIED.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("API keys are managed on the web account page (https://wariwari.jp/account), not through the API; save one with `wariwari auth login TOKEN`")
	},
}

func init() {
	// Tolerate the flags the removed subcommands took (e.g. `token create
	// --name X`) so those callers get the explanation, not a flag error.
	tokenCmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	rootCmd.AddCommand(tokenCmd)
}
