package commands

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/lostcoords/wariwari-cli/internal/client"
	"github.com/spf13/cobra"
)

// One command runs per process, so every write command binds the same flag
// variables.
var (
	dataFlag           string
	dataFileFlag       string
	idempotencyKeyFlag string
	lockVersionFlag    int
)

func addDataFlags(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		cmd.Flags().StringVar(&dataFlag, "data", "", "JSON request body")
		cmd.Flags().StringVar(&dataFileFlag, "data-file", "", "Read JSON request body from file")
	}
}

func addIdempotencyFlag(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		cmd.Flags().StringVar(&idempotencyKeyFlag, "idempotency-key", "", "Operation UUID; retrying with the same key and body replays the first outcome")
	}
}

const guardedDeleteHelp = `Delete the record. The API checks its lockVersion: pass --lock-version with
the value you read to have a concurrent edit rejected with 409 CONFLICT, or let
the CLI read the current one first. Deleting a record that is already gone
succeeds again. Returns the refreshed group snapshot.`

func addLockVersionFlag(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		cmd.Flags().IntVar(&lockVersionFlag, "lock-version", 0, "lockVersion you last read (default: read the current one first)")
	}
}

func commandData() (map[string]any, error) {
	if dataFileFlag != "" {
		return parseObjectFile(dataFileFlag)
	}
	return parseObject(dataFlag)
}

func write(body any) client.Write {
	return client.Write{Body: body, IdempotencyKey: idempotencyKeyFlag}
}

// errRecordMissing means the read found no live record to take a lockVersion
// from.
var errRecordMissing = errors.New("record not found")

// revisedWrite adds WariWari-Expected-Revision. Without --lock-version it
// reads the record first, so the write applies to whatever is current; pass
// the lockVersion you acted on to have a concurrent edit rejected instead.
//
// A record the read cannot find is sent at revision 0 and the server decides:
// deleting something already deleted succeeds again, an unknown token is 404,
// and a live record at another revision is 409. A wrong revision can only be
// refused, never applied, so retrying a delete stays idempotent.
func revisedWrite(cmd *cobra.Command, body any, read func(context.Context) (map[string]any, error)) (client.Write, error) {
	w := write(body)
	revision := lockVersionFlag
	if !cmd.Flags().Changed("lock-version") {
		record, err := read(cmd.Context())
		switch {
		case errors.Is(err, errRecordMissing):
			revision = 0
		case err != nil:
			return w, err
		default:
			if revision, err = lockVersion(record); err != nil {
				return w, err
			}
		}
	}
	w.ExpectedRevision = &revision
	return w, nil
}

func lockVersion(record map[string]any) (int, error) {
	value, ok := record["lockVersion"].(float64)
	if !ok {
		return 0, fmt.Errorf("record has no lockVersion; pass --lock-version")
	}
	return int(value), nil
}

// snapshotRecord finds a participant or category in the group snapshot, the
// only read that returns them.
func snapshotRecord(group, key, token string) func(context.Context) (map[string]any, error) {
	return func(ctx context.Context) (map[string]any, error) {
		snapshot, err := groupSnapshot(ctx, group)
		if err != nil {
			return nil, err
		}
		for _, row := range snapshotList(snapshot, key) {
			if record, ok := row.(map[string]any); ok && record["token"] == token {
				return record, nil
			}
		}
		return nil, errRecordMissing
	}
}

func groupRecord(group string) func(context.Context) (map[string]any, error) {
	return func(ctx context.Context) (map[string]any, error) {
		snapshot, err := groupSnapshot(ctx, group)
		return snapshot, missingOn404(err)
	}
}

func transactionRecord(group, token string) func(context.Context) (map[string]any, error) {
	return func(ctx context.Context) (map[string]any, error) {
		data, err := api.Get(ctx, "/groups/"+group+"/transactions/"+token, nil)
		if err != nil {
			return nil, missingOn404(err)
		}
		record, ok := data.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unexpected transaction response")
		}
		return record, nil
	}
}

// absentOn404 tells "this group or record has none" from "no such group or
// record": both are 404, so a 404 is only absence when the parent still reads.
func absentOn404(ctx context.Context, err error, parent func(context.Context) (map[string]any, error)) (bool, error) {
	var apiErr *client.Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		return false, err
	}
	if _, parentErr := parent(ctx); parentErr != nil {
		if errors.Is(parentErr, errRecordMissing) {
			return false, err
		}
		return false, parentErr
	}
	return true, nil
}

func missingOn404(err error) error {
	var apiErr *client.Error
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return errRecordMissing
	}
	return err
}

// groupArg resolves the group token for a command whose group is the
// positional argument at index, falling back to --group and config.
func groupArg(args []string, index int) (string, error) {
	if err := requireAuth(); err != nil {
		return "", err
	}
	return requireGroup(optionalArg(args, index))
}
