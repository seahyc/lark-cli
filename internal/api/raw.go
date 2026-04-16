package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yjwong/lark-cli/internal/auth"
)

// RawAPIResponse holds the raw response from the Lark API
type RawAPIResponse struct {
	StatusCode  int
	Body        []byte
	ContentType string
}

// DoRawRequest executes a raw API request with the specified method, path, and options.
// asUser controls whether to use user token (true) or tenant token (false).
func (c *Client) DoRawRequest(method, path string, params, data map[string]interface{}, formFields []FormField, asUser bool, outputPath string) (*RawAPIResponse, error) {
	// Ensure token
	if asUser {
		if err := auth.EnsureValidToken(); err != nil {
			return nil, err
		}
	} else {
		if err := auth.EnsureValidTenantToken(); err != nil {
			return nil, err
		}
	}

	// Normalize path
	fullURL := path
	if strings.HasPrefix(path, "/open-apis") {
		fullURL = baseURL + strings.TrimPrefix(path, "/open-apis")
	} else if !strings.HasPrefix(path, "http") {
		fullURL = baseURL + path
	}

	// Append query params
	if len(params) > 0 {
		q := make([]string, 0, len(params))
		for k, v := range params {
			q = append(q, fmt.Sprintf("%s=%v", k, v))
		}
		sep := "?"
		if strings.Contains(fullURL, "?") {
			sep = "&"
		}
		fullURL += sep + strings.Join(q, "&")
	}

	var reqBody io.Reader
	var contentType string

	if len(formFields) > 0 {
		// Multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		for _, field := range formFields {
			if field.IsFile {
				file, err := os.Open(field.Value)
				if err != nil {
					return nil, fmt.Errorf("failed to open file %s: %w", field.Value, err)
				}
				defer file.Close()
				part, err := writer.CreateFormFile(field.Key, filepath.Base(field.Value))
				if err != nil {
					return nil, fmt.Errorf("failed to create form file: %w", err)
				}
				if _, err := io.Copy(part, file); err != nil {
					return nil, fmt.Errorf("failed to copy file: %w", err)
				}
			} else {
				if err := writer.WriteField(field.Key, field.Value); err != nil {
					return nil, fmt.Errorf("failed to write field: %w", err)
				}
			}
		}
		writer.Close()
		reqBody = &buf
		contentType = writer.FormDataContentType()
	} else if data != nil {
		jsonBody, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		contentType = "application/json; charset=utf-8"
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set auth header
	var token string
	if asUser {
		token = auth.GetTokenStore().GetAccessToken()
	} else {
		token = auth.GetTenantTokenStore().GetAccessToken()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle file download
	if outputPath != "" {
		outFile, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		defer outFile.Close()
		written, err := io.Copy(outFile, resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to write output: %w", err)
		}
		return &RawAPIResponse{
			StatusCode:  resp.StatusCode,
			Body:        []byte(fmt.Sprintf(`{"success":true,"output_path":%q,"bytes_written":%d}`, outputPath, written)),
			ContentType: resp.Header.Get("Content-Type"),
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &RawAPIResponse{
		StatusCode:  resp.StatusCode,
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// FormField represents a multipart form field
type FormField struct {
	Key    string
	Value  string
	IsFile bool
}

// FetchAPIIndex fetches the Lark API module index from open.larksuite.com/llms.txt
func (c *Client) FetchAPIIndex() ([]byte, error) {
	resp, err := c.httpClient.Get("https://open.larksuite.com/llms.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch API index: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// FetchAPIModuleDocs fetches documentation for a specific API module
func (c *Client) FetchAPIModuleDocs(module string) ([]byte, error) {
	url := fmt.Sprintf("https://open.larksuite.com/llms-docs/zh-CN/llms-%s.txt", module)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch module docs: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
