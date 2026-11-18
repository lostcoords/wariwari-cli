package commands

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/config"
)

func TestBulkEditSendsTheRevisionsItRead(t *testing.T) {
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	_, _, requests, err := runRootCommand(t, cfg, contractHandler,
		"transaction", "bulk-edit", "grp_1", "--data", `{"transactions":[{"token":"txn_1"},{"token":"txn_2","expectedRevision":5}],"date":"2026-05-01"}`, "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(requests) != 2 || requests[0].Path != "/v1/groups/grp_1/transactions/txn_1" {
		t.Fatalf("expected one read of txn_1 before the write, got %+v", requests)
	}
	var body struct {
		Transactions []struct {
			Token            string `json:"token"`
			ExpectedRevision int    `json:"expectedRevision"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal([]byte(requests[1].Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Transactions) != 2 || body.Transactions[0].ExpectedRevision != 7 || body.Transactions[1].ExpectedRevision != 5 {
		t.Fatalf("transactions = %+v, want txn_1 read as 7 and txn_2 kept at 5", body.Transactions)
	}
}

// A retry of a delete that already went through finds no record to read; it
// still sends the delete and lets the server answer.
func TestDeleteOfAMissingRecordLetsTheServerDecide(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/transactions/") && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"status":404,"code":"NOT_FOUND"}`))
			return
		}
		contractHandler(w, r)
	}
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	for _, args := range [][]string{
		{"participant", "delete", "part_gone", "grp_1"},
		{"category", "delete", "cat_gone", "grp_1"},
		{"transaction", "delete", "txn_gone", "grp_1"},
	} {
		_, _, requests, err := runRootCommand(t, cfg, handler, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		last := requests[len(requests)-1]
		if last.Method != http.MethodDelete || last.Header.Get("Expected-Revision") != "0" {
			t.Fatalf("%v: last request %s with revision %q, want DELETE at 0", args, last.Method, last.Header.Get("Expected-Revision"))
		}
	}
}

func TestParticipantMoveNeedsAPosition(t *testing.T) {
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	_, _, requests, err := runRootCommand(t, cfg, contractHandler, "participant", "move", "part_1", "grp_1")
	if err == nil || len(requests) != 0 {
		t.Fatalf("err = %v, %d requests; want a usage error before any request", err, len(requests))
	}
}

func TestInvitationCreatePrintsTheLinkToShare(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"token":"inv_1","membershipRole":"editor","url":"https://wariwari.jp/invitations/inv_1","expiresAt":null,"createdAt":"2026-11-18T00:00:00Z"}`))
	}
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	stdout, _, _, err := runRootCommand(t, cfg, handler, "invitation", "create", "grp_1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"share https://wariwari.jp/invitations/inv_1", "wariwari invitation revoke inv_1 grp_1"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("output missing %q:\n%s", want, stdout)
		}
	}
}
