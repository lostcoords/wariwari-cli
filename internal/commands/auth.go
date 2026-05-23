package commands

import (
	"fmt"

	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login TOKEN",
	Short: "Save an API token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile := cfg.Profile
		if profile == "" {
			profile = "default"
			cfg.Profile = profile
		}
		storage, err := config.SaveToken(profile, args[0])
		if err != nil {
			return err
		}
		store, _ := config.LoadProfiles()
		if store != nil {
			_ = store.Set(profile, config.Profile{APIURL: cfg.APIURL, Group: cfg.Group})
			store.Default = profile
			_ = store.Save()
		}
		global := config.LoadGlobal()
		global.Profile = profile
		global.APIURL = cfg.APIURL
		global.Group = cfg.Group
		global.Token = ""
		_ = global.Save()
		return printOK(map[string]any{
			"authenticated": true,
			"profile":       profile,
			"storage":       storage,
		}, "Token saved", crumb("status", "wariwari auth status", "Check auth status"))
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, source := config.LoadToken(cfg.Profile)
		return printOK(map[string]any{
			"authenticated": cfg.Token != "",
			"profile":       cfg.Profile,
			"api_url":       cfg.APIURL,
			"group":         cfg.Group,
			"token_source":  source,
		}, "Authentication status")
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials for the active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Profile == "" {
			return fmt.Errorf("no active profile")
		}
		config.DeleteToken(cfg.Profile)
		return printOK(map[string]any{
			"authenticated": false,
			"profile":       cfg.Profile,
		}, "Logged out")
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := config.LoadProfiles()
		if err != nil {
			return err
		}
		return printOK(map[string]any{
			"default":  store.Default,
			"profiles": store.Profiles,
		}, "Profiles")
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd, authListCmd)
}
