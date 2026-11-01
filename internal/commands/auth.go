package commands

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authNoBrowser bool

var authLoginCmd = &cobra.Command{
	Use:   "login [TOKEN]",
	Short: "Sign in with the browser, or save an API token",
	Long: `With no argument, sign in through the browser: the CLI shows a one-time code,
opens the WariWari approval page, and waits while you approve it. A person has
to approve the login, so scripts and agents should pass an API key instead:

  wariwari auth login wari_...

or set WARIWARI_TOKEN. Either way the key is saved to the active profile.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			profile, storage, err := saveLogin(args[0])
			if err != nil {
				return err
			}
			return printOK(map[string]any{
				"authenticated": true,
				"profile":       profile,
				"storage":       storage,
			}, "Token saved", crumb("status", "wariwari auth status", "Check auth status"))
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()
		login, err := browserLogin(ctx, cfg.APIURL, canOpenBrowser(authNoBrowser))
		if err != nil {
			return err
		}
		profile, storage, err := saveLogin(login.Token)
		if err != nil {
			return err
		}
		user := userLabel(login.User)
		fmt.Fprintf(errOut, "Logged in as %s.\n", user)
		return printOK(map[string]any{
			"authenticated": true,
			"profile":       profile,
			"api_url":       cfg.APIURL,
			"user":          user,
			"storage":       storage,
			"key_hint":      "This login created an API key; see or revoke it under Account -> API keys.",
		}, "Logged in as "+user, crumb("status", "wariwari auth status", "Check auth status"))
	},
}

// saveLogin stores token for the active profile (default "default") and makes
// that profile, API URL and group the global default.
func saveLogin(token string) (string, string, error) {
	profile := cfg.Profile
	if profile == "" {
		profile = "default"
		cfg.Profile = profile
	}
	storage, err := config.SaveToken(profile, token)
	if err != nil {
		return "", "", err
	}
	cfg.Token = token
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
	return profile, storage, nil
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
	authLoginCmd.Flags().BoolVar(&authNoBrowser, "no-browser", false, "Print the approval URL instead of opening a browser")
}
