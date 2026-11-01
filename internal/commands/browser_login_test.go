package commands

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lostcoords/wariwari-cli/internal/config"
)

const authorizationJSON = `{"deviceCode":"dev_123","userCode":"BCDF-GHJK",
	"verificationUri":"https://wariwari.jp/cli","verificationUriComplete":"https://wariwari.jp/cli?code=BCDF-GHJK",
	"expiresIn":600,"interval":5}`

// deviceFlowServer fakes both endpoints. Each token poll takes the next reply: each poll takes the next
// in order: a problem code, or "" for success.
func deviceFlowServer(t *testing.T, replies ...string) http.HandlerFunc {
	t.Helper()
	polls := 0
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/v1/cli_authorizations":
			var req map[string]string
			_ = json.Unmarshal(body, &req)
			if req["clientName"] == "" {
				t.Errorf("clientName missing from %s", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(authorizationJSON))
		case "/v1/cli_authorizations/token":
			if !strings.Contains(string(body), `"deviceCode":"dev_123"`) {
				t.Errorf("token poll body = %s", body)
			}
			if polls >= len(replies) {
				t.Fatalf("unexpected poll %d", polls+1)
			}
			reply := replies[polls]
			polls++
			if reply == "" {
				_, _ = w.Write([]byte(`{"token":"wari_fresh","user":{"id":2,"emailAddress":"ada@example.com","name":"Ada"}}`))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"title":"Pending","status":400,"code":"` + reply + `","detail":"` + reply + `"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}
}

// stubDeviceFlow skips the poll waits, records them and any opened URL, and
// keeps credentials in a throwaway home.
func stubDeviceFlow(t *testing.T) (*[]time.Duration, *[]string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("WARIWARI_NO_KEYRING", "1")
	t.Setenv("DISPLAY", ":0")

	var waits []time.Duration
	var opened []string
	previousSleep, previousOpen := sleepFor, openURL
	sleepFor = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	openURL = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	t.Cleanup(func() { sleepFor, openURL = previousSleep, previousOpen })
	return &waits, &opened
}

func TestBrowserLoginPollsSlowsDownAndSavesTheToken(t *testing.T) {
	waits, opened := stubDeviceFlow(t)
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "wari_old_configured"}

	stdout, stderr, requests, err := runRootCommand(t, cfg,
		deviceFlowServer(t, "AUTHORIZATION_PENDING", "SLOW_DOWN", "AUTHORIZATION_PENDING", ""),
		"auth", "login", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr: %s", err, stderr)
	}

	for _, r := range requests {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("%s sent Authorization %q; the device flow must be unauthenticated", r.Path, got)
		}
	}
	want := []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second, 10 * time.Second}
	if len(*waits) != len(want) {
		t.Fatalf("waits = %v, want %v", *waits, want)
	}
	for i := range want {
		if (*waits)[i] != want[i] {
			t.Fatalf("waits = %v, want %v (SLOW_DOWN adds 5s for good)", *waits, want)
		}
	}
	if len(*opened) != 1 || (*opened)[0] != "https://wariwari.jp/cli?code=BCDF-GHJK" {
		t.Fatalf("opened = %v", *opened)
	}
	for _, line := range []string{
		"First copy your one-time code: BCDF-GHJK",
		"Opening https://wariwari.jp/cli?code=BCDF-GHJK in your browser",
		"Waiting for approval",
		"Logged in as ada@example.com.",
	} {
		if !strings.Contains(stderr, line) {
			t.Fatalf("stderr missing %q:\n%s", line, stderr)
		}
	}
	if strings.Contains(stdout, "BCDF-GHJK") {
		t.Fatalf("stdout should hold only the JSON envelope:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"user": "ada@example.com"`) || !strings.Contains(stdout, `"profile": "default"`) {
		t.Fatalf("unexpected envelope:\n%s", stdout)
	}
	if token, _ := config.LoadToken("default"); token != "wari_fresh" {
		t.Fatalf("saved token = %q, want wari_fresh", token)
	}
}

