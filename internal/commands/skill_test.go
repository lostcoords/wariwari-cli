package commands

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/lostcoords/wariwari-cli/skills"
)

func TestEmbeddedSkillDocumentsCurrentContract(t *testing.T) {
	data, err := skills.FS.ReadFile("wariwari/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	skill := strings.Join(strings.Fields(string(data)), " ")
	for _, want := range []string{
		"Positive means that participant is owed money; negative means they owe.",
		"Transaction reads always include `timeOfDay`, but it can be `null`",
		"developer keys have no monthly usage quota",
		"`AUTH_CREDENTIALS_EXPIRED`",
		"`INTERNAL_SERVER_EXCEPTION`",
		"`QUOTA_EXCEEDED`",
		"as `--lock-version` for deletes",
	} {
		if !strings.Contains(skill, want) {
			t.Fatalf("embedded skill is missing %q", want)
		}
	}
	if strings.Contains(skill, "free key quota is 50 calls/month") {
		t.Fatal("embedded skill still documents the inactive monthly quota")
	}
}

// Every `wariwari ...` invocation in the skill must name a real command, so an
// agent following it never hits "unknown command".
func TestEmbeddedSkillCommandsExist(t *testing.T) {
	data, err := skills.FS.ReadFile("wariwari/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "wariwari ") {
			continue
		}
		var path []string
		for _, word := range strings.Fields(line)[1:] {
			if strings.HasPrefix(word, "-") || strings.ToUpper(word) == word || strings.ContainsAny(word, "\"'#<") {
				break
			}
			path = append(path, word)
		}
		if len(path) == 0 {
			continue
		}
		cmd, rest, err := rootCmd.Find(path)
		if err != nil || cmd == rootCmd {
			t.Errorf("skill line %q names no command: %v", line, err)
			continue
		}
		if len(rest) > 0 && cmd.HasSubCommands() && (cmd.Args == nil || cmd.Args(cmd, rest) != nil) {
			t.Errorf("skill line %q: %q is not a subcommand of %q", line, rest[0], cmd.CommandPath())
		}
	}
}

func TestSkillInstallWritesEveryAgentDirectory(t *testing.T) {
	home := t.TempDir()
	codexHome := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", codexHome)

	// An older `setup claude` left ~/.claude/skills/wariwari as a symlink.
	claudeDir := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(claudeDir, "wariwari")); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := runRootCommand(t, &config.Config{Token: "wari_test"}, http.NotFound, "skill", "install"); err != nil {
		t.Fatalf("skill install: %v", err)
	}

	for _, path := range []string{
		filepath.Join(home, ".agents", "skills", "wariwari", "SKILL.md"),
		filepath.Join(home, ".claude", "skills", "wariwari", "SKILL.md"),
		filepath.Join(codexHome, "skills", "wariwari", "SKILL.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("skill missing at %s: %v", path, err)
		}
	}
	if info, _ := os.Lstat(filepath.Join(claudeDir, "wariwari")); info.Mode()&os.ModeSymlink != 0 {
		t.Error("Claude skill directory is still a symlink")
	}
}
