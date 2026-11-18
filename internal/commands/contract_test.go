package commands

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/config"
)

func okJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// contract pins one command to the request it ends with. operation is the
// spec's operationId; spec_test.go checks the method, path template and
// required headers against the vendored OpenAPI document.
type contract struct {
	name      string
	args      []string
	operation string
	method    string
	path      string
	headers   map[string]string
	// reads is how many GETs the command makes before its final request,
	// e.g. to learn a lockVersion.
	reads int
}

// contractRecord answers every GET: a group snapshot carrying one participant
// and one category, which also stands in for a transaction record.
const contractRecord = `{"ok":true,"token":"grp_1","lockVersion":7,
	"participants":[{"token":"part_1","lockVersion":3}],
	"categories":[{"token":"cat_1","lockVersion":2}]}`

func contractHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		_, _ = w.Write([]byte(contractRecord))
		return
	}
	_, _ = w.Write([]byte(`{"ok":true}`))
}

var contracts = []contract{
	{name: "account show", args: []string{"account", "show"}, operation: "showAccount", method: "GET", path: "/v1/account"},
	{name: "me is account show", args: []string{"me"}, operation: "showAccount", method: "GET", path: "/v1/account"},
	{name: "account update", args: []string{"account", "update", "--data", `{"locale":"ja"}`}, operation: "updateAccount", method: "PATCH", path: "/v1/account"},
	{name: "search joins the query", args: []string{"search", "ramen", "night"}, operation: "listSearchResults", method: "GET", path: "/v1/search?q=ramen+night"},
	{name: "feedback", args: []string{"feedback", "Love", "it", "--email", "a@example.com"}, operation: "createFeedback", method: "POST", path: "/v1/feedback"},

	{name: "group list pages", args: []string{"group", "list", "--take", "25", "--cursor", "abc"}, operation: "listGroups", method: "GET", path: "/v1/groups?cursor=abc&take=25"},
	{name: "group list archived", args: []string{"group", "list", "--archived"}, operation: "listGroups", method: "GET", path: "/v1/groups?archived=true"},
	{name: "group show", args: []string{"group", "show", "grp_1"}, operation: "showGroup", method: "GET", path: "/v1/groups/grp_1"},
	{name: "group snapshot is show", args: []string{"group", "snapshot", "grp_1"}, operation: "showGroup", method: "GET", path: "/v1/groups/grp_1"},
	{name: "balance reads the snapshot", args: []string{"balance", "grp_1"}, operation: "showGroup", method: "GET", path: "/v1/groups/grp_1"},
	{name: "participant list reads the snapshot", args: []string{"participant", "list", "grp_1"}, operation: "showGroup", method: "GET", path: "/v1/groups/grp_1"},
	{name: "category list reads the snapshot", args: []string{"category", "list", "grp_1"}, operation: "showGroup", method: "GET", path: "/v1/groups/grp_1"},
	{name: "group create", args: []string{"group", "create", "--data", `{"name":"Trip","icon":"🧳","participants":[{"name":"Ada"}]}`}, operation: "createGroup", method: "POST", path: "/v1/groups"},
	{name: "group update reads the lockVersion", args: []string{"group", "update", "grp_1", "--data", `{"visibility":"private_registered"}`}, operation: "updateGroup", method: "PATCH", path: "/v1/groups/grp_1", headers: map[string]string{"Expected-Revision": "7"}, reads: 1},
	{name: "group update with --lock-version", args: []string{"group", "update", "grp_1", "--data", `{"name":"Trip"}`, "--lock-version", "2"}, operation: "updateGroup", method: "PATCH", path: "/v1/groups/grp_1", headers: map[string]string{"Expected-Revision": "2"}},
	{name: "group delete reads the lockVersion", args: []string{"group", "delete", "grp_1"}, operation: "deleteGroup", method: "DELETE", path: "/v1/groups/grp_1", headers: map[string]string{"Expected-Revision": "7"}, reads: 1},
	{name: "group delete with --lock-version", args: []string{"group", "delete", "grp_1", "--lock-version", "0"}, operation: "deleteGroup", method: "DELETE", path: "/v1/groups/grp_1", headers: map[string]string{"Expected-Revision": "0"}},
	{name: "group activity", args: []string{"group", "activity", "grp_1"}, operation: "listGroupActivity", method: "GET", path: "/v1/groups/grp_1/activity"},
	{name: "group archive", args: []string{"group", "archive", "grp_1"}, operation: "archiveGroup", method: "POST", path: "/v1/groups/grp_1/archival"},
	{name: "group unarchive", args: []string{"group", "unarchive", "grp_1"}, operation: "unarchiveGroup", method: "DELETE", path: "/v1/groups/grp_1/archival"},
	{name: "group leave", args: []string{"group", "leave", "grp_1"}, operation: "leaveGroup", method: "DELETE", path: "/v1/groups/grp_1/membership"},
	{name: "group duplicate", args: []string{"group", "duplicate", "grp_1", "--include-members"}, operation: "duplicateGroup", method: "POST", path: "/v1/groups/grp_1/duplication"},
	{name: "group join", args: []string{"group", "join", "grp_1"}, operation: "joinGroup", method: "POST", path: "/v1/groups/grp_1/membership"},
	{name: "group notifications show", args: []string{"group", "notifications", "show", "grp_1"}, operation: "showGroupNotificationSettings", method: "GET", path: "/v1/groups/grp_1/notification_settings"},
	{name: "group notifications mute", args: []string{"group", "notifications", "mute", "grp_1"}, operation: "setGroupNotificationSettings", method: "PUT", path: "/v1/groups/grp_1/notification_settings"},
	{name: "group notifications unmute", args: []string{"group", "notifications", "unmute", "grp_1"}, operation: "setGroupNotificationSettings", method: "PUT", path: "/v1/groups/grp_1/notification_settings"},
	{name: "group stats filters", args: []string{"group", "stats", "grp_1", "--date-range", "this_month", "--participant", "part_1"}, operation: "showGroupStats", method: "GET", path: "/v1/groups/grp_1/stats?dateRange=this_month&participantToken=part_1"},
	{name: "group fx-rate", args: []string{"group", "fx-rate", "grp_1", "--from", "USD", "--date", "2026-05-01"}, operation: "showGroupFxRate", method: "GET", path: "/v1/groups/grp_1/fx_rate?date=2026-05-01&from=USD"},
	{name: "group export", args: []string{"group", "export", "grp_1"}, operation: "showGroupCsvExport", method: "GET", path: "/v1/groups/grp_1/csv_export"},

	{name: "participant create", args: []string{"participant", "create", "grp_1", "--data", `{"name":"Ada"}`}, operation: "createParticipant", method: "POST", path: "/v1/groups/grp_1/participants"},
	{name: "participant update reads the lockVersion", args: []string{"participant", "update", "part_1", "grp_1", "--data", `{"membershipRole":"admin"}`}, operation: "updateParticipant", method: "PATCH", path: "/v1/groups/grp_1/participants/part_1", headers: map[string]string{"Expected-Revision": "3"}, reads: 1},
	{name: "participant delete reads the lockVersion", args: []string{"participant", "delete", "part_1", "grp_1"}, operation: "deleteParticipant", method: "DELETE", path: "/v1/groups/grp_1/participants/part_1", headers: map[string]string{"Expected-Revision": "3"}, reads: 1},
	{name: "participant delete with --lock-version", args: []string{"participant", "delete", "part_1", "grp_1", "--lock-version", "1"}, operation: "deleteParticipant", method: "DELETE", path: "/v1/groups/grp_1/participants/part_1", headers: map[string]string{"Expected-Revision": "1"}},
	{name: "participant balance", args: []string{"participant", "balance", "part_1", "grp_1"}, operation: "showParticipantBalance", method: "GET", path: "/v1/groups/grp_1/participants/part_1/balance"},
	{name: "participant claim", args: []string{"participant", "claim", "part_1", "grp_1"}, operation: "claimParticipant", method: "POST", path: "/v1/groups/grp_1/participants/part_1/claim"},
	{name: "participant merge", args: []string{"participant", "merge", "part_1", "grp_1", "--into", "part_2"}, operation: "mergeParticipant", method: "POST", path: "/v1/groups/grp_1/participants/part_1/merge", headers: map[string]string{"Expected-Revision": "3"}, reads: 1},
	{name: "participant move", args: []string{"participant", "move", "part_1", "grp_1", "--position", "2"}, operation: "moveParticipant", method: "PATCH", path: "/v1/groups/grp_1/participants/part_1/position"},

	{name: "category create", args: []string{"category", "create", "grp_1", "--data", `{"name":"Food","emoji":"🍜"}`}, operation: "createGroupCategory", method: "POST", path: "/v1/groups/grp_1/categories"},
	{name: "category update reads the lockVersion", args: []string{"category", "update", "cat_1", "grp_1", "--data", `{"name":"Food"}`}, operation: "updateGroupCategory", method: "PATCH", path: "/v1/groups/grp_1/categories/cat_1", headers: map[string]string{"Expected-Revision": "2"}, reads: 1},
	{name: "category delete reads the lockVersion", args: []string{"category", "delete", "cat_1", "grp_1"}, operation: "deleteGroupCategory", method: "DELETE", path: "/v1/groups/grp_1/categories/cat_1", headers: map[string]string{"Expected-Revision": "2"}, reads: 1},

	{name: "transaction list", args: []string{"transaction", "list", "grp_1"}, operation: "listTransactions", method: "GET", path: "/v1/groups/grp_1/transactions"},
	{name: "transaction list filters", args: []string{"transaction", "list", "grp_1", "--from", "2026-01-01", "--to", "2026-01-31", "--category", "cat_1", "--participant", "part_1", "--take", "10", "--cursor", "abc"}, operation: "listTransactions", method: "GET", path: "/v1/groups/grp_1/transactions?categoryToken=cat_1&cursor=abc&from=2026-01-01&participantToken=part_1&take=10&to=2026-01-31"},
	{name: "transaction show", args: []string{"transaction", "show", "txn_1", "grp_1"}, operation: "showTransaction", method: "GET", path: "/v1/groups/grp_1/transactions/txn_1"},
	{name: "transaction create", args: []string{"transaction", "create", "grp_1", "--data", `{"kind":"expense","description":"Dinner"}`}, operation: "createTransaction", method: "POST", path: "/v1/groups/grp_1/transactions"},
	{name: "transaction update reads the lockVersion", args: []string{"transaction", "update", "txn_1", "grp_1", "--data", `{"kind":"expense","amountCents":1300}`}, operation: "updateTransaction", method: "PATCH", path: "/v1/groups/grp_1/transactions/txn_1", headers: map[string]string{"Expected-Revision": "7"}, reads: 1},
	{name: "transaction delete reads the lockVersion", args: []string{"transaction", "delete", "txn_1", "grp_1"}, operation: "deleteTransaction", method: "DELETE", path: "/v1/groups/grp_1/transactions/txn_1", headers: map[string]string{"Expected-Revision": "7"}, reads: 1},
	{name: "transaction delete with --lock-version", args: []string{"transaction", "delete", "txn_1", "grp_1", "--lock-version", "4"}, operation: "deleteTransaction", method: "DELETE", path: "/v1/groups/grp_1/transactions/txn_1", headers: map[string]string{"Expected-Revision": "4"}},
	{name: "transaction activity", args: []string{"transaction", "activity", "txn_1", "grp_1"}, operation: "listTransactionActivity", method: "GET", path: "/v1/groups/grp_1/transactions/txn_1/activity"},
	{name: "transaction bulk-edit reads the revision", args: []string{"transaction", "bulk-edit", "grp_1", "--data", `{"transactions":[{"token":"txn_1"}],"date":"2026-05-01"}`}, operation: "updateTransactionBatch", method: "PATCH", path: "/v1/groups/grp_1/transaction_batch", reads: 1},
	{name: "transaction bulk-delete", args: []string{"transaction", "bulk-delete", "grp_1", "--data", `{"transactions":[{"token":"txn_1","expectedRevision":2}]}`}, operation: "deleteTransactions", method: "POST", path: "/v1/groups/grp_1/transaction_deletions"},

	{name: "transaction receipt upload", args: []string{"transaction", "receipt", "upload", "txn_1", "testdata/receipt.jpg", "grp_1"}, operation: "updateReceiptPhoto", method: "PUT", path: "/v1/groups/grp_1/transactions/txn_1/receipt_photo"},
	{name: "transaction receipt show", args: []string{"transaction", "receipt", "show", "txn_1", "grp_1"}, operation: "showReceiptPhoto", method: "GET", path: "/v1/groups/grp_1/transactions/txn_1/receipt_photo"},
	{name: "expense is transaction", args: []string{"expense", "receipt", "delete", "txn_1", "grp_1"}, operation: "deleteReceiptPhoto", method: "DELETE", path: "/v1/groups/grp_1/transactions/txn_1/receipt_photo"},

	{name: "reminder send", args: []string{"reminder", "send", "grp_1", "--participant", "part_1"}, operation: "createReminder", method: "POST", path: "/v1/groups/grp_1/reminders"},
	{name: "remind is reminder send", args: []string{"remind", "grp_1", "--participant", "part_1"}, operation: "createReminder", method: "POST", path: "/v1/groups/grp_1/reminders"},
	{name: "reminder schedule weekly", args: []string{"reminder", "schedule", "weekly", "grp_1", "--weekday", "mon"}, operation: "setReminderSchedule", method: "PUT", path: "/v1/groups/grp_1/reminder_schedule"},
	{name: "reminder schedule monthly", args: []string{"reminders", "schedule", "monthly", "grp_1", "--day", "25"}, operation: "setReminderSchedule", method: "PUT", path: "/v1/groups/grp_1/reminder_schedule"},
	{name: "reminder schedule show", args: []string{"reminder", "schedule", "show", "grp_1"}, operation: "showReminderSchedule", method: "GET", path: "/v1/groups/grp_1/reminder_schedule"},
	{name: "reminder send to everyone who owes", args: []string{"remind", "grp_1"}, operation: "createReminder", method: "POST", path: "/v1/groups/grp_1/reminders"},
	{name: "reminder schedule off", args: []string{"reminder", "schedule", "off", "grp_1"}, operation: "deleteReminderSchedule", method: "DELETE", path: "/v1/groups/grp_1/reminder_schedule"},

	{name: "recurring list", args: []string{"recurring", "list", "grp_1"}, operation: "listRecurringExpenses", method: "GET", path: "/v1/groups/grp_1/recurring_expenses"},
	{name: "recurring show", args: []string{"recurring", "show", "rec_1", "grp_1"}, operation: "showRecurringExpense", method: "GET", path: "/v1/groups/grp_1/recurring_expenses/rec_1"},
	{name: "recurring create", args: []string{"recurring", "add", "grp_1", "--data", `{"description":"Rent"}`, "--every", "monthly", "--starts", "2026-12-01"}, operation: "createRecurringExpense", method: "POST", path: "/v1/groups/grp_1/recurring_expenses"},
	{name: "recurring update", args: []string{"recurring", "update", "rec_1", "grp_1", "--interval", "2"}, operation: "updateRecurringExpense", method: "PATCH", path: "/v1/groups/grp_1/recurring_expenses/rec_1"},
	{name: "recurring delete", args: []string{"recurring", "rm", "rec_1", "grp_1"}, operation: "deleteRecurringExpense", method: "DELETE", path: "/v1/groups/grp_1/recurring_expenses/rec_1"},

	{name: "file list", args: []string{"file", "list", "grp_1", "--take", "10"}, operation: "listGroupAttachments", method: "GET", path: "/v1/groups/grp_1/attachments?take=10"},
	{name: "file show", args: []string{"files", "show", "12", "grp_1"}, operation: "showGroupAttachment", method: "GET", path: "/v1/groups/grp_1/attachments/12"},
	{name: "file upload", args: []string{"file", "upload", "testdata/receipt.jpg", "grp_1"}, operation: "createGroupAttachment", method: "POST", path: "/v1/groups/grp_1/attachments"},
	{name: "file delete", args: []string{"file", "rm", "12", "grp_1"}, operation: "deleteGroupAttachment", method: "DELETE", path: "/v1/groups/grp_1/attachments/12"},

	{name: "payment create is a transaction", args: []string{"payment", "create", "grp_1", "--data", `{"fromParticipantToken":"part_1","toParticipantToken":"part_2","amountCents":100}`}, operation: "createTransaction", method: "POST", path: "/v1/groups/grp_1/transactions"},
	{name: "payment update is a transaction", args: []string{"payment", "update", "txn_1", "grp_1", "--data", `{"amountCents":200}`, "--lock-version", "0"}, operation: "updateTransaction", method: "PATCH", path: "/v1/groups/grp_1/transactions/txn_1", headers: map[string]string{"Expected-Revision": "0"}},

	{name: "invitation list", args: []string{"invitation", "list", "grp_1"}, operation: "listInvitations", method: "GET", path: "/v1/groups/grp_1/invitations"},
	{name: "invitation create", args: []string{"invitation", "create", "grp_1", "--role", "editor", "--expires-at", "2026-12-31T23:59:00+09:00"}, operation: "createInvitation", method: "POST", path: "/v1/groups/grp_1/invitations"},
	{name: "invitation revoke", args: []string{"invitation", "revoke", "inv_1", "grp_1"}, operation: "revokeInvitation", method: "DELETE", path: "/v1/groups/grp_1/invitations/inv_1"},
	{name: "invitation accept", args: []string{"invitation", "accept", "inv_1", "grp_1"}, operation: "acceptInvitation", method: "POST", path: "/v1/groups/grp_1/invitations/inv_1/acceptance"},
	{name: "access-request status", args: []string{"access-request", "status", "grp_1"}, operation: "showAccessRequest", method: "GET", path: "/v1/groups/grp_1/access_request"},
	{name: "access-request create", args: []string{"access-request", "create", "grp_1"}, operation: "requestGroupAccess", method: "POST", path: "/v1/groups/grp_1/access_request"},
	{name: "access-request list", args: []string{"access-request", "list", "grp_1"}, operation: "listAccessRequests", method: "GET", path: "/v1/groups/grp_1/access_requests"},
	{name: "access-request approve", args: []string{"access-request", "approve", "42", "grp_1"}, operation: "approveAccessRequest", method: "POST", path: "/v1/groups/grp_1/access_requests/42/approval"},
	{name: "access-request decline", args: []string{"access-request", "decline", "42", "grp_1"}, operation: "declineAccessRequest", method: "DELETE", path: "/v1/groups/grp_1/access_requests/42"},
}

