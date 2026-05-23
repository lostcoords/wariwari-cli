package commands

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lostcoords/wariwari-cli/skills"
	"github.com/spf13/cobra"
)

const skillFilename = "SKILL.md"

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Print or install the embedded agent skill",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := skills.FS.ReadFile("wariwari/SKILL.md")
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(data)
		return err
	},
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the WariWari agent skill",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := installSkill()
		if err != nil {
			return err
		}
		return printOK(map[string]any{"skill_path": path}, "WariWari skill installed")
	},
}

func installSkill() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	data, err := skills.FS.ReadFile("wariwari/SKILL.md")
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".agents", "skills", "wariwari")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, skillFilename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func linkSkillToClaude() (string, error) {
	skillPath, err := installSkill()
	if err != nil {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	claudeDir := filepath.Join(home, ".claude", "skills")
	claudeSkillDir := filepath.Join(claudeDir, "wariwari")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return "", err
	}

	_ = os.Remove(claudeSkillDir)
	target := filepath.Join("..", "..", ".agents", "skills", "wariwari")
	if err := os.Symlink(target, claudeSkillDir); err != nil {
		if err := os.MkdirAll(claudeSkillDir, 0o755); err != nil {
			return "", err
		}
		data, readErr := os.ReadFile(skillPath)
		if readErr != nil {
			return "", readErr
		}
		if writeErr := os.WriteFile(filepath.Join(claudeSkillDir, skillFilename), data, 0o644); writeErr != nil {
			return "", writeErr
		}
	}

	return claudeSkillDir, nil
}

func codexSkillPath() string {
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "skills", "wariwari", skillFilename)
	}
	return "~/.codex/skills/wariwari/SKILL.md"
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillInstallCmd)
}
