package commands

import (
	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and update configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"profile": cfg.Profile,
			"api_url": cfg.APIURL,
			"group":   cfg.Group,
			"token":   map[string]any{"configured": cfg.Token != ""},
		}, "Effective configuration")
	},
}

var configExplainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain configuration precedence",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printOK(map[string]any{
			"precedence": []string{
				"CLI flags (--token, --profile, --api-url, --group)",
				"Environment variables (WARIWARI_TOKEN, WARIWARI_PROFILE, WARIWARI_API_URL, WARIWARI_GROUP)",
				"Named profile settings (~/.config/wariwari/config.json)",
				"Local project config (.wariwari.yaml)",
				"Global config (~/.config/wariwari/config.yaml or ~/.wariwari/config.yaml)",
				"Defaults",
			},
			"global_paths": config.GlobalConfigPaths(),
			"local_path":   config.LocalConfigPath(),
		}, "Configuration precedence")
	},
}

var configSetGroupCmd = &cobra.Command{
	Use:   "set-group TOKEN",
	Short: "Save a default group token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Group = args[0]
		global := config.LoadGlobal()
		global.Profile = cfg.Profile
		global.APIURL = cfg.APIURL
		global.Group = cfg.Group
		if err := global.Save(); err != nil {
			return err
		}
		store, _ := config.LoadProfiles()
		if store != nil && cfg.Profile != "" {
			_ = store.Set(cfg.Profile, config.Profile{APIURL: cfg.APIURL, Group: cfg.Group})
		}
		return printOK(map[string]any{"group": cfg.Group}, "Default group saved")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd, configExplainCmd, configSetGroupCmd)
}