func runContract(t *testing.T, c contract, extraArgs ...string) requestRecord {
	t.Helper()
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	args := append(append(append([]string{}, c.args...), extraArgs...), "--json")
	_, _, requests, err := runRootCommand(t, cfg, contractHandler, args...)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(requests) != c.reads+1 {
		t.Fatalf("expected %d requests, got %d", c.reads+1, len(requests))
	}
	for _, read := range requests[:c.reads] {
		if read.Method != http.MethodGet {
			t.Fatalf("preparatory request %s %s is not a read", read.Method, read.Path)
		}
	}
	return requests[c.reads]
}

func TestCommandContractsUseCurrentAPIRoutes(t *testing.T) {
	for _, c := range contracts {
		t.Run(c.name, func(t *testing.T) {
			got := runContract(t, c)
			if got.Method != c.method || got.Path != c.path {
				t.Fatalf("request = %s %s, want %s %s", got.Method, got.Path, c.method, c.path)
			}
			if auth := got.Header.Get("Authorization"); auth != "Bearer test-token" {
				t.Fatalf("authorization header = %q", auth)
			}
			for name, want := range c.headers {
				if value := got.Header.Get(name); value != want {
					t.Fatalf("%s = %q, want %q", name, value, want)
				}
			}
		})
	}
}

