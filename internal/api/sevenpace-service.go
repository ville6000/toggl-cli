package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/ville6000/toggl-cli/internal/data"
)

// CreateWorkLog posts a single worklog to 7pace Timetracker.
func (c *SevenPaceClient) CreateWorkLog(workLog data.SevenPaceWorkLog) (*data.SevenPaceWorkLog, error) {
	req, err := c.newRequest(http.MethodPost, "/workLogs?api-version=3.0", workLog)
	if err != nil {
		return nil, err
	}

	var created data.SevenPaceWorkLog
	if reqErr := c.doRequest(req, http.StatusOK, &created); reqErr != nil {
		return nil, reqErr
	}

	return &created, nil
}

func (c *SevenPaceClient) newRequest(method, endpoint string, body any) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, buf)
	if err != nil {
		return nil, err
	}

	c.setDefaultRequestHeaders(req)

	return req, nil
}

func (c *SevenPaceClient) doRequest(req *http.Request, expectedStatus int, result any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("request failed: %s (server auth schemes: %q): %s\n"+
				"check sevenpace.domain/username/password in your config; the Windows credentials were rejected",
				resp.Status, resp.Header.Get("WWW-Authenticate"), strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// setDefaultRequestHeaders sets the Basic-auth credentials that the NTLM
// negotiator uses for the handshake. The username is qualified with the domain
// (DOMAIN\user) when a domain is configured.
func (c *SevenPaceClient) setDefaultRequestHeaders(req *http.Request) {
	user := c.Username
	if c.Domain != "" {
		user = c.Domain + "\\" + c.Username
	}

	req.SetBasicAuth(user, c.Password)
	req.Header.Set("Content-Type", "application/json")
}
