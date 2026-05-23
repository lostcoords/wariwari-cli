package commands

import (
	"encoding/json"
	"fmt"
	"os"
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

func queryWithPaging(cursor string) map[string]string {
	q := map[string]string{}
	if limit > 0 {
		q["take"] = fmt.Sprintf("%d", limit)
	}
	if cursor != "" {
		q["cursor"] = cursor
	}
	return q
}
