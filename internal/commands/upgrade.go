package commands

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	releasesURL = "https://github.com/lostcoords/wariwari-cli/releases"
	brewUpgrade = "brew upgrade lostcoords/tap/wariwari"
)

var upgradeVersion string
var upgradeCheck bool

// executable is os.Executable, swapped in tests for a throwaway binary.
var executable = os.Executable

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade wariwari to the latest release",
	Long: `Download the latest release (or --version vX.Y.Z), check it against the
release's checksums.txt, and replace this binary in place. A Homebrew install
runs "brew upgrade lostcoords/tap/wariwari" instead. An installed agent skill
is refreshed from the new binary.

--check only reports whether an update exists.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := executable()
		if err != nil {
			return fmt.Errorf("cannot find the running wariwari: %w", err)
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}

		release := upgradeReleaseURL(upgradeVersion)
		target, err := fetchReleaseTarget(cmd.Context(), release)
		if err != nil {
			return err
		}
		current := strings.TrimPrefix(version, "v")
		result := map[string]any{"current": current, "latest": target.version, "updateAvailable": target.version != current}

		if upgradeCheck {
			summary := "wariwari is up to date"
			if target.version != current {
				summary = fmt.Sprintf("wariwari %s is available (current %s); run wariwari upgrade", target.version, current)
			}
			return printOK(result, summary)
		}
		if target.version == current {
			return printOK(result, "wariwari is up to date")
		}

		if isHomebrew(exe) {
			fmt.Fprintf(errOut, "Installed with Homebrew; running %s\n", brewUpgrade)
			brew := exec.CommandContext(cmd.Context(), "brew", "upgrade", "lostcoords/tap/wariwari")
			brew.Stdout, brew.Stderr = errOut, errOut
			if err := brew.Run(); err != nil {
				return fmt.Errorf("%s failed: %w", brewUpgrade, err)
			}
			result["path"] = exe
			return printOK(result, "Upgraded with Homebrew")
		}

		if runtime.GOOS == "windows" {
			return fmt.Errorf("upgrade cannot replace a running binary on Windows; download %s from %s", target.asset, release)
		}
		binary, err := downloadVerifiedBinary(cmd.Context(), release, target)
		if err != nil {
			return err
		}
		if err := replaceExecutable(exe, binary); err != nil {
			return err
		}
		result["path"] = exe
		result["skillRefreshed"] = refreshInstalledSkill(exe)
		return printOK(result, fmt.Sprintf("Upgraded wariwari %s -> %s", current, target.version))
	},
}

// upgradeReleaseURL is the directory holding checksums.txt and the archives.
// WARIWARI_RELEASE_URL overrides it to test against a local build.
func upgradeReleaseURL(pinned string) string {
	if override := os.Getenv("WARIWARI_RELEASE_URL"); override != "" {
		return strings.TrimRight(override, "/")
	}
	if pinned != "" {
		if pinned == "nightly" {
			return releasesURL + "/download/nightly"
		}
		return releasesURL + "/download/v" + strings.TrimPrefix(pinned, "v")
	}
	return releasesURL + "/latest/download"
}

type releaseTarget struct {
	asset   string
	sha256  string
	version string
}

// fetchReleaseTarget reads checksums.txt and picks this platform's archive.
// GoReleaser names archives wariwari_<version>_<os>_<arch>.tar.gz, so the
// checksum file also says which version the release is.
func fetchReleaseTarget(ctx context.Context, release string) (releaseTarget, error) {
	checksums, err := fetchRelease(ctx, release+"/checksums.txt")
	if err != nil {
		return releaseTarget{}, err
	}
	suffix := fmt.Sprintf("_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	scanner := bufio.NewScanner(bytes.NewReader(checksums))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || !strings.HasPrefix(fields[1], "wariwari_") || !strings.HasSuffix(fields[1], suffix) {
			continue
		}
		return releaseTarget{
			asset:   fields[1],
			sha256:  fields[0],
			version: strings.TrimSuffix(strings.TrimPrefix(fields[1], "wariwari_"), suffix),
		}, nil
	}
	return releaseTarget{}, fmt.Errorf("no %s/%s archive in %s/checksums.txt", runtime.GOOS, runtime.GOARCH, release)
}

func fetchRelease(ctx context.Context, url string) ([]byte, error) {
	if path, ok := strings.CutPrefix(url, "file://"); ok {
		return os.ReadFile(path)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// downloadVerifiedBinary fetches the archive, checks its SHA-256, and returns
// the wariwari binary inside it.
func downloadVerifiedBinary(ctx context.Context, release string, target releaseTarget) ([]byte, error) {
	archive, err := fetchRelease(ctx, release+"/"+target.asset)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(archive)
	if hex.EncodeToString(sum[:]) != target.sha256 {
		return nil, fmt.Errorf("checksum mismatch for %s; nothing installed", target.asset)
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", target.asset, err)
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s has no wariwari binary", target.asset)
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", target.asset, err)
		}
		if header.Name == "wariwari" && header.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
}

// replaceExecutable writes the new binary beside the old one and renames it
// over it, so a failed upgrade never leaves a half-written wariwari.
func replaceExecutable(exe string, binary []byte) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".wariwari-upgrade-*")
	if err != nil {
		return writeError(dir, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		return writeError(dir, err)
	}
	return nil
}

func writeError(dir string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("cannot write to %s: permission denied. Rerun with permission to write there, or reinstall with the install script", dir)
	}
	return err
}

func isHomebrew(exe string) bool {
	return strings.Contains(filepath.ToSlash(exe), "/Cellar/")
}

// refreshInstalledSkill reinstalls the agent skill from the new binary when
// `wariwari skill install` has put it somewhere before.
func refreshInstalledSkill(exe string) bool {
	installed := false
	for _, dir := range skillDirs() {
		installed = installed || fileExists(filepath.Join(dir, skillFilename))
	}
	if !installed {
		return false
	}
	refresh := exec.Command(exe, "skill", "install", "--quiet")
	refresh.Stdout = io.Discard
	refresh.Stderr = errOut
	if err := refresh.Run(); err != nil {
		fmt.Fprintf(errOut, "Could not refresh the agent skill (%v); run wariwari skill install\n", err)
		return false
	}
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().StringVar(&upgradeVersion, "version", "", "Install this release, e.g. v0.2.0 or nightly, instead of the latest")
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, "Only report whether an update exists")
}
