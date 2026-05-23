package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	UserAgent  string
	HTTPClient *http.Client
}

// Error carries an RFC 9457 Problem Details response. The API always answers
// failures with `code`, `title`, `detail` and `status`, plus `errors` on
// validation failures and `current` (the fresh record) on a 409 CONFLICT.
type Error struct {
	StatusCode int
	Body       map[string]any
}

func (e *Error) Error() string {
	code := e.Code()
	detail := e.Detail()

	switch {
	case code != "" && detail != "":
		return fmt.Sprintf("%s: %s (HTTP %d)", code, detail, e.StatusCode)
	case code != "":
		return fmt.Sprintf("%s (HTTP %d)", code, e.StatusCode)
	case detail != "":
		return fmt.Sprintf("%s (HTTP %d)", detail, e.StatusCode)
	default:
		return fmt.Sprintf("request failed with HTTP %d", e.StatusCode)
	}
}

// Code is the stable UPPER_SNAKE_CASE identifier to switch on.
func (e *Error) Code() string {
	code, _ := e.Body["code"].(string)
	return code
}

func (e *Error) Detail() string {
	if detail, _ := e.Body["detail"].(string); detail != "" {
		return detail
	}
	title, _ := e.Body["title"].(string)
	return title
}

// Problem returns the Problem Details document, so callers can render `errors`
// and the `current` record a stale write was rejected against.
func (e *Error) Problem() map[string]any {
	problem := map[string]any{"status": e.StatusCode}
	for key, value := range e.Body {
		problem[key] = value
	}
	return problem
}

func New(baseURL, token, version string) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		Token:     token,
		UserAgent: "wariwari-cli/" + version,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Get(ctx context.Context, path string, query map[string]string) (any, error) {
	return c.Do(ctx, http.MethodGet, path, query, nil, "")
}

func (c *Client) Post(ctx context.Context, path string, body any, idempotencyKey string) (any, error) {
	return c.Do(ctx, http.MethodPost, path, nil, body, idempotencyKey)
}

func (c *Client) Patch(ctx context.Context, path string, body any) (any, error) {
	return c.Do(ctx, http.MethodPatch, path, nil, body, "")
}

func (c *Client) Delete(ctx context.Context, path string) (any, error) {
	return c.Do(ctx, http.MethodDelete, path, nil, nil, "")
}

func (c *Client) Do(ctx context.Context, method, path string, query map[string]string, body any, idempotencyKey string) (any, error) {
	u, err := url.Parse(c.BaseURL + "/v1" + path)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}

	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Only `DELETE /v1/groups/:group_token` answers 204; every other delete
	// returns the refreshed group snapshot.
	if resp.StatusCode == http.StatusNoContent {
		return map[string]any{"deleted": true}, nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var decoded any
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
	}

	if resp.StatusCode >= 400 {
		body, _ := decoded.(map[string]any)
		return nil, &Error{StatusCode: resp.StatusCode, Body: body}
	}
	return decoded, nil
}
