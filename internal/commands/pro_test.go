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

func TestProRequiredSaysWhatProCostsAndWhereToBuyIt(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"title":"Pro required","status":403,"code":"PRO_REQUIRED",
			"detail":"Recurring expenses need a Pro group.","upgradeUrl":"https://wariwari.jp/pricing"}`))
	}
	args := []string{"recurring", "create", "grp_1", "--data", `{"description":"Rent"}`, "--every", "monthly", "--starts", "2026-12-01"}

	_, stderr, _, err := runRootCommand(t, proTestConfig(), handler, args...)
	if err == nil {
		t.Fatal("expected a 403 error")
	}
	for _, want := range []string{"Recurring expenses need a Pro group.", "Pro feature", "¥980, paid once", "Upgrade at https://wariwari.jp/pricing"} {
		if !strings.Contains(err.Error()+stderr, want) {
			t.Fatalf("output missing %q:\nerr: %v\nstderr: %s", want, err, stderr)
		}
	}

	stdout, _, _, _ := runRootCommand(t, proTestConfig(), handler, append(args, "--json")...)
	for _, want := range []string{`"code": "PRO_REQUIRED"`, `"upgradeUrl": "https://wariwari.jp/pricing"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("JSON error missing %s:\n%s", want, stdout)
		}
	}
}

func TestQuotaExceededExplainsTheFreeKeyLimit(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "432000")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"title":"Quota exceeded","status":429,"code":"QUOTA_EXCEEDED"}`))
	}
	_, stderr, _, err := runRootCommand(t, proTestConfig(), handler, "group", "list")
	if err == nil {
		t.Fatal("expected a 429 error")
	}
	for _, want := range []string{"50 read calls a month", "writes need Pro", "resets in 5 days"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr)
		}
	}
}

// Without --participant, everyone the snapshot shows owing money is reminded;
// one reminded too recently is skipped rather than failing the rest.
func TestRemindWithoutAParticipantRemindsEveryoneWhoOwes(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"token":"grp_1","netByCurrency":{
				"JPY":{"part_a":-4000,"part_b":4000,"part_c":0},
				"USD":{"part_b":-100,"part_d":-50}}}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "part_d") {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"status":429,"code":"RATE_LIMIT_EXCEEDED","detail":"Reminded recently."}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(append([]byte(`{"channel":"email","echo":`), append(body, '}')...))
	}
	stdout, _, requests, err := runRootCommand(t, proTestConfig(), handler, "remind", "grp_1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var posted []string
	for _, r := range requests[1:] {
		posted = append(posted, r.Body)
	}
	want := []string{`{"participantToken":"part_a"}`, `{"participantToken":"part_b"}`, `{"participantToken":"part_d"}`}
	if strings.Join(posted, " ") != strings.Join(want, " ") {
		t.Fatalf("posted %v, want %v", posted, want)
	}
	for _, want := range []string{`"summary": "2 reminded, 1 skipped"`, `"code": "RATE_LIMIT_EXCEEDED"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("output missing %s:\n%s", want, stdout)
		}
	}
}

func TestReminderScheduleChecksItsDayBeforeCalling(t *testing.T) {
	for _, args := range [][]string{
		{"reminder", "schedule", "weekly", "grp_1"},
		{"reminder", "schedule", "weekly", "grp_1", "--weekday", "7"},
		{"reminder", "schedule", "monthly", "grp_1", "--day", "31"},
		{"reminder", "schedule", "daily", "grp_1"},
		{"remind", "grp_1", "--idempotency-key", "1b4e28ba-2fa1-11d2-883f-0016d3cca427"},
	} {
		_, _, requests, err := runRootCommand(t, proTestConfig(), contractHandler, args...)
		if err == nil || len(requests) != 0 {
			t.Fatalf("%v: err = %v, %d requests; want a usage error before any request", args, err, len(requests))
		}
	}
}

func TestReceiptUploadSendsThePhotoAsMultipart(t *testing.T) {
	_, _, requests, err := runRootCommand(t, proTestConfig(), contractHandler,
		"transaction", "receipt", "upload", "txn_1", "testdata/receipt.jpg", "grp_1")
	if err != nil {
		t.Fatal(err)
	}
	mediaType, params, err := mime.ParseMediaType(requests[0].Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("Content-Type = %q", requests[0].Header.Get("Content-Type"))
	}
	part, err := multipart.NewReader(strings.NewReader(requests[0].Body), params["boundary"]).NextPart()
	if err != nil {
		t.Fatal(err)
	}
	content, _ := io.ReadAll(part)
	original, _ := os.ReadFile("testdata/receipt.jpg")
	if part.FormName() != "photo" || part.FileName() != "receipt.jpg" || part.Header.Get("Content-Type") != "image/jpeg" || !bytes.Equal(content, original) {
		t.Fatalf("part %q %q %q with %d bytes", part.FormName(), part.FileName(), part.Header.Get("Content-Type"), len(content))
	}
}

func TestAccountAndGroupSummariesShowPro(t *testing.T) {
	tests := []struct {
		body string
		args []string
		want string
	}{
		{`{"token":"usr_1","pro":true}`, []string{"account", "show"}, `"summary": "Account (Pro)"`},
		{`{"token":"usr_1","pro":false}`, []string{"me"}, `"summary": "Account (free)"`},
		{`{"token":"grp_1","pro":true}`, []string{"group", "show", "grp_1"}, `"summary": "Group snapshot (Pro)"`},
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
