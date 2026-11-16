package commands

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/spf13/cobra"
)

var transactionReceiptCmd = &cobra.Command{
	Use:   "receipt",
	Short: "Attach or remove an expense's receipt photo (Pro groups)",
}

var transactionReceiptUploadCmd = &cobra.Command{
	Use:   "upload TRANSACTION_TOKEN FILE [GROUP_TOKEN]",
	Short: "Attach a receipt photo, replacing any earlier one",
	Long: `Store one photo of the receipt for an expense, replacing any earlier one.
FILE is a JPEG, PNG, WebP or HEIC image; the server re-encodes it. Returns the
stored photo with a short-lived url. Needs a Pro group; otherwise 403
PRO_REQUIRED.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 2)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		data, err := api.PutFile(cmd.Context(), receiptPath(group, args[0]), client.File{
			Field:       "photo",
			Name:        filepath.Base(args[1]),
			ContentType: imageType(args[1], content),
			Content:     content,
		})
		if err != nil {
			return err
		}
		return printOK(data, "Receipt photo stored")
	},
}

var transactionReceiptDeleteCmd = &cobra.Command{
	Use:     "delete TRANSACTION_TOKEN [GROUP_TOKEN]",
	Aliases: []string{"rm"},
	Short:   "Remove an expense's receipt photo",
	Long:    `Delete the receipt photo. Works without Pro, so a group that lost Pro can still clean up.`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), receiptPath(group, args[0]), client.Write{}); err != nil {
			return err
		}
		return printOK(map[string]any{"transactionToken": args[0], "receiptPhoto": nil}, "Receipt photo removed")
	},
}

func receiptPath(group, transaction string) string {
	return "/groups/" + group + "/transactions/" + transaction + "/receipt_photo"
}

// imageType names the upload's media type. Go sniffs JPEG, PNG and WebP but
// not HEIC, the iPhone camera's format, which goes by its extension.
func imageType(name string, content []byte) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".heic", ".heif":
		return "image/heic"
	}
	return http.DetectContentType(content)
}

func init() {
	transactionCmd.AddCommand(transactionReceiptCmd)
	transactionReceiptCmd.AddCommand(transactionReceiptUploadCmd, transactionReceiptDeleteCmd)
}
