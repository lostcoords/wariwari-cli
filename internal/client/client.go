package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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

// Error carries the API's Problem body. Every failure answers with `code`,
// `title`, `detail`, `status` and `instance`, plus `errors` on validation
// failures and, on a 409 CONFLICT, `current`: the fresh record under the key
// that names its kind (`transaction`, `group`, ...).
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

// Write carries what a mutation may send beyond its JSON body.
type Write struct {
	Body any
	// IdempotencyKey is an operation UUID; a retry with the same key and body
	// replays the stored outcome instead of writing twice.
	IdempotencyKey string
	// ExpectedRevision is the target record's lockVersion as last observed,
	// sent as Expected-Revision on every update, delete and merge of a record
	// that has one. Nil sends no header.
	ExpectedRevision *int
}

func (w Write) header() http.Header {
	header := http.Header{}
	if w.IdempotencyKey != "" {
		header.Set("Idempotency-Key", w.IdempotencyKey)
	}
	if w.ExpectedRevision != nil {
		header.Set("Expected-Revision", strconv.Itoa(*w.ExpectedRevision))
	}
	return header
}

func (c *Client) Post(ctx context.Context, path string, w Write) (any, error) {
	return c.Do(ctx, http.MethodPost, path, nil, w.Body, w.header())
}

func (c *Client) Patch(ctx context.Context, path string, w Write) (any, error) {
	return c.Do(ctx, http.MethodPatch, path, nil, w.Body, w.header())
}

func (c *Client) Put(ctx context.Context, path string, w Write) (any, error) {
	return c.Do(ctx, http.MethodPut, path, nil, w.Body, w.header())
}

func (c *Client) Delete(ctx context.Context, path string, w Write) (any, error) {
	return c.Do(ctx, http.MethodDelete, path, nil, w.Body, w.header())
}

// File is one file part of a multipart upload.
type File struct {
	Field       string
	Name        string
	ContentType string
	Content     []byte
}

// Upload sends one file as multipart/form-data. w carries the write's headers;
// its Body is not sent.
func (c *Client) Upload(ctx context.Context, method, path string, file File, w Write) (any, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part := textproto.MIMEHeader{}
	part.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, file.Field, file.Name))
	part.Set("Content-Type", file.ContentType)
	dst, err := form.CreatePart(part)
	if err != nil {
		return nil, err
	}
	if _, err := dst.Write(file.Content); err != nil {
		return nil, err
	}
	if err := form.Close(); err != nil {
		return nil, err
	}
	header := w.header()
	header.Set("Content-Type", form.FormDataContentType())
	return c.decode(c.send(ctx, method, path, nil, &body, header))
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
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(data)
		header = header.Clone()
		if header == nil {
			header = http.Header{}
		}
		header.Set("Content-Type", "application/json")
	}
	return c.decode(c.send(ctx, method, path, query, r, header))
}

func (c *Client) decode(resp *http.Response, data []byte, err error) (any, error) {
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

func (c *Client) send(ctx context.Context, method, path string, query map[string]string, body io.Reader, header http.Header) (*http.Response, []byte, error) {
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

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
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
