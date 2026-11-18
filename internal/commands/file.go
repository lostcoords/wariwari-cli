package commands

import (
	"net/http"

	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:     "file",
	Aliases: []string{"files"},
	Short:   "Manage a group's shared photos and files",
}

var fileCursor string
var fileTake int

var fileListCmd = &cobra.Command{
	Use:     "list [GROUP_TOKEN]",
	Aliases: []string{"ls"},
	Short:   "List a group's photos and files",
	Long: `List a group's photos and files, newest first, with your own storage use
across every group (storage.usedBytes against storage.limitBytes). Any member can list and
download them, with or without Pro. The urls are short-lived.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 0)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/attachments", pagingQuery(fileTake, fileCursor))
		if err != nil {
			return err
		}
		nextCommand := commandTakeFlag("wariwari file list "+group, fileTake)
		return printOK(data, "Files", nextPageCrumb(data, nextCommand+" --cursor"))
	},
}

var fileShowCmd = &cobra.Command{
	Use:     "show FILE_ID [GROUP_TOKEN]",
	Aliases: []string{"view"},
	Short:   "Show one photo or file",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		data, err := api.Get(cmd.Context(), "/groups/"+group+"/attachments/"+args[0], nil)
		if err != nil {
			return err
		}
		return printOK(data, "File")
	},
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload PATH [GROUP_TOKEN]",
	Short: "Share a photo or file in a group",
	Long: `Share a photo (JPEG, PNG, WebP, HEIC, GIF), PDF, text or CSV file of up to
10 MB. Photos other than GIFs are re-encoded to WebP. Everything you upload,
in every group, counts against your 5 GB; a file past that is refused with 413
STORAGE_LIMIT. Needs Pro on your account; otherwise 403 PRO_REQUIRED.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		file, err := uploadFile("file", args[0])
		if err != nil {
			return err
		}
		data, err := api.Upload(cmd.Context(), http.MethodPost, "/groups/"+group+"/attachments", file, write(nil))
		if err != nil {
			return err
		}
		return printOK(data, "File shared")
	},
}

var fileDeleteCmd = &cobra.Command{
	Use:     "delete FILE_ID [GROUP_TOKEN]",
	Aliases: []string{"rm"},
	Short:   "Delete a photo or file",
	Long:    `Delete a photo or file. Works without Pro.`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/attachments/"+args[0], write(nil)); err != nil {
			return err
		}
		return printOK(map[string]any{"id": args[0], "deleted": true}, "File deleted")
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)
	fileCmd.AddCommand(fileListCmd, fileShowCmd, fileUploadCmd, fileDeleteCmd)
	addIdempotencyFlag(fileUploadCmd, fileDeleteCmd)
	fileListCmd.Flags().StringVar(&fileCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	fileListCmd.Flags().IntVar(&fileTake, "take", 0, "Results per page (max 100)")
}
