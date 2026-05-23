package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		if agent || jsonOutput || quiet {
			return fmt.Errorf("setup requires an interactive terminal")
		}

		apiURL := config.DefaultAPIURL
		token := ""
		profile := cfg.Profile
		group := cfg.Group
		saveGlobal := true

		if profile == "" {
			profile = "default"
		}

		if err := huh.NewInput().
			Title("Profile name").
			Value(&profile).
			Run(); err != nil {
			return nil
		}

		if err := huh.NewInput().
			Title("WariWari API URL").
			Value(&apiURL).
			Run(); err != nil {
			return nil
		}

		if err := huh.NewInput().
			Title("API token").
			Description("Create one from Account -> API keys in WariWari.").
			EchoMode(huh.EchoModePassword).
			Value(&token).
			Run(); err != nil {
			return nil
		}

		if err := huh.NewInput().
			Title("Default group token (optional)").
			Value(&group).
			Run(); err != nil {
			return nil
		}

		if err := huh.NewConfirm().
			Title("Save globally?").
			Affirmative("Global ~/.config/wariwari/config.yaml").
			Negative("Local .wariwari.yaml").
			Value(&saveGlobal).
			Run(); err != nil {
			return nil
		}

		newConfig := &config.Config{Profile: profile, APIURL: apiURL, Group: group}
		if token != "" {
			if _, err := config.SaveToken(profile, token); err != nil {
				return err
			}
		}
		store, _ := config.LoadProfiles()
		if store != nil {
			_ = store.Set(profile, config.Profile{APIURL: apiURL, Group: group})
			store.Default = profile
			_ = store.Save()
		}

		if saveGlobal {
			if err := newConfig.Save(); err != nil {
				return err
			}
		} else {
			newConfig.Token = token
			if err := newConfig.SaveLocal(); err != nil {
				return err
			}
		}

		return printOK(map[string]any{
			"profile": profile,
			"api_url": apiURL,
			"group":   group,
		}, "Configuration saved", crumb("doctor", "wariwari doctor", "Run a health check"))
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.AddCommand(setupClaudeCmd)
}

var setupClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Install the WariWari skill for Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		skillPath, err := installSkill()
		if err != nil {
			return err
		}
		claudePath, err := linkSkillToClaude()
		if err != nil {
			return err
		}

		result := map[string]any{
			"skill_path":        skillPath,
			"claude_skill_path": claudePath,
			"plugin_path":       ".claude-plugin",
		}

		if claude, err := exec.LookPath("claude"); err == nil {
			installCmd := exec.CommandContext(cmd.Context(), claude, "plugin", "install", "./.claude-plugin")
			output, runErr := installCmd.CombinedOutput()
			result["plugin_command"] = "claude plugin install ./.claude-plugin"
			if runErr != nil {
				result["plugin_installed"] = false
				result["plugin_error"] = strings.TrimSpace(string(output))
			} else {
				result["plugin_installed"] = true
			}
		} else {
			result["notice"] = "Claude Code binary not found on PATH; skill link was still installed."
		}

		return printOK(result, "Claude Code linkage installed")
	},
}
