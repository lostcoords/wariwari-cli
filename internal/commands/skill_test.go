package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	} {
		if !strings.Contains(skill, want) {
			t.Fatalf("embedded skill is missing %q", want)
		}
	}
	if strings.Contains(skill, "free key quota is 50 calls/month") {
		t.Fatal("embedded skill still documents the inactive monthly quota")
	}
}

func TestInstallSkillAlsoInstallsCodexSkill(t *testing.T) {
	home := t.TempDir()
	codexHome := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", codexHome)

	agentsPath, err := installSkill()
	if err != nil {
		t.Fatalf("installSkill() error = %v", err)
	}
	codexPath, err := installCodexSkill()
	if err != nil {
		t.Fatalf("installCodexSkill() error = %v", err)
	}

	if _, err := os.Stat(agentsPath); err != nil {
		t.Fatalf("agents skill missing: %v", err)
	}
	wantCodexPath := filepath.Join(codexHome, "skills", "wariwari", "SKILL.md")
	if codexPath != wantCodexPath {
		t.Fatalf("codexPath = %q, want %q", codexPath, wantCodexPath)
	}
	if _, err := os.Stat(codexPath); err != nil {
		t.Fatalf("codex skill missing: %v", err)
	}
}
