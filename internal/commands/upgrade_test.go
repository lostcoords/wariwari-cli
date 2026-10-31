package commands

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/config"
)

// fakeRelease serves checksums.txt and one archive whose wariwari binary is a
// shell script that records how it was invoked.
type fakeRelease struct {
	asset     string
	archive   []byte
	checksums string
}

func newFakeRelease(t *testing.T, releaseVersion, script string) fakeRelease {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{"README.md": "readme", "wariwari": script} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	asset := fmt.Sprintf("wariwari_%s_%s_%s.tar.gz", releaseVersion, runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256(buf.Bytes())
	other := fmt.Sprintf("wariwari_%s_plan9_mips.tar.gz", releaseVersion)
	return fakeRelease{
		asset:     asset,
		archive:   buf.Bytes(),
		checksums: fmt.Sprintf("%s  %s\n%s  %s\n", strings.Repeat("0", 64), other, hex.EncodeToString(sum[:]), asset),
	}
}

// setupUpgrade points upgrade at the fake release and at a throwaway
// "installed" binary, and returns that binary's path.
func setupUpgrade(t *testing.T, dir string) string {
	t.Helper()
	exe := filepath.Join(dir, "wariwari")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	previousExecutable, previousVersion := executable, version
	executable = func() (string, error) { return exe, nil }
	version = "0.1.0"
	t.Cleanup(func() { executable, version = previousExecutable, previousVersion })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	return exe
}

// withReleaseServer serves the release and points WARIWARI_RELEASE_URL at it.
// It returns the paths requested so far.
func withReleaseServer(t *testing.T, release fakeRelease) *[]string {
	t.Helper()
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		switch filepath.Base(r.URL.Path) {
		case "checksums.txt":
			_, _ = w.Write([]byte(release.checksums))
		case release.asset:
			_, _ = w.Write(release.archive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("WARIWARI_RELEASE_URL", server.URL+"/download/v-test")
	return &requested
}

// runUpgradeDirect runs `wariwari upgrade`; the API server is unused.
func runUpgradeDirect(t *testing.T, args ...string) (string, string, []requestRecord, error) {
	t.Helper()
	cfg := &config.Config{APIURL: "http://example.invalid"}
	return runRootCommand(t, cfg, okJSON, append([]string{"upgrade"}, args...)...)
}

func TestUpgradeCheckOnlyReports(t *testing.T) {
	exe := setupUpgrade(t, t.TempDir())
	release := newFakeRelease(t, "0.2.0", "#!/bin/sh\n")
	withReleaseServer(t, release)

	stdout, _, _, err := runUpgradeDirect(t, "--check", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{`"latest": "0.2.0"`, `"updateAvailable": true`, "run wariwari upgrade"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("output missing %s:\n%s", want, stdout)
		}
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("--check replaced the binary: %q", got)
	}
}

func TestUpgradeReplacesTheBinaryAndRefreshesTheSkill(t *testing.T) {
	dir := t.TempDir()
	exe := setupUpgrade(t, dir)
	marker := filepath.Join(dir, "invoked")
	release := newFakeRelease(t, "0.2.0", "#!/bin/sh\necho \"$@\" > "+marker+"\n")
	withReleaseServer(t, release)

	home, _ := os.UserHomeDir()
	skillDir := filepath.Join(home, ".agents", "skills", "wariwari")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("old skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, _, err := runUpgradeDirect(t, "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout, "Upgraded wariwari 0.1.0 -> 0.2.0") || !strings.Contains(stdout, `"skillRefreshed": true`) {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
	got, _ := os.ReadFile(exe)
	if !strings.HasPrefix(string(got), "#!/bin/sh") {
		t.Fatalf("binary not replaced: %q", got)
	}
	if info, _ := os.Stat(exe); info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
	if invoked, _ := os.ReadFile(marker); strings.TrimSpace(string(invoked)) != "skill install --quiet" {
		t.Fatalf("new binary invoked with %q, want skill install --quiet", invoked)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(dir, ".wariwari-upgrade-*")); len(leftovers) != 0 {
		t.Fatalf("temp files left behind: %v", leftovers)
	}
}

func TestUpgradeSkipsTheSkillWhenNotInstalled(t *testing.T) {
	dir := t.TempDir()
	setupUpgrade(t, dir)
	marker := filepath.Join(dir, "invoked")
	withReleaseServer(t, newFakeRelease(t, "0.2.0", "#!/bin/sh\necho \"$@\" > "+marker+"\n"))

	stdout, _, _, err := runUpgradeDirect(t, "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout, `"skillRefreshed": false`) {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("new binary ran although no skill was installed")
	}
}

func TestUpgradeRejectsAChecksumMismatch(t *testing.T) {
	exe := setupUpgrade(t, t.TempDir())
	release := newFakeRelease(t, "0.2.0", "#!/bin/sh\n")
	release.checksums = strings.Replace(release.checksums, hexSum(release.archive), strings.Repeat("f", 64), 1)
	withReleaseServer(t, release)

	_, _, _, err := runUpgradeDirect(t)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %v, want a checksum mismatch", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("binary changed after a failed upgrade: %q", got)
	}
}

func TestUpgradeIsANoOpWhenUpToDate(t *testing.T) {
	exe := setupUpgrade(t, t.TempDir())
	requests := withReleaseServer(t, newFakeRelease(t, "0.1.0", "#!/bin/sh\n"))

	stdout, _, _, err := runUpgradeDirect(t, "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout, "wariwari is up to date") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
	if len(*requests) != 1 {
		t.Fatalf("expected only checksums.txt to be fetched, got %v", *requests)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("binary changed: %q", got)
	}
}

func TestUpgradePinsAVersion(t *testing.T) {
	if got := upgradeReleaseURL("v0.3.0"); got != releasesURL+"/download/v0.3.0" {
		t.Fatalf("pinned URL = %s", got)
	}
	if got := upgradeReleaseURL("0.3.0"); got != releasesURL+"/download/v0.3.0" {
		t.Fatalf("pinned URL without v = %s", got)
	}
	if got := upgradeReleaseURL(""); got != releasesURL+"/latest/download" {
		t.Fatalf("latest URL = %s", got)
	}
}

func TestUpgradeDefersToHomebrew(t *testing.T) {
	cellar := filepath.Join(t.TempDir(), "Cellar", "wariwari", "0.1.0", "bin")
	if err := os.MkdirAll(cellar, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := setupUpgrade(t, cellar)
	withReleaseServer(t, newFakeRelease(t, "0.2.0", "#!/bin/sh\n"))

	brewDir := t.TempDir()
	marker := filepath.Join(brewDir, "brew-args")
	if err := os.WriteFile(filepath.Join(brewDir, "brew"), []byte("#!/bin/sh\necho \"$@\" > "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", brewDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, _, _, err := runUpgradeDirect(t, "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if args, _ := os.ReadFile(marker); strings.TrimSpace(string(args)) != "upgrade lostcoords/tap/wariwari" {
		t.Fatalf("brew invoked with %q", args)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("upgrade replaced a Homebrew binary itself: %q", got)
	}
}

func hexSum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestUpgradeExplainsAPermissionProblem(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write anywhere")
	}
	dir := t.TempDir()
	exe := setupUpgrade(t, dir)
	withReleaseServer(t, newFakeRelease(t, "0.2.0", "#!/bin/sh\n"))
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, _, _, err := runUpgradeDirect(t)
	if err == nil || !strings.Contains(err.Error(), "permission denied") || !strings.Contains(err.Error(), dir) {
		t.Fatalf("error = %v, want a permission hint naming %s", err, dir)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("binary changed: %q", got)
	}
}
