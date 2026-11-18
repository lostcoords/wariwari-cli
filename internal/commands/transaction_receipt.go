package commands

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/spf13/cobra"
)

var transactionReceiptCmd = &cobra.Command{
	Use:   "receipt",
	Short: "Show, attach or remove an expense's receipt photo",
}

var transactionReceiptUploadCmd = &cobra.Command{
	Use:   "upload TRANSACTION_TOKEN FILE [GROUP_TOKEN]",
	Short: "Attach a receipt photo, replacing any earlier one",
	Long: `Store one photo of the receipt for an expense, replacing any earlier one.
FILE is a JPEG, PNG, WebP or HEIC image; the server re-encodes it. Returns the
stored photo with a short-lived url. Needs Pro on your account; otherwise
403 PRO_REQUIRED.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 2)
		if err != nil {
			return err
		}
		file, err := uploadFile("photo", args[1])
		if err != nil {
			return err
		}
		data, err := api.Upload(cmd.Context(), http.MethodPut, receiptPath(group, args[0]), file, write(nil))
		if err != nil {
			return err
		}
		return printOK(data, "Receipt photo stored")
	},
}

var transactionReceiptShowCmd = &cobra.Command{
	Use:     "show TRANSACTION_TOKEN [GROUP_TOKEN]",
	Aliases: []string{"view"},
	Short:   "Show an expense's receipt photo",
	Long: `Show the receipt photo with a short-lived url, or "receiptPhoto": null when
the expense has none; every transaction's hasReceiptPhoto says which. Any
member can read it, with or without Pro.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), receiptPath(group, args[0]), nil)
		if absent, err := absentOn404(cmd.Context(), err, transactionRecord(group, args[0])); absent {
			return printOK(map[string]any{"transactionToken": args[0], "receiptPhoto": nil}, "No receipt photo")
		} else if err != nil {
			return err
		}
		return printOK(data, "Receipt photo")
	},
}

var transactionReceiptDeleteCmd = &cobra.Command{
	Use:     "delete TRANSACTION_TOKEN [GROUP_TOKEN]",
	Aliases: []string{"rm"},
	Short:   "Remove an expense's receipt photo",
	Long:    `Delete the receipt photo. Works without Pro.`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), receiptPath(group, args[0]), write(nil)); err != nil {
			return err
		}
		return printOK(map[string]any{"transactionToken": args[0], "receiptPhoto": nil}, "Receipt photo removed")
	},
}

func receiptPath(group, transaction string) string {
	return "/groups/" + group + "/transactions/" + transaction + "/receipt_photo"
}

// uploadFile reads a file for a multipart upload and names its media type:
// by extension first, then by sniffing. HEIC, the iPhone camera's format, is
// in neither Go's table nor its sniffer.
func uploadFile(field, path string) (client.File, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return client.File{}, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	contentType := mime.TypeByExtension(ext)
	switch {
	case ext == ".heic" || ext == ".heif":
		contentType = "image/heic"
	case contentType == "":
		contentType = http.DetectContentType(content)
	}
	return client.File{Field: field, Name: filepath.Base(path), ContentType: contentType, Content: content}, nil
}

func init() {
	transactionCmd.AddCommand(transactionReceiptCmd)
	transactionReceiptCmd.AddCommand(transactionReceiptShowCmd, transactionReceiptUploadCmd, transactionReceiptDeleteCmd)
	addIdempotencyFlag(transactionReceiptUploadCmd, transactionReceiptDeleteCmd)
}
