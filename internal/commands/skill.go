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
	Short: "Install the WariWari agent skill for Claude Code, Codex and other agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths := []string{}
		for _, dir := range skillDirs() {
			path, err := writeEmbeddedSkill(dir)
			if err != nil {
				return err
			}
			paths = append(paths, path)
		}
		return printOK(map[string]any{"paths": paths}, "WariWari skill installed")
	},
}

// skillDirs lists where agents look for user skills: the shared ~/.agents
// directory, Claude Code and Codex.
func skillDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return []string{
		filepath.Join(home, ".agents", "skills", "wariwari"),
		filepath.Join(home, ".claude", "skills", "wariwari"),
		codexSkillDir(),
	}
}

func writeEmbeddedSkill(dir string) (string, error) {
	data, err := skills.FS.ReadFile("wariwari/SKILL.md")
	if err != nil {
		return "", err
	}
	// Replace an older symlinked install with a real directory.
	if info, err := os.Lstat(dir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(dir); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, skillFilename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func codexSkillDir() string {
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "skills", "wariwari")
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".codex", "skills", "wariwari")
	}
	return filepath.Join(userHome, ".codex", "skills", "wariwari")
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillInstallCmd)
}
