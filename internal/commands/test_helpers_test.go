package commands

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/spf13/cobra"
)

type requestRecord struct {
	Method string
	Path   string
	Header http.Header
}

func runRootCommand(t *testing.T, cfgValue *config.Config, handler http.HandlerFunc, args ...string) (string, string, []requestRecord, error) {
	t.Helper()

	var records []requestRecord
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records = append(records, requestRecord{
			Method: r.Method,
			Path:   r.URL.RequestURI(),
			Header: r.Header.Clone(),
		})
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	resetCommandGlobals()
	cfg = cfgValue
	cfg.APIURL = server.URL
	api = client.New(server.URL, cfg.Token, "test")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out = &stdout
	errOut = &stderr
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(args)
	setContextOnAll(context.Background(), rootCmd)

	// Mirror Execute(): rendering the error is part of the behaviour under test.
	err := rootCmd.Execute()
	if err != nil {
		printErr(err)
	}
	return stdout.String(), stderr.String(), records, err
}

func setContextOnAll(ctx context.Context, cmd *cobra.Command) {
	cmd.SetContext(ctx)
	for _, child := range cmd.Commands() {
		setContextOnAll(ctx, child)
	}
}

func resetCommandGlobals() {
	cfg = nil
	api = nil
	out = &bytes.Buffer{}
	errOut = &bytes.Buffer{}
	tokenFlag = ""
	profileFlag = ""
	apiURLFlag = ""
	groupFlag = ""
	jsonOutput = false
	quiet = false
	agent = false
	idsOnly = false
	countOnly = false
	markdown = false
	styled = false
	verbose = false
	jqFilter = ""
	limit = 0
	groupCursor = ""
	groupTake = 0
	groupArchived = false
	groupBody = ""
	groupBodyFile = ""
	groupIdempotencyKey = ""
	participantBody = ""
	participantBodyFile = ""
	categoryBody = ""
	categoryBodyFile = ""
	transactionBody = ""
	transactionBodyFile = ""
	transactionCursor = ""
	transactionTake = 0
	transactionFrom = ""
	transactionTo = ""
	transactionCategory = ""
	transactionParticipant = ""
}
