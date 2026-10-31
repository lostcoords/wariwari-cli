package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	// RetryAfter is the 429 Retry-After header, in seconds, when sent.
	RetryAfter string
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
	if e.RetryAfter != "" {
		problem["retryAfter"] = e.RetryAfter
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
	return c.Do(ctx, http.MethodGet, path, query, nil, nil)
}

func (c *Client) Post(ctx context.Context, path string, body any, idempotencyKey string) (any, error) {
	var header http.Header
	if idempotencyKey != "" {
		header = http.Header{"Idempotency-Key": {idempotencyKey}}
	}
	return c.Do(ctx, http.MethodPost, path, nil, body, header)
}

func (c *Client) Patch(ctx context.Context, path string, body any) (any, error) {
	return c.Do(ctx, http.MethodPatch, path, nil, body, nil)
}

func (c *Client) Delete(ctx context.Context, path string) (any, error) {
	return c.Do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// DeleteRevision deletes a record that guards against stale deletes: the API
// requires the record's last-read lockVersion in WariWari-Expected-Revision.
func (c *Client) DeleteRevision(ctx context.Context, path string, lockVersion int) (any, error) {
	header := http.Header{"WariWari-Expected-Revision": {strconv.Itoa(lockVersion)}}
	return c.Do(ctx, http.MethodDelete, path, nil, nil, header)
}

// GetRaw fetches a non-JSON body, such as the CSV export.
func (c *Client) GetRaw(ctx context.Context, path, accept string) ([]byte, error) {
	resp, data, err := c.send(ctx, http.MethodGet, path, nil, nil, http.Header{"Accept": {accept}})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, apiError(resp, data)
	}
	return data, nil
}

// Do sends a JSON request. A 204 No Content returns nil data.
func (c *Client) Do(ctx context.Context, method, path string, query map[string]string, body any, header http.Header) (any, error) {
	resp, data, err := c.send(ctx, method, path, query, body, header)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, apiError(resp, data)
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var decoded any
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
	}
	return decoded, nil
}

func (c *Client) send(ctx context.Context, method, path string, query map[string]string, body any, header http.Header) (*http.Response, []byte, error) {
	u, err := url.Parse(c.BaseURL + "/v1" + path)
	if err != nil {
		return nil, nil, err
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
			return nil, nil, err
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), r)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	for key, values := range header {
		req.Header[http.CanonicalHeaderKey(key)] = values
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return resp, data, nil
}

func apiError(resp *http.Response, data []byte) *Error {
	var body map[string]any
	_ = json.Unmarshal(data, &body)
	return &Error{StatusCode: resp.StatusCode, Body: body, RetryAfter: resp.Header.Get("Retry-After")}
}