func TestWriteBodiesMatchTheSpec(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"account update is flat", []string{"account", "update", "--data", `{"name":"Ada"}`}, `{"name":"Ada"}`},
		{"invitation create sends role and expiry", []string{"invitation", "create", "grp_1", "--role", "read_only", "--expires-at", "2026-12-31T23:59:00+09:00"}, `{"expiresAt":"2026-12-31T23:59:00+09:00","membershipRole":"read_only"}`},
		{"payment create names its kind", []string{"payment", "create", "grp_1", "--data", `{"fromParticipantToken":"part_1","toParticipantToken":"part_2","amountCents":100}`}, `{"amountCents":100,"fromParticipantToken":"part_1","kind":"payment","toParticipantToken":"part_2"}`},
		{"payment update names its kind", []string{"payment", "update", "txn_1", "grp_1", "--data", `{"amountCents":200}`, "--lock-version", "0"}, `{"amountCents":200,"kind":"payment"}`},
		{"an update keeps the lockVersion out of the body", []string{"group", "update", "grp_1", "--data", `{"visibility":"private_registered"}`, "--lock-version", "3"}, `{"visibility":"private_registered"}`},
		{"participant merge names the winner", []string{"participant", "merge", "part_1", "grp_1", "--into", "part_2", "--lock-version", "3"}, `{"intoParticipantToken":"part_2"}`},
		{"participant move sends the position", []string{"participant", "move", "part_1", "grp_1", "--position", "2"}, `{"position":2}`},
		{"muting sends the whole setting", []string{"group", "notifications", "mute", "grp_1"}, `{"muted":true}`},
		{"leaving sends no body", []string{"group", "leave", "grp_1"}, ``},
		{"invitation create defaults to an empty body", []string{"invitation", "create", "grp_1"}, `{}`},
		{"feedback joins the message", []string{"feedback", "Love", "it"}, `{"body":"Love it"}`},
		{"reminder send names the participant", []string{"remind", "grp_1", "--participant", "part_1"}, `{"participantToken":"part_1"}`},
		{"reminding everyone sends no body", []string{"remind", "grp_1"}, ``},
		{"weekly reminders take a weekday name", []string{"reminder", "schedule", "weekly", "grp_1", "--weekday", "Monday"}, `{"cadence":"weekly","weekday":1}`},
		{"monthly reminders take a day", []string{"reminder", "schedule", "monthly", "grp_1", "--day", "25"}, `{"cadence":"monthly","day":25}`},
		{"recurring flags override the body", []string{"recurring", "create", "grp_1", "--data", `{"description":"Rent","cadence":"weekly"}`, "--every", "monthly", "--interval", "3", "--starts", "2026-12-01"}, `{"cadence":"monthly","description":"Rent","interval":3,"startsOn":"2026-12-01"}`},
		{"recurring update sends only what changes", []string{"recurring", "update", "rec_1", "grp_1", "--starts", "2027-01-01"}, `{"startsOn":"2027-01-01"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
			_, _, requests, err := runRootCommand(t, cfg, contractHandler, tt.args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if requests[0].Body != tt.want {
				t.Fatalf("body = %s, want %s", requests[0].Body, tt.want)
			}
		})
	}
}

func TestNoContentWritesReportTheirOwnResult(t *testing.T) {
	noContent := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"group", "archive", "grp_1"}, `"archived": true`},
		{[]string{"group", "unarchive", "grp_1"}, `"archived": false`},
		{[]string{"group", "delete", "grp_1", "--lock-version", "0"}, `"deleted": true`},
		{[]string{"group", "leave", "grp_1"}, `"left": true`},
		{[]string{"participant", "delete", "part_1", "grp_1", "--lock-version", "0"}, `"deleted": true`},
		{[]string{"category", "delete", "cat_1", "grp_1", "--lock-version", "0"}, `"deleted": true`},
		{[]string{"transaction", "delete", "txn_1", "grp_1", "--lock-version", "0"}, `"deleted": true`},
		{[]string{"transaction", "bulk-delete", "grp_1", "--data", `{"transactions":[{"token":"txn_1","expectedRevision":0}]}`}, `"tokens"`},
		{[]string{"transaction", "receipt", "delete", "txn_1", "grp_1"}, `"receiptPhoto": null`},
		{[]string{"invitation", "revoke", "inv_1", "grp_1"}, `"revoked": true`},
		{[]string{"access-request", "decline", "7", "grp_1"}, `"declined": true`},
		{[]string{"recurring", "delete", "rec_1", "grp_1"}, `"deleted": true`},
		{[]string{"reminder", "schedule", "off", "grp_1"}, `"reminders": "off"`},
		{[]string{"file", "delete", "12", "grp_1"}, `"deleted": true`},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args[:2], " "), func(t *testing.T) {
			cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
			stdout, _, _, err := runRootCommand(t, cfg, noContent, append(tt.args, "--json")...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Fatalf("output missing %s:\n%s", tt.want, stdout)
			}
		})
	}
}

func TestGroupExportWritesTheCSV(t *testing.T) {
	csv := "date,description,amount\n2026-05-23,Dinner,12000\n"
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte(csv))
	}
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}

	stdout, _, requests, err := runRootCommand(t, cfg, handler, "group", "export", "grp_1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requests[0].Path != "/v1/groups/grp_1/csv_export" || requests[0].Header.Get("Accept") != "text/csv" {
		t.Fatalf("got %s with Accept %q", requests[0].Path, requests[0].Header.Get("Accept"))
	}
	if stdout != csv {
		t.Fatalf("stdout = %q, want the raw CSV", stdout)
	}

	path := filepath.Join(t.TempDir(), "trip.csv")
	if _, _, _, err := runRootCommand(t, cfg, handler, "group", "export", "grp_1", "--output", path); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if written, _ := os.ReadFile(path); string(written) != csv {
		t.Fatalf("file = %q, want the raw CSV", written)
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
			_, _, requests, err := runRootCommand(t, cfg, contractHandler, args...)
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
		"categories":[{"token":"cat_1","name":"Food","emoji":"🍜","lockVersion":0}],
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
		{"categories", []string{"category", "list", "grp_1"}, []string{`"categories"`, `"cat_1"`}},
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

func TestPaginationBreadcrumbPreservesTheQuery(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[{"token":"grp_1"}],"pagination":{"hasMore":true,"cursor":"eyJraW5kIjoia2V5c2V0In0"}}`))
	}

	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	stdout, _, _, err := runRootCommand(t, cfg, handler, "group", "list", "--archived", "--take", "25", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout, "wariwari group list --archived --take 25 --cursor eyJraW5kIjoia2V5c2V0In0") {
		t.Fatalf("expected a next-page breadcrumb, got %s", stdout)
	}

	stdout, _, _, err = runRootCommand(t, cfg, handler,
		"transaction", "list", "grp_1",
		"--from", "2026-01-01", "--to", "2026-01-31",
		"--category", "cat_1", "--participant", "part_1", "--take", "10", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := "wariwari transaction list grp_1 --from 2026-01-01 --to 2026-01-31 --category cat_1 --participant part_1 --take 10 --cursor eyJraW5kIjoia2V5c2V0In0"
	if !strings.Contains(stdout, want) {
		t.Fatalf("expected the filtered next-page breadcrumb %q, got %s", want, stdout)
	}
}

