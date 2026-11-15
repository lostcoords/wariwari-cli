package commands

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The public OpenAPI bundle, vendored so these checks run without the server
// checkout. Refresh it with `make sync-spec`.
const vendoredSpec = "testdata/wariwari.openapi.yaml"

// serverSpec is the server checkout's bundle, when it sits beside this repo.
var serverSpec = filepath.Join("..", "..", "..", "wariwari", "openapi", "dist", "wariwari.bundled.yaml")

// Operations the CLI deliberately does not wrap, with the reason.
var unwrappedOperations = map[string]string{
	"startCliAuthorization": "auth login drives the device flow; see TestAuthLoginRunsTheDeviceFlow",
	"claimCliAuthorization": "auth login drives the device flow; see TestAuthLoginRunsTheDeviceFlow",
}

type specOperation struct {
	Method          string
	Path            string
	RequiredHeaders []string
	Headers         []string
}

func loadOperations(t *testing.T, path string) map[string]specOperation {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var doc struct {
		Paths      map[string]map[string]yaml.Node `yaml:"paths"`
		Components struct {
			Parameters map[string]specParameter `yaml:"parameters"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse spec: %v", err)
	}

	operations := map[string]specOperation{}
	for path, methods := range doc.Paths {
		for method, node := range methods {
			var op struct {
				OperationID string          `yaml:"operationId"`
				Parameters  []specParameter `yaml:"parameters"`
			}
			if err := node.Decode(&op); err != nil || op.OperationID == "" {
				continue
			}
			entry := specOperation{Method: strings.ToUpper(method), Path: path}
			for _, param := range op.Parameters {
				if ref := strings.TrimPrefix(param.Ref, "#/components/parameters/"); ref != param.Ref {
					param = doc.Components.Parameters[ref]
				}
				if param.In != "header" {
					continue
				}
				entry.Headers = append(entry.Headers, param.Name)
				if param.Required {
					entry.RequiredHeaders = append(entry.RequiredHeaders, param.Name)
				}
			}
			sort.Strings(entry.Headers)
			sort.Strings(entry.RequiredHeaders)
			operations[op.OperationID] = entry
		}
	}
	return operations
}

type specParameter struct {
	Ref      string `yaml:"$ref"`
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
}

func pathPattern(template string) *regexp.Regexp {
	literals := regexp.MustCompile(`\{[^}]+\}`).Split(template, -1)
	for i, literal := range literals {
		literals[i] = regexp.QuoteMeta(literal)
	}
	return regexp.MustCompile("^" + strings.Join(literals, "[^/]+") + "$")
}

func TestContractsMatchTheSpec(t *testing.T) {
	operations := loadOperations(t, vendoredSpec)

	for _, c := range contracts {
		t.Run(c.name, func(t *testing.T) {
			op, ok := operations[c.operation]
			if !ok {
				t.Fatalf("operation %q is not in the spec", c.operation)
			}
			path, _, _ := strings.Cut(c.path, "?")
			if c.method != op.Method || !pathPattern(op.Path).MatchString(path) {
				t.Fatalf("%s %s does not match %s %s", c.method, path, op.Method, op.Path)
			}
			got := runContract(t, c)
			for _, header := range op.RequiredHeaders {
				if got.Header.Get(header) == "" {
					t.Fatalf("required header %s was not sent", header)
				}
			}
		})
	}
}

// A write that declares Idempotency-Key must let the caller send one, and a
// write that does not declare it must not grow the flag.
func TestIdempotencyKeyFollowsTheSpec(t *testing.T) {
	operations := loadOperations(t, vendoredSpec)

	for _, c := range contracts {
		op := operations[c.operation]
		declared := false
		for _, header := range op.Headers {
			declared = declared || header == "Idempotency-Key"
		}
		cmd, _, err := rootCmd.Find(c.args)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		hasFlag := cmd.Flags().Lookup("idempotency-key") != nil
		if declared != hasFlag {
			t.Errorf("%s: spec declares Idempotency-Key = %v, command has --idempotency-key = %v", c.name, declared, hasFlag)
			continue
		}
		if !declared {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			got := runContract(t, c, "--idempotency-key", "1b4e28ba-2fa1-11d2-883f-0016d3cca427")
			if got.Header.Get("Idempotency-Key") != "1b4e28ba-2fa1-11d2-883f-0016d3cca427" {
				t.Fatalf("Idempotency-Key = %q", got.Header.Get("Idempotency-Key"))
			}
		})
	}
}

func TestEveryPublicOperationHasACommand(t *testing.T) {
	covered := map[string]bool{}
	for _, c := range contracts {
		covered[c.operation] = true
	}
	for id := range loadOperations(t, vendoredSpec) {
		if !covered[id] && unwrappedOperations[id] == "" {
			t.Errorf("operation %s has no command contract", id)
		}
	}
}

// Fails when the server's bundle changed an operation, path or header since
// the vendored copy; run `make sync-spec` and update the commands.
func TestVendoredSpecMatchesTheServer(t *testing.T) {
	if _, err := os.Stat(serverSpec); err != nil {
		t.Skip("server checkout not found beside this repo")
	}
	vendored := loadOperations(t, vendoredSpec)
	server := loadOperations(t, serverSpec)
	if !reflect.DeepEqual(vendored, server) {
		for id, op := range server {
			if !reflect.DeepEqual(vendored[id], op) {
				t.Errorf("%s: vendored %+v, server %+v", id, vendored[id], op)
			}
		}
		for id := range vendored {
			if _, ok := server[id]; !ok {
				t.Errorf("%s: removed from the server spec", id)
			}
		}
		t.Fatal("vendored spec is stale; run `make sync-spec`")
	}
}
