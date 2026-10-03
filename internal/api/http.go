package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// statusError is returned when a response does not have the expected status.
// It keeps the response details so a client can add its own hints, such as
// the 7pace client does for rejected Windows credentials.
type statusError struct {
	Status     string
	StatusCode int
	Header     http.Header
	Body       string
}

func (e *statusError) Error() string {
	if e.Body == "" {
		return "request failed: " + e.Status
	}

	return "request failed: " + e.Status + ": " + e.Body
}

// newJSONRequest builds a request to url, encoding body as JSON when it is
// not nil. Authentication is left to the caller.
func newJSONRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// doJSON sends req and decodes the JSON response into result, when result is
// not nil. A status other than 200 OK yields a *statusError.
func doJSON(client *http.Client, req *http.Request, result any) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &statusError{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}
