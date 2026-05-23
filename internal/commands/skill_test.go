package commands

import (
	"os"
	"path/filepath"
	"testing"
)

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
