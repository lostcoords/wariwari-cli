package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/itchyny/gojq"
	"github.com/lostcoords/wariwari-cli/internal/client"
)

type breadcrumb struct {
	Action      string `json:"action"`
	Cmd         string `json:"cmd"`
	Description string `json:"description,omitempty"`
}

type response struct {
	OK          bool         `json:"ok"`
	Data        any          `json:"data,omitempty"`
	Error       any          `json:"error,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Breadcrumbs []breadcrumb `json:"breadcrumbs,omitempty"`
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func printOK(data any, summary string, crumbs ...breadcrumb) error {
	payload := any(response{OK: true, Data: data, Summary: summary, Breadcrumbs: keptCrumbs(crumbs)})
	if quiet || agent {
		payload = data
	}
	if jqFilter != "" {
		return printJQ(payload)
	}
	if countOnly {
		fmt.Fprintln(out, countData(data))
		return nil
	}
	if idsOnly {
		for _, id := range collectIDs(data) {
			fmt.Fprintln(out, id)
		}
		return nil
	}
	if markdown {
		fmt.Fprintln(out, "```json")
		err := writeJSON(out, payload)
		fmt.Fprintln(out, "```")
		return err
	}
	if quiet {
		return writeJSON(out, data)
	}
	return writeJSON(out, payload)
}

func printErr(err error) {
	msg := err.Error()
	if jsonOutput || quiet || agent {
		payload := map[string]any{"message": msg}
		// API failures are RFC 9457 Problem Details. Pass the whole document
		// through so a caller can switch on `code` and reconcile against the
		// `current` record a stale write was rejected against.
		var apiErr *client.Error
		if errors.As(err, &apiErr) {
			payload = apiErr.Problem()
			payload["message"] = msg
		}
		_ = writeJSON(out, response{OK: false, Error: payload})
		return
	}
	fmt.Fprintln(errOut, msg)

	var apiErr *client.Error
	if errors.As(err, &apiErr) {
		if fields, ok := apiErr.Body["errors"].(map[string]any); ok {
			for _, name := range sortedKeys(fields) {
				fmt.Fprintf(errOut, "  %s: %v\n", name, fields[name])
			}
		}
		switch {
		case apiErr.Code() == "CONFLICT" && apiErr.Body["current"] != nil:
			fmt.Fprintln(errOut, "  Re-read the record and retry with the lockVersion from `current`.")
		case apiErr.Code() == "CONFLICT":
			fmt.Fprintln(errOut, "  Re-read the record and retry with its latest lockVersion.")
		case apiErr.RetryAfter != "":
			fmt.Fprintf(errOut, "  Retry after %s seconds.\n", apiErr.RetryAfter)
		}
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func crumb(action, cmd, description string) breadcrumb {
	return breadcrumb{Action: action, Cmd: cmd, Description: description}
}

func keptCrumbs(crumbs []breadcrumb) []breadcrumb {
	var kept []breadcrumb
	for _, c := range crumbs {
		if c.Cmd != "" {
			kept = append(kept, c)
		}
	}
	return kept
}

// nextPageCrumb turns pagination.hasMore/pagination.cursor into a runnable
// follow-up command. It returns a zero breadcrumb on the last page, which
// printOK drops.
func nextPageCrumb(data any, cmdPrefix string) breadcrumb {
	body, ok := data.(map[string]any)
	if !ok {
		return breadcrumb{}
	}
	pagination, ok := body["pagination"].(map[string]any)
	if !ok {
		return breadcrumb{}
	}
	if hasMore, _ := pagination["hasMore"].(bool); !hasMore {
		return breadcrumb{}
	}
	cursor, _ := pagination["cursor"].(string)
	if cursor == "" {
		return breadcrumb{}
	}
	return crumb("next-page", cmdPrefix+" "+cursor, "Fetch the next page")
}

func printJQ(payload any) error {
	payload, err := jsonShape(payload)
	if err != nil {
		return err
	}
	query, err := gojq.Parse(jqFilter)
	if err != nil {
		return err
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return err
	}
	iter := code.Run(payload)
	var values []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return err
		}
		values = append(values, v)
	}
	if len(values) == 1 {
		return writeJSON(out, values[0])
	}
	return writeJSON(out, values)
}

func jsonShape(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// collectionKeys are the camelCase array fields the API nests rows under, in
// the order --count and --ids-only should prefer them.
var collectionKeys = []string{"groups", "transactions", "participants", "groupCategories", "activity", "suggestedPayments", "accessRequests"}

func countData(v any) int {
	switch data := v.(type) {
	case []any:
		return len(data)
	case []map[string]any:
		return len(data)
	case map[string]any:
		for _, key := range collectionKeys {
			if nested, ok := data[key]; ok {
				return countData(nested)
			}
		}
		return len(data)
	default:
		return 0
	}
}

func collectIDs(v any) []string {
	var ids []string
	var walk func(any)
	walk = func(value any) {
		switch data := value.(type) {
		case []any:
			for _, item := range data {
				walk(item)
			}
		case []map[string]any:
			for _, item := range data {
				walk(item)
			}
		case map[string]any:
			if id := idValue(data); id != "" {
				ids = append(ids, id)
				return
			}
			for _, key := range collectionKeys {
				if nested, ok := data[key]; ok {
					walk(nested)
				}
			}
		}
	}
	walk(v)
	sort.Strings(ids)
	return ids
}

func idValue(m map[string]any) string {
	for _, key := range []string{"token", "id"} {
		switch v := m[key].(type) {
		case string:
			return v
		case float64:
			return fmt.Sprintf("%.0f", v)
		}
	}
	return ""
}