func TestBrowserLoginStopsWhenDenied(t *testing.T) {
	stubDeviceFlow(t)
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, _, _, err := runRootCommand(t, cfg, deviceFlowServer(t, "AUTHORIZATION_PENDING", "ACCESS_DENIED"), "auth", "login")
	if err == nil || err.Error() != "Login was denied in the browser." {
		t.Fatalf("error = %v", err)
	}
	if token, _ := config.LoadToken("default"); token != "" {
		t.Fatalf("a denied login saved %q", token)
	}
}

func TestBrowserLoginStopsWhenExpired(t *testing.T) {
	stubDeviceFlow(t)
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, _, _, err := runRootCommand(t, cfg, deviceFlowServer(t, "EXPIRED_TOKEN"), "auth", "login")
	if err == nil || err.Error() != "The login request expired; run wariwari auth login again." {
		t.Fatalf("error = %v", err)
	}
}

func TestBrowserLoginGivesUpAtExpiresIn(t *testing.T) {
	waits, _ := stubDeviceFlow(t)
	pending := make([]string, 120)
	for i := range pending {
		pending[i] = "AUTHORIZATION_PENDING"
	}
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, _, _, err := runRootCommand(t, cfg, deviceFlowServer(t, pending...), "auth", "login")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %v", err)
	}
	if len(*waits) != 120 {
		t.Fatalf("polled %d times, want 600s / 5s = 120", len(*waits))
	}
}

func TestBrowserLoginTreatsOtherErrorsAsFatal(t *testing.T) {
	stubDeviceFlow(t)
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, _, _, err := runRootCommand(t, cfg, deviceFlowServer(t, "INVALID_REQUEST"), "auth", "login")
	if err == nil || !strings.Contains(err.Error(), "INVALID_REQUEST") {
		t.Fatalf("error = %v", err)
	}
}

func TestBrowserLoginPrintsTheURLWithoutABrowser(t *testing.T) {
	_, opened := stubDeviceFlow(t)
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, stderr, _, err := runRootCommand(t, cfg, deviceFlowServer(t, ""), "auth", "login", "--no-browser")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(*opened) != 0 {
		t.Fatalf("--no-browser opened %v", *opened)
	}
	if !strings.Contains(stderr, "Open this URL in a browser: https://wariwari.jp/cli?code=BCDF-GHJK") {
		t.Fatalf("stderr:\n%s", stderr)
	}
}

func TestBrowserLoginSkipsTheBrowserWithoutADisplay(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the display check is Linux-only")
	}
	_, opened := stubDeviceFlow(t)
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, stderr, _, err := runRootCommand(t, cfg, deviceFlowServer(t, ""), "auth", "login")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(*opened) != 0 || !strings.Contains(stderr, "Open this URL in a browser") {
		t.Fatalf("opened %v with no display; stderr:\n%s", *opened, stderr)
	}
}

func TestAuthLoginWithATokenSkipsTheBrowser(t *testing.T) {
	_, opened := stubDeviceFlow(t)
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, _, requests, err := runRootCommand(t, cfg, okJSON, "auth", "login", "wari_pasted", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(requests) != 0 || len(*opened) != 0 {
		t.Fatalf("token login made %d requests and opened %v", len(requests), *opened)
	}
	if token, _ := config.LoadToken("default"); token != "wari_pasted" {
		t.Fatalf("saved token = %q", token)
	}
}

func TestBrowserLoginStopsCleanlyWhenCancelled(t *testing.T) {
	stubDeviceFlow(t)
	sleepFor = func(context.Context, time.Duration) error { return context.Canceled }
	cfg := &config.Config{APIURL: "http://example.invalid"}
	_, _, requests, err := runRootCommand(t, cfg, deviceFlowServer(t), "auth", "login")
	if err == nil || err.Error() != "Login cancelled." {
		t.Fatalf("error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected only the authorization request, got %d", len(requests))
	}
}
