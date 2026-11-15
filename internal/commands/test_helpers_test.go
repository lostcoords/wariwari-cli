package commands

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/lostcoords/wariwari-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type requestRecord struct {
	Method string
	Path   string
	Header http.Header
	Body   string
}

func runRootCommand(t *testing.T, cfgValue *config.Config, handler http.HandlerFunc, args ...string) (string, string, []requestRecord, error) {
	t.Helper()

	var records []requestRecord
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		records = append(records, requestRecord{
			Method: r.Method,
			Path:   r.URL.RequestURI(),
			Header: r.Header.Clone(),
			Body:   string(body),
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
	resetFlags(rootCmd)
}

// resetFlags returns every flag to its default between in-process runs: cobra
// binds flags to package variables and keeps their "changed" marks, both of
// which outlive one Execute.
func resetFlags(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	}
	cmd.PersistentFlags().VisitAll(reset)
	cmd.Flags().VisitAll(reset)
	for _, child := range cmd.Commands() {
		resetFlags(child)
	}
}
