package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/itchyny/gojq"
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
	return enc.Encode(v)
}

func printOK(data any, summary string, crumbs ...breadcrumb) error {
	payload := any(response{OK: true, Data: data, Summary: summary, Breadcrumbs: crumbs})
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
		_ = writeJSON(out, response{OK: false, Error: map[string]any{"message": msg}})
		return
	}
	fmt.Fprintln(errOut, msg)
}

func crumb(action, cmd, description string) breadcrumb {
	return breadcrumb{Action: action, Cmd: cmd, Description: description}
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

func countData(v any) int {
	switch data := v.(type) {
	case []any:
		return len(data)
	case []map[string]any:
		return len(data)
	case map[string]any:
		for _, key := range []string{"groups", "participants", "group_categories", "transactions", "activity", "api_keys", "balances", "suggested_payments"} {
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
			for _, key := range []string{"groups", "participants", "group_categories", "transactions", "activity", "api_keys", "balances", "suggested_payments"} {
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
