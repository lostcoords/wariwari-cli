package commands

import (
	"net/http"
	"strings"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/config"
)

func okJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func TestCommandContractsUseCurrentAPIRoutes(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		group      string
		wantMethod string
		wantPath   string
	}{
		{
			name:       "group list pages with take and cursor",
			args:       []string{"group", "list", "--take", "25", "--cursor", "abc"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups?cursor=abc&take=25",
		},
		{
			name:       "group list filters archived",
			args:       []string{"group", "list", "--archived"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups?archived=true",
		},
		{
			name:       "group show reads the snapshot",
			args:       []string{"group", "show", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1",
		},
		{
			name:       "group snapshot is an alias for show",
			args:       []string{"group", "snapshot", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1",
		},
		{
			name:       "group update uses patch",
			args:       []string{"group", "update", "grp_1", "--data", `{"lockVersion":1}`},
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/groups/grp_1",
		},
		{
			name:       "balance derives from the group snapshot",
			args:       []string{"balance", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1",
		},
		{
			name:       "participant list derives from the group snapshot",
			args:       []string{"participant", "list", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1",
		},
		{
			name:       "category list derives from the group snapshot",
			args:       []string{"category", "list", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1",
		},
		{
			name:       "participant create uses collection post",
			args:       []string{"participant", "create", "grp_1", "--data", `{"name":"Ada"}`},
			wantMethod: http.MethodPost,
			wantPath:   "/v1/groups/grp_1/participants",
		},
		{
			name:       "participant update uses patch",
			args:       []string{"participant", "update", "part_1", "grp_1", "--data", `{"lockVersion":1}`},
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/groups/grp_1/participants/part_1",
		},
		{
			name:       "category create uses collection post",
			args:       []string{"category", "create", "grp_1", "--data", `{"name":"Food","emoji":"🍜"}`},
			wantMethod: http.MethodPost,
			wantPath:   "/v1/groups/grp_1/categories",
		},
		{
			name:       "category update uses patch",
			args:       []string{"category", "update", "cat_1", "grp_1", "--data", `{"lockVersion":1}`},
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/groups/grp_1/categories/cat_1",
		},
		{
			name:       "transaction list reads the group collection",
			args:       []string{"transaction", "list", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1/transactions",
		},
		{
			name:       "transaction list passes every filter",
			args:       []string{"transaction", "list", "grp_1", "--from", "2026-01-01", "--to", "2026-01-31", "--category", "cat_1", "--participant", "part_1", "--take", "10", "--cursor", "abc"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1/transactions?category=cat_1&cursor=abc&from=2026-01-01&participant=part_1&take=10&to=2026-01-31",
		},
		{
			name:       "transaction show reads one transaction",
			args:       []string{"transaction", "show", "txn_1", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1/transactions/txn_1",
		},
		{
			name:       "transaction create uses collection post",
			args:       []string{"transaction", "create", "grp_1", "--data", `{"description":"Dinner","amountCents":1000,"currency":"JPY","splitType":"equal","date":"2026-05-23","paidByToken":"part_1","splits":[{"participantToken":"part_1","amountCents":1000}]}`},
			wantMethod: http.MethodPost,
			wantPath:   "/v1/groups/grp_1/transactions",
		},
		{
			name:       "transaction update uses patch",
			args:       []string{"transaction", "update", "txn_1", "grp_1", "--data", `{"lockVersion":1}`},
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/groups/grp_1/transactions/txn_1",
		},
		{
			name:       "transaction delete soft-deletes",
			args:       []string{"transaction", "delete", "txn_1", "grp_1"},
			wantMethod: http.MethodDelete,
			wantPath:   "/v1/groups/grp_1/transactions/txn_1",
		},
		{
			name:       "transaction activity reads the nested feed",
			args:       []string{"transaction", "activity", "txn_1", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1/transactions/txn_1/activity",
		},
		{
			name:       "group activity reads the group feed",
			args:       []string{"group", "activity", "grp_1"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/groups/grp_1/activity",
		},
		{
			name:       "me reads the identity endpoint",
			args:       []string{"me"},
			wantMethod: http.MethodGet,
			wantPath:   "/v1/me",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token", Group: tt.group}
			stdout, _, requests, err := runRootCommand(t, cfg, okJSON, append(tt.args, "--json")...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(stdout, `"ok": true`) {
				t.Fatalf("expected JSON success output, got %s", stdout)
			}
			if len(requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(requests))
			}
			if requests[0].Method != tt.wantMethod {
				t.Fatalf("method = %s, want %s", requests[0].Method, tt.wantMethod)
			}
			if requests[0].Path != tt.wantPath {
				t.Fatalf("path = %s, want %s", requests[0].Path, tt.wantPath)
			}
			if got := requests[0].Header.Get("Authorization"); got != "Bearer test-token" {
				t.Fatalf("authorization header = %q", got)
			}
		})
	}
}

func TestGroupCreateSendsIdempotencyKey(t *testing.T) {
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	_, _, requests, err := runRootCommand(t, cfg, okJSON,
		"group", "create", "--data", `{"name":"Trip","icon":"🧳","participants":[{"name":"Ada"}]}`,
		"--idempotency-key", "retry-1234", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Method != http.MethodPost || requests[0].Path != "/v1/groups" {
		t.Fatalf("got %s %s, want POST /v1/groups", requests[0].Method, requests[0].Path)
	}
	if got := requests[0].Header.Get("Idempotency-Key"); got != "retry-1234" {
		t.Fatalf("Idempotency-Key = %q", got)
	}
}

// API keys are managed on the web account page; there is no /v1/api_keys route
// for the CLI to call.
func TestTokenCommandNeverCallsTheAPI(t *testing.T) {
	for _, args := range [][]string{
		{"token"},
		{"token", "list"},
		{"token", "create", "--name", "Automation"},
		{"token", "delete", "key_1"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
			_, _, requests, err := runRootCommand(t, cfg, okJSON, args...)
			if err == nil {
				t.Fatal("expected an error explaining that keys are web-managed")
			}
			if !strings.Contains(err.Error(), "web account page") {
				t.Fatalf("error = %v, want the web account page hint", err)
			}
			if len(requests) != 0 {
				t.Fatalf("expected no API requests, got %d", len(requests))
			}
		})
	}
}

func TestListsDeriveRowsFromTheSnapshot(t *testing.T) {
	snapshot := `{"token":"grp_1","name":"Trip","currency":"JPY",
		"participants":[{"token":"part_1","name":"Ada","lockVersion":0}],
		"groupCategories":[{"token":"cat_1","name":"Food","emoji":"🍜","lockVersion":0}],
		"netByCurrency":{"JPY":{"part_1":-4000}},
		"suggestedPayments":[{"fromParticipantToken":"part_1","toParticipantToken":"part_2","amountCents":4000,"currency":"JPY"}],
		"spendingByCurrency":{"JPY":12000},"totalBaseCents":12000}`
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(snapshot))
	}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"participants", []string{"participant", "list", "grp_1"}, []string{`"participants"`, `"part_1"`}},
		{"categories", []string{"category", "list", "grp_1"}, []string{`"groupCategories"`, `"cat_1"`}},
		{"balances", []string{"balance", "grp_1"}, []string{`"netByCurrency"`, `"suggestedPayments"`, `"spendingByCurrency"`, `"totalBaseCents"`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
			stdout, _, _, err := runRootCommand(t, cfg, handler, append(tt.args, "--quiet")...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(stdout, want) {
					t.Fatalf("output missing %s:\n%s", want, stdout)
				}
			}
		})
	}
}

func TestCountAndIDsReadCamelCaseCollections(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transactions":[{"token":"txn_1"},{"token":"txn_2"}],"pagination":{"hasMore":false,"cursor":null}}`))
	}

	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	stdout, _, _, err := runRootCommand(t, cfg, handler, "transaction", "list", "grp_1", "--count")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.TrimSpace(stdout) != "2" {
		t.Fatalf("--count = %q, want 2", strings.TrimSpace(stdout))
	}

	stdout, _, _, err = runRootCommand(t, cfg, handler, "transaction", "list", "grp_1", "--ids-only")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := strings.Fields(stdout); len(got) != 2 || got[0] != "txn_1" || got[1] != "txn_2" {
		t.Fatalf("--ids-only = %v, want [txn_1 txn_2]", got)
	}
}

func TestPaginationBreadcrumbFollowsTheCursor(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[{"token":"grp_1"}],"pagination":{"hasMore":true,"cursor":"eyJraW5kIjoia2V5c2V0In0"}}`))
	}

	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	stdout, _, _, err := runRootCommand(t, cfg, handler, "group", "list", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout, "wariwari group list --cursor eyJraW5kIjoia2V5c2V0In0") {
		t.Fatalf("expected a next-page breadcrumb, got %s", stdout)
	}
}

func TestProblemDetailsSurfaceCodeAndDetail(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"type":"https://wariwari.jp/docs/errors/conflict","title":"Conflict",
			"status":409,"detail":"This transaction changed since you read it.","code":"CONFLICT",
			"instance":"req_1","current":{"token":"txn_1","lockVersion":4}}`))
	}
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	args := []string{"transaction", "update", "txn_1", "grp_1", "--data", `{"lockVersion":1}`}

	_, stderr, _, err := runRootCommand(t, cfg, handler, args...)
	if err == nil {
		t.Fatal("expected a 409 error")
	}
	if !strings.Contains(err.Error(), "CONFLICT") || !strings.Contains(err.Error(), "This transaction changed since you read it.") {
		t.Fatalf("error = %q, want the code and the detail", err.Error())
	}
	if !strings.Contains(stderr, "lockVersion from `current`") {
		t.Fatalf("stderr missing the conflict hint:\n%s", stderr)
	}

	stdout, _, _, err := runRootCommand(t, cfg, handler, append(args, "--json")...)
	if err == nil {
		t.Fatal("expected a 409 error")
	}
	for _, want := range []string{`"code": "CONFLICT"`, `"current"`, `"instance": "req_1"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("JSON error output missing %s:\n%s", want, stdout)
		}
	}
}

func TestValidationErrorsListOffendingFields(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"title":"Validation failed","status":400,
			"detail":"The transaction could not be saved.","code":"VALIDATION_FAILED",
			"errors":{"amountCents":"must be greater than 0"}}`))
	}

	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	_, stderr, _, err := runRootCommand(t, cfg, handler, "transaction", "create", "grp_1", "--data", `{"amountCents":0}`)
	if err == nil {
		t.Fatal("expected a 400 error")
	}
	if !strings.Contains(err.Error(), "VALIDATION_FAILED") {
		t.Fatalf("error = %q, want the code", err.Error())
	}
	if !strings.Contains(stderr, "amountCents: must be greater than 0") {
		t.Fatalf("stderr missing the field error:\n%s", stderr)
	}
}
