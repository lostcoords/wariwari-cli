package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/lostcoords/wariwari-cli/internal/client"
)

// cliAuthorization is the RFC 8628 device authorization the API hands out
// before a human approves the login in the browser.
type cliAuthorization struct {
	DeviceCode              string `json:"deviceCode"`
	UserCode                string `json:"userCode"`
	VerificationURI         string `json:"verificationUri"`
	VerificationURIComplete string `json:"verificationUriComplete"`
	ExpiresIn               int    `json:"expiresIn"`
	Interval                int    `json:"interval"`
}

type browserLoginResult struct {
	Token string
	User  map[string]any
}

// sleepFor waits between token polls; tests swap it to skip the wait and
// record the intervals.
var sleepFor = func(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// openURL starts the platform's URL opener without waiting for it.
var openURL = func(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// browserLogin runs the device authorization grant against apiURL: it asks for
// a code, sends the human to the browser, and polls until they approve.
// Both requests are unauthenticated, whatever token is configured.
func browserLogin(ctx context.Context, apiURL string, openBrowser bool) (browserLoginResult, error) {
	anonymous := client.New(apiURL, "", version)

	hostname, _ := os.Hostname()
	if len(hostname) > 80 {
		hostname = hostname[:80]
	}
	data, err := anonymous.Post(ctx, "/cli_authorizations", map[string]any{"clientName": hostname}, "")
	if err != nil {
		return browserLoginResult{}, err
	}
	var authorization cliAuthorization
	if err := decodeInto(data, &authorization); err != nil || authorization.DeviceCode == "" {
		return browserLoginResult{}, fmt.Errorf("unexpected login response from %s", apiURL)
	}

	fmt.Fprintf(errOut, "! First copy your one-time code: %s\n", authorization.UserCode)
	url := authorization.VerificationURIComplete
	if url == "" {
		url = authorization.VerificationURI
	}
	if openBrowser && openURL(url) == nil {
		fmt.Fprintf(errOut, "Opening %s in your browser…\n", url)
	} else {
		fmt.Fprintf(errOut, "Open this URL in a browser: %s\n", url)
	}
	fmt.Fprintln(errOut, "Waiting for approval…")

	interval := time.Duration(authorization.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	remaining := time.Duration(authorization.ExpiresIn) * time.Second
	for {
		if remaining <= 0 {
			return browserLoginResult{}, errLoginExpired
		}
		if err := sleepFor(ctx, interval); err != nil {
			return browserLoginResult{}, errLoginCancelled
		}
		remaining -= interval

		data, err := anonymous.Post(ctx, "/cli_authorizations/token", map[string]any{"deviceCode": authorization.DeviceCode}, "")
		if err == nil {
			body, _ := data.(map[string]any)
			token, _ := body["token"].(string)
			if token == "" {
				return browserLoginResult{}, fmt.Errorf("unexpected token response from %s", apiURL)
			}
			user, _ := body["user"].(map[string]any)
			return browserLoginResult{Token: token, User: user}, nil
		}
		if ctx.Err() != nil {
			return browserLoginResult{}, errLoginCancelled
		}

		var apiErr *client.Error
		if !errors.As(err, &apiErr) {
			return browserLoginResult{}, err
		}
		switch apiErr.Code() {
		case "AUTHORIZATION_PENDING":
		case "SLOW_DOWN":
			interval += 5 * time.Second
		case "ACCESS_DENIED":
			return browserLoginResult{}, errors.New("Login was denied in the browser.")
		case "EXPIRED_TOKEN":
			return browserLoginResult{}, errLoginExpired
		default:
			return browserLoginResult{}, err
		}
	}
}

var (
	errLoginExpired   = errors.New("The login request expired; run wariwari auth login again.")
	errLoginCancelled = errors.New("Login cancelled.")
)

// canOpenBrowser says whether opening a browser could reach the human at this
// terminal.
func canOpenBrowser(noBrowser bool) bool {
	if noBrowser {
		return false
	}
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return false
	}
	if agent && !stdinIsTerminal() {
		return false
	}
	return true
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// userLabel picks the email, else the name, from an account payload.
func userLabel(user map[string]any) string {
	for _, key := range []string{"emailAddress", "name"} {
		if value, ok := user[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}
