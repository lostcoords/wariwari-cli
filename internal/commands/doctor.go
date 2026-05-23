package commands

import "github.com/spf13/cobra"

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run a read-only health check",
	RunE: func(cmd *cobra.Command, args []string) error {
		checks := []map[string]any{
			{"name": "config", "status": "pass", "message": "Configuration loaded"},
			{"name": "token", "status": status(cfg.Token != ""), "message": configured(cfg.Token != "")},
			{"name": "profile", "status": status(cfg.Profile != ""), "message": configured(cfg.Profile != "")},
			{"name": "group", "status": status(cfg.Group != ""), "message": configured(cfg.Group != "")},
			{"name": "api_url", "status": "pass", "message": cfg.APIURL},
		}
		return printOK(map[string]any{"checks": checks}, "WariWari CLI health check")
	},
}

func status(ok bool) string {
	if ok {
		return "pass"
	}
	return "warn"
}

func configured(ok bool) string {
	if ok {
		return "Configured"
	}
	return "Not configured"
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
