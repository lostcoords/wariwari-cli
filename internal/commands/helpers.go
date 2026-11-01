package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func parseObject(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}
	return body, nil
}

func parseObjectFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseObject(string(data))
}

// lockVersion is the record's last-read lockVersion for a guarded delete,
// sent as WariWari-Expected-Revision.
var lockVersion int

const guardedDeleteHelp = `Delete the record. --lock-version is required: pass the lockVersion from the
record you just read (the group snapshot). A stale value returns 409 CONFLICT.
Returns the refreshed group snapshot.`

func addLockVersionFlag(cmd *cobra.Command) {
	cmd.Flags().IntVar(&lockVersion, "lock-version", 0, "The record's current lockVersion (required)")
	_ = cmd.MarkFlagRequired("lock-version")
}

// decodeInto converts a decoded JSON value into a typed struct.
func decodeInto(data any, target any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// bodyFrom reads a command's JSON body from --data-file, else --data.
func bodyFrom(raw, path string) (map[string]any, error) {
	if path != "" {
		return parseObjectFile(path)
	}
	return parseObject(raw)
}

// pagingQuery builds the keyset pagination query. `take` caps at 100 server
// side; the global --limit flag is the fallback when --take is not given.
func pagingQuery(take int, cursor string) map[string]string {
	q := map[string]string{}
	if take <= 0 {
		take = limit
	}
	if take > 0 {
		q["take"] = fmt.Sprintf("%d", take)
	}
	if cursor != "" {
		q["cursor"] = cursor
	}
	return q
}

func commandFlag(command, name, value string) string {
	if value == "" {
		return command
	}
	return command + " --" + name + " " + value
}

func commandTakeFlag(command string, take int) string {
	if take <= 0 {
		take = limit
	}
	if take <= 0 {
		return command
	}
	return commandFlag(command, "take", fmt.Sprintf("%d", take))
}

// groupSnapshot fetches `GET /v1/groups/:group_token`, the full group render
// state. Participants, categories and balances have no list endpoints of their
// own — they are read out of this one payload.
func groupSnapshot(ctx context.Context, group string) (map[string]any, error) {
	data, err := api.Get(ctx, "/groups/"+group, nil)
	if err != nil {
		return nil, err
	}
	snapshot, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected group snapshot response")
	}
	return snapshot, nil
}

func snapshotList(snapshot map[string]any, key string) []any {
	rows, _ := snapshot[key].([]any)
	if rows == nil {
		rows = []any{}
	}
	return rows
}

func firstArg(args []string) string {
	return optionalArg(args, 0)
}

func optionalArg(args []string, index int) string {
	if len(args) <= index {
		return ""
	}
	return args[index]
}
