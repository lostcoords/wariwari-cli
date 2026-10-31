package commands

import (
	"net/http"
	"strings"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/config"
)

func TestDoctorWarnsWithoutToken(t *testing.T) {
	t.Setenv("WARIWARI_TOKEN", "must-not-leak-into-injected-config")
	cfg := &config.Config{APIURL: "http://example.invalid"}
	stdout, _, requests, err := runRootCommand(t, cfg, okJSON, "doctor", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("expected no live API requests, got %d", len(requests))
	}
	if !strings.Contains(stdout, `"auth_check"`) || !strings.Contains(stdout, `skipping live API check`) {
		t.Fatalf("unexpected doctor output: %s", stdout)
	}
}

func TestDoctorChecksLiveAPI(t *testing.T) {
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token", Group: "grp_1"}
	stdout, _, requests, err := runRootCommand(t, cfg, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/account":
			_, _ = w.Write([]byte(`{"id":1,"emailAddress":"doctor@example.com","name":null}`))
		case "/v1/groups/grp_1":
			_, _ = w.Write([]byte(`{"token":"grp_1"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}, "doctor", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 live API requests, got %d", len(requests))
	}
	if !strings.Contains(stdout, `Authenticated as doctor@example.com`) {
		t.Fatalf("expected authenticated message, got %s", stdout)
	}
	if !strings.Contains(stdout, `Default group is reachable`) {
		t.Fatalf("expected default group message, got %s", stdout)
	}
}