func TestProblemDetailsSurfaceCodeAndDetail(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"title":"Conflict",
			"status":409,"detail":"This transaction changed since you read it.","code":"CONFLICT",
			"instance":"req_1","current":{"transaction":{"token":"txn_1","lockVersion":4}}}`))
	}
	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	args := []string{"transaction", "update", "txn_1", "grp_1", "--data", `{"kind":"expense","amountCents":1300}`, "--lock-version", "1"}

	_, stderr, _, err := runRootCommand(t, cfg, handler, args...)
	if err == nil {
		t.Fatal("expected a 409 error")
	}
	if !strings.Contains(err.Error(), "CONFLICT") || !strings.Contains(err.Error(), "This transaction changed since you read it.") {
		t.Fatalf("error = %q, want the code and the detail", err.Error())
	}
	if !strings.Contains(stderr, "The transaction is now at lockVersion 4") {
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
			"errors":{"splits/0/amountCents":"must be greater than 0"}}`))
	}

	cfg := &config.Config{APIURL: "http://example.invalid", Token: "test-token"}
	_, stderr, _, err := runRootCommand(t, cfg, handler, "transaction", "create", "grp_1", "--data", `{"amountCents":0}`)
	if err == nil {
		t.Fatal("expected a 400 error")
	}
	if !strings.Contains(err.Error(), "VALIDATION_FAILED") {
		t.Fatalf("error = %q, want the code", err.Error())
	}
	if !strings.Contains(stderr, "splits/0/amountCents: must be greater than 0") {
		t.Fatalf("stderr missing the field error:\n%s", stderr)
	}
}
