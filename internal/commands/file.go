package commands

import (
	"net/http"

	"github.com/lostcoords/wariwari-cli/internal/client"
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
	Long: `List a group's photos and files, newest first, with the group's storage use
(storage.usedBytes against storage.limitBytes). Any member can list and
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
10 MB. Photos other than GIFs are re-encoded to WebP. A group holds at most 50
files and 500 MB; a file past that, or past its sponsor's 2 GB, is refused
with 413 STORAGE_LIMIT. Needs a Pro group; otherwise 403 PRO_REQUIRED.`,
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
		data, err := api.Upload(cmd.Context(), http.MethodPost, "/groups/"+group+"/attachments", file)
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
	Long:    `Delete a photo or file. Works without Pro, so a group that lost Pro can still clean up.`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := groupArg(args, 1)
		if err != nil {
			return err
		}
		if _, err := api.Delete(cmd.Context(), "/groups/"+group+"/attachments/"+args[0], client.Write{}); err != nil {
			return err
		}
		return printOK(map[string]any{"id": args[0], "deleted": true}, "File deleted")
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)
	fileCmd.AddCommand(fileListCmd, fileShowCmd, fileUploadCmd, fileDeleteCmd)
	fileListCmd.Flags().StringVar(&fileCursor, "cursor", "", "Pagination cursor from pagination.cursor")
	fileListCmd.Flags().IntVar(&fileTake, "take", 0, "Results per page (max 100)")
}
