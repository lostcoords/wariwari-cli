package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

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

		if cfg.Token == "" {
			checks = append(checks,
				map[string]any{"name": "auth_check", "status": "warn", "message": "No token configured; skipping live API check"},
			)
			return printOK(map[string]any{"checks": checks}, "WariWari CLI health check")
		}

		me, err := api.Get(cmd.Context(), "/account", nil)
		if err != nil {
			checks = append(checks,
				map[string]any{"name": "auth_check", "status": "fail", "message": fmt.Sprintf("Live API auth check failed: %v", err)},
			)
		} else {
			checks = append(checks,
				map[string]any{"name": "auth_check", "status": "pass", "message": liveAuthMessage(me)},
			)
		}

		if cfg.Group != "" {
			if _, err := api.Get(cmd.Context(), "/groups/"+cfg.Group, nil); err != nil {
				checks = append(checks,
					map[string]any{"name": "group_check", "status": "warn", "message": fmt.Sprintf("Default group check failed: %v", err)},
				)
			} else {
				checks = append(checks,
					map[string]any{"name": "group_check", "status": "pass", "message": "Default group is reachable"},
				)
			}
		}
		return printOK(map[string]any{"checks": checks}, "WariWari CLI health check")
	},
}

func liveAuthMessage(me any) string {
	body, ok := me.(map[string]any)
	if !ok {
		return "Authenticated"
	}
	for _, key := range []string{"emailAddress", "name"} {
		if value, ok := body[key].(string); ok && value != "" {
			return fmt.Sprintf("Authenticated as %s", value)
		}
	}
	return "Authenticated"
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
