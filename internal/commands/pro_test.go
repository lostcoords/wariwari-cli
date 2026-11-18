package commands

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/config"
)

func proTestConfig() *config.Config {
	return &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
}

// The server's detail carries the price and limit, so a change to either
// never needs a CLI release; the CLI adds the upgrade link.
func TestProErrorsShowTheServerDetailAndUpgradeLink(t *testing.T) {
	tests := []struct {
		name, status, body string
		retryAfter         string
		args               []string
		want               []string
	}{
		{"pro required", "403", `{"status":403,"code":"PRO_REQUIRED","detail":"Recurring expenses need Pro: ¥980, once.","upgradeUrl":"https://wariwari.jp/pricing"}`, "",
			[]string{"recurring", "create", "grp_1", "--data", `{"description":"Rent"}`},
			[]string{"Recurring expenses need Pro: ¥980, once.", "Upgrade at https://wariwari.jp/pricing"}},
		{"quota exceeded", "429", `{"status":429,"code":"QUOTA_EXCEEDED","detail":"Free keys get 50 calls a month.","upgradeUrl":"https://wariwari.jp/pricing"}`, "432000",
			[]string{"group", "list"},
			[]string{"Free keys get 50 calls a month.", "Upgrade at https://wariwari.jp/pricing", "resets in 5 days"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.retryAfter != "" {
					w.Header().Set("Retry-After", tt.retryAfter)
				}
				status := http.StatusForbidden
				if tt.status == "429" {
					status = http.StatusTooManyRequests
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(tt.body))
			}
			_, stderr, _, err := runRootCommand(t, proTestConfig(), handler, tt.args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error()+"\n"+stderr, want) {
					t.Fatalf("output missing %q:\nerr: %v\nstderr: %s", want, err, stderr)
				}
			}
			stdout, _, _, _ := runRootCommand(t, proTestConfig(), handler, append(tt.args, "--json")...)
			if !strings.Contains(stdout, `"upgradeUrl": "https://wariwari.jp/pricing"`) {
				t.Fatalf("JSON error missing upgradeUrl:\n%s", stdout)
			}
		})
	}
}

// Both "none here" and "no such group or expense" are 404; only the first is
// reported as an empty result.
func TestA404IsAbsenceOnlyWhenTheParentExists(t *testing.T) {
	notFound := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"code":"NOT_FOUND"}`))
	}
	parentExists := func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/receipt_photo") {
			notFound(w)
			return
		}
		contractHandler(w, r)
	}
	nothingExists := func(w http.ResponseWriter, _ *http.Request) { notFound(w) }

	tests := []struct {
		args []string
		want string
	}{
		{[]string{"transaction", "receipt", "show", "txn_1", "grp_1"}, `"receiptPhoto": null`},
	}
	for _, tt := range tests {
		stdout, _, _, err := runRootCommand(t, proTestConfig(), parentExists, append(tt.args, "--json")...)
		if err != nil || !strings.Contains(stdout, tt.want) {
			t.Fatalf("%v: err = %v, output missing %s:\n%s", tt.args, err, tt.want, stdout)
		}
		if _, _, _, err := runRootCommand(t, proTestConfig(), nothingExists, tt.args...); err == nil || !strings.Contains(err.Error(), "NOT_FOUND") {
			t.Fatalf("%v: err = %v, want NOT_FOUND for a missing parent", tt.args, err)
		}
	}
}

func TestUploadsSendTheFileAsMultipart(t *testing.T) {
	tests := []struct {
		args  []string
		field string
	}{
		{[]string{"transaction", "receipt", "upload", "txn_1", "testdata/receipt.jpg", "grp_1"}, "photo"},
		{[]string{"file", "upload", "testdata/receipt.jpg", "grp_1"}, "file"},
	}
	original, _ := os.ReadFile("testdata/receipt.jpg")
	for _, tt := range tests {
		_, _, requests, err := runRootCommand(t, proTestConfig(), contractHandler, tt.args...)
		if err != nil {
			t.Fatal(err)
		}
		mediaType, params, err := mime.ParseMediaType(requests[0].Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Fatalf("%v: Content-Type = %q", tt.args, requests[0].Header.Get("Content-Type"))
		}
		part, err := multipart.NewReader(strings.NewReader(requests[0].Body), params["boundary"]).NextPart()
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(part)
		if part.FormName() != tt.field || part.FileName() != "receipt.jpg" || part.Header.Get("Content-Type") != "image/jpeg" || !bytes.Equal(content, original) {
			t.Fatalf("%v: part %q %q %q with %d bytes", tt.args, part.FormName(), part.FileName(), part.Header.Get("Content-Type"), len(content))
		}
	}
}

func TestAccountSummaryShowsPro(t *testing.T) {
	tests := []struct {
		body string
		args []string
		want string
	}{
		{`{"analyticsId":"acct_1","pro":true}`, []string{"account", "show"}, `"summary": "Account (Pro)"`},
		{`{"analyticsId":"acct_1","pro":false}`, []string{"me"}, `"summary": "Account (free)"`},
	}
	for _, tt := range tests {
		handler := func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tt.body))
		}
		stdout, _, _, err := runRootCommand(t, proTestConfig(), handler, append(tt.args, "--json")...)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout, tt.want) {
			t.Fatalf("%v: output missing %s:\n%s", tt.args, tt.want, stdout)
		}
	}
}
