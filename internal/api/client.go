package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yjwong/lark-cli/internal/auth"
)

const (
	baseURL        = "https://open.larksuite.com/open-apis"
	defaultTimeout = 30 * time.Second
)

// Client is the Lark API client
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// rateLimitErrCode is the Lark API error code for rate-limited requests.
// We retry write requests with exponential backoff when we see this code.
const rateLimitErrCode = 99991400

// writeBackoffSchedule is the backoff sequence (in seconds) for rate-limited
// write requests. After exhausting all entries, we surface the underlying error.
var writeBackoffSchedule = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
}

// isWriteMethod returns true for HTTP methods that mutate state. We only retry
// these on rate-limit errors (idempotent reads don't need it; non-idempotent
// writes are safe to retry because Lark dedupes via client_token / index).
func isWriteMethod(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}

// doRequest performs an authenticated HTTP request with automatic retry on
// rate-limit errors for write methods.
func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	// Ensure we have a valid token
	if err := auth.EnsureValidToken(); err != nil {
		return err
	}

	// Marshal once; we may retry the same payload several times.
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := baseURL + path
	retryable := isWriteMethod(method)

	attempt := func() ([]byte, int, error) {
		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewReader(jsonBody)
		}
		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create request: %w", err)
		}
		token := auth.GetTokenStore().GetAccessToken()
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
		}
		return respBody, resp.StatusCode, nil
	}

	var respBody []byte
	for i := 0; ; i++ {
		body, status, err := attempt()
		if err != nil {
			return err
		}
		respBody = body

		// On write methods, peek at the API error code to detect rate limiting.
		if retryable && i < len(writeBackoffSchedule) {
			var probe BaseResponse
			// Ignore unmarshal errors on the probe; some endpoints return
			// non-JSON or empty bodies on success (the real parse below catches
			// the genuine errors).
			_ = json.Unmarshal(respBody, &probe)
			if probe.Code == rateLimitErrCode || status == http.StatusTooManyRequests {
				time.Sleep(writeBackoffSchedule[i])
				continue
			}
		}
		break
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}
	return nil
}

// Get performs a GET request
func (c *Client) Get(path string, result interface{}) error {
	return c.doRequest("GET", path, nil, result)
}

// Post performs a POST request
func (c *Client) Post(path string, body interface{}, result interface{}) error {
	return c.doRequest("POST", path, body, result)
}

// Patch performs a PATCH request
func (c *Client) Patch(path string, body interface{}, result interface{}) error {
	return c.doRequest("PATCH", path, body, result)
}

// Put performs a PUT request
func (c *Client) Put(path string, body interface{}, result interface{}) error {
	return c.doRequest("PUT", path, body, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(path string, result interface{}) error {
	return c.doRequest("DELETE", path, nil, result)
}

// DeleteWithBody performs a DELETE request with a body
func (c *Client) DeleteWithBody(path string, body interface{}, result interface{}) error {
	return c.doRequest("DELETE", path, body, result)
}

// doRequestWithTenantToken performs an HTTP request using tenant access token,
// with the same rate-limit retry behaviour as doRequest.
func (c *Client) doRequestWithTenantToken(method, path string, body interface{}, result interface{}) error {
	if err := auth.EnsureValidTenantToken(); err != nil {
		return err
	}

	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := baseURL + path
	retryable := isWriteMethod(method)

	attempt := func() ([]byte, int, error) {
		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewReader(jsonBody)
		}
		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create request: %w", err)
		}
		token := auth.GetTenantTokenStore().GetAccessToken()
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
		}
		return respBody, resp.StatusCode, nil
	}

	var respBody []byte
	for i := 0; ; i++ {
		body, status, err := attempt()
		if err != nil {
			return err
		}
		respBody = body
		if retryable && i < len(writeBackoffSchedule) {
			var probe BaseResponse
			_ = json.Unmarshal(respBody, &probe)
			if probe.Code == rateLimitErrCode || status == http.StatusTooManyRequests {
				time.Sleep(writeBackoffSchedule[i])
				continue
			}
		}
		break
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}
	return nil
}

// PostWithTenantToken performs a POST request using tenant access token
func (c *Client) PostWithTenantToken(path string, body interface{}, result interface{}) error {
	return c.doRequestWithTenantToken("POST", path, body, result)
}

// GetWithTenantToken performs a GET request using tenant access token
func (c *Client) GetWithTenantToken(path string, result interface{}) error {
	return c.doRequestWithTenantToken("GET", path, nil, result)
}

// DeleteWithTenantToken performs a DELETE request using tenant access token
func (c *Client) DeleteWithTenantToken(path string, result interface{}) error {
	return c.doRequestWithTenantToken("DELETE", path, nil, result)
}

// PutWithTenantToken performs a PUT request using tenant access token
func (c *Client) PutWithTenantToken(path string, body interface{}, result interface{}) error {
	return c.doRequestWithTenantToken("PUT", path, body, result)
}

// PatchWithTenantToken performs a PATCH request using tenant access token
func (c *Client) PatchWithTenantToken(path string, body interface{}, result interface{}) error {
	return c.doRequestWithTenantToken("PATCH", path, body, result)
}

// DownloadWithTenantToken performs a GET request that returns binary data
// The caller is responsible for closing the returned ReadCloser
func (c *Client) DownloadWithTenantToken(path string) (io.ReadCloser, string, error) {
	// Ensure we have a valid tenant token
	if err := auth.EnsureValidTenantToken(); err != nil {
		return nil, "", err
	}

	url := baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers with tenant token
	token := auth.GetTenantTokenStore().GetAccessToken()
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}

	// Check for error response (non-2xx status)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	return resp.Body, contentType, nil
}

// Download performs a GET request that returns binary data using user access token
// The caller is responsible for closing the returned ReadCloser
func (c *Client) Download(path string) (io.ReadCloser, string, error) {
	// Ensure we have a valid token
	if err := auth.EnsureValidToken(); err != nil {
		return nil, "", err
	}

	url := baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers with user token
	token := auth.GetTokenStore().GetAccessToken()
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}

	// Check for error response (non-2xx status)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	return resp.Body, contentType, nil
}
