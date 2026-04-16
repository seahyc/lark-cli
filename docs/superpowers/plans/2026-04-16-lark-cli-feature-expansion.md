# Lark CLI Feature Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add raw API passthrough + ergonomic commands for tasks, bitable, group chat, message search, meetings, wiki, and real-time events.

**Architecture:** Foundation layer (`lark api`) provides generic HTTP access to all 2500+ Lark endpoints. Ergonomic commands layer on top with input parsing, auto-detection, and structured output. Both reuse the existing `client.go` user/tenant token infrastructure.

**Tech Stack:** Go 1.24, cobra CLI framework, Lark Open API (open.larksuite.com)

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `internal/cmd/api.go` | `lark api` command — raw API passthrough + `list` discovery |
| `internal/api/raw.go` | Generic HTTP execution for raw API (JSON, multipart, download) |
| `internal/api/tasks_write.go` | Task write methods (create/update/complete/subtask/comment) |
| `internal/api/chats_write.go` | Chat CRUD + member management methods |
| `internal/api/bitable_write.go` | Bitable write methods (table/field/record CRUD, batch ops) |
| `internal/api/search.go` | Message search + batch get + forward methods |
| `internal/api/meetings.go` | Video conferencing API methods |
| `internal/api/wiki.go` | Wiki space + node management methods |
| `internal/api/events.go` | WebSocket event subscription |
| `internal/cmd/meetings.go` | `lark meetings` command group |
| `internal/cmd/wiki.go` | `lark wiki` command group |
| `internal/cmd/events.go` | `lark events` command group |

### Modified Files
| File | Changes |
|------|---------|
| `internal/cmd/root.go` | Register `apiCmd`, `meetingsCmd`, `wikiCmd`, `eventsCmd` |
| `internal/cmd/task.go` | Add create/update/complete/reopen/assign/subtask/comment/remind/tasklist subcommands |
| `internal/cmd/chat.go` | Add create/get/update/members/pin/link subcommands |
| `internal/cmd/msg.go` | Add search/get/forward/merge-forward subcommands |
| `internal/cmd/bitable.go` | Add table/field/record CRUD subcommands |
| `internal/scopes/scopes.go` | Add new scopes for tasks, bitable, chat, meetings, wiki |
| `internal/api/types.go` | Add new type definitions for tasks, chats, bitable, meetings, wiki, events |

---

## Task 1: Raw API Passthrough — Types & API Layer

**Files:**
- Create: `internal/api/raw.go`

- [ ] **Step 1: Create raw.go with RawRequest and RawResponse types**

```go
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
```

- [ ] **Step 2: Build to verify compilation**

Run: `cd /Users/yingcong/Code/lark-cli && go build ./internal/api/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/api/raw.go
git commit -m "Add raw API request infrastructure for lark api passthrough"
```

---

## Task 2: Raw API Passthrough — Command Layer

**Files:**
- Create: `internal/cmd/api.go`
- Modify: `internal/cmd/root.go`

- [ ] **Step 1: Create api.go with the lark api command**

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var (
	apiAs     string
	apiParams string
	apiData   string
	apiForms  []string
	apiOutput string
)

var apiCmd = &cobra.Command{
	Use:   "api <METHOD> <path>",
	Short: "Raw API passthrough for any Lark endpoint",
	Long: `Execute raw HTTP requests against any Lark Open API endpoint.

Examples:
  # GET with query params
  lark api GET /open-apis/im/v1/chats --params '{"page_size":20}' --as user

  # POST with JSON body
  lark api POST /open-apis/task/v2/tasks --data '{"summary":"Ship it"}' --as user

  # File upload (multipart)
  lark api POST /open-apis/im/v1/files --form file=@./report.pdf --form file_type=stream

  # File download
  lark api GET /open-apis/im/v1/messages/om_xxx/resources/file_xxx --output ./download.pdf

  # DELETE
  lark api DELETE /open-apis/im/v1/messages/om_xxx --as bot`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle "lark api list" subcommand
		if args[0] == "list" {
			runAPIList(args[1:])
			return
		}

		if len(args) < 2 {
			output.Fatalf("VALIDATION_ERROR", "usage: lark api <METHOD> <path>")
		}

		method := strings.ToUpper(args[0])
		path := args[1]

		if apiAs != "" && apiAs != "bot" && apiAs != "user" {
			output.Fatalf("VALIDATION_ERROR", "--as must be 'bot' or 'user'")
		}
		asUser := apiAs == "user"

		// Parse --params
		var params map[string]interface{}
		if apiParams != "" {
			if err := json.Unmarshal([]byte(apiParams), &params); err != nil {
				output.Fatalf("VALIDATION_ERROR", "invalid --params JSON: %v", err)
			}
		}

		// Parse --data
		var data map[string]interface{}
		if apiData != "" {
			if err := json.Unmarshal([]byte(apiData), &data); err != nil {
				output.Fatalf("VALIDATION_ERROR", "invalid --data JSON: %v", err)
			}
		}

		// Parse --form fields
		var formFields []api.FormField
		for _, f := range apiForms {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				output.Fatalf("VALIDATION_ERROR", "invalid --form format: %s (use key=value or key=@./path)", f)
			}
			field := api.FormField{Key: parts[0]}
			if strings.HasPrefix(parts[1], "@") {
				field.Value = strings.TrimPrefix(parts[1], "@")
				field.IsFile = true
			} else {
				field.Value = parts[1]
			}
			formFields = append(formFields, field)
		}

		client := api.NewClient()
		resp, err := client.DoRawRequest(method, path, params, data, formFields, asUser, apiOutput)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		// Print response
		if apiOutput != "" {
			// File download — response body is already the success JSON
			fmt.Println(string(resp.Body))
		} else {
			// Try to pretty-print JSON, fall back to raw
			var prettyJSON map[string]interface{}
			if err := json.Unmarshal(resp.Body, &prettyJSON); err == nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(prettyJSON)
			} else {
				fmt.Println(string(resp.Body))
			}
		}
	},
}

func runAPIList(args []string) {
	client := api.NewClient()

	if len(args) == 0 {
		// List all API domains
		body, err := client.FetchAPIIndex()
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		fmt.Println(string(body))
	} else {
		// List endpoints for a specific domain
		body, err := client.FetchAPIModuleDocs(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		fmt.Println(string(body))
	}
}

func init() {
	apiCmd.Flags().StringVar(&apiAs, "as", "bot", "Identity: 'bot' (default) or 'user'")
	apiCmd.Flags().StringVar(&apiParams, "params", "", "JSON query parameters")
	apiCmd.Flags().StringVar(&apiData, "data", "", "JSON request body")
	apiCmd.Flags().StringSliceVar(&apiForms, "form", nil, "Multipart form field (repeatable). key=value or key=@./path for files")
	apiCmd.Flags().StringVar(&apiOutput, "output", "", "Save response body to file (for binary downloads)")
}
```

- [ ] **Step 2: Register apiCmd in root.go**

Add to the `init()` function in `internal/cmd/root.go`:
```go
rootCmd.AddCommand(apiCmd)
```

- [ ] **Step 3: Build and test**

Run: `cd /Users/yingcong/Code/lark-cli && go build -o lark ./cmd/lark/`
Then: `./lark api --help`
Expected: Shows usage with examples

Test with a real call:
```bash
./lark api GET /open-apis/im/v1/chats --params '{"page_size":5}' --as user
```
Expected: JSON response with chat list

Test list:
```bash
./lark api list
```
Expected: Lark API module index text

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/api.go internal/cmd/root.go
git commit -m "Add raw API passthrough command (lark api)"
```

---

## Task 3: Update Scopes

**Files:**
- Modify: `internal/scopes/scopes.go`

- [ ] **Step 1: Add new scopes and expand existing groups**

In `internal/scopes/scopes.go`, update the `Groups` map:

```go
"messages": {
	Name:        "messages",
	Description: "Chat and messaging",
	Scopes:      []string{
		"im:message:readonly", "im:message", "im:message:send_as_bot",
		"im:message.send_as_user", "im:resource",
		"im:message.reactions:read", "im:message.reactions:write_only",
		"im:chat", "im:chat:readonly",
		"im:message:search",
	},
	Commands:    []string{"msg", "chat"},
},
"tasks": {
	Name:        "tasks",
	Description: "Lark Tasks management",
	Scopes:      []string{
		"task:task", "task:task:readonly",
		"task:tasklist", "task:tasklist:readonly",
	},
	Commands:    []string{"task"},
},
"bitable": {
	Name:        "bitable",
	Description: "Lark Bitable (database) access",
	Scopes:      []string{"bitable:app:readonly", "bitable:app"},
	Commands:    []string{"bitable"},
},
"documents": {
	Name:        "documents",
	Description: "Lark Docs and Drive access",
	Scopes:      []string{
		"docx:document:readonly", "docx:document", "docx:document:create",
		"docs:doc:readonly", "docs:document.content:read", "docs:document.comment:read",
		"drive:drive:readonly", "drive:drive",
		"wiki:wiki:readonly", "wiki:wiki",
		"space:document:retrieve",
	},
	Commands:    []string{"doc", "wiki"},
},
```

Add new groups:
```go
"meetings": {
	Name:        "meetings",
	Description: "Video conferencing records and notes",
	Scopes:      []string{"vc:meeting:readonly", "vc:record:readonly"},
	Commands:    []string{"meetings"},
},
```

- [ ] **Step 2: Build to verify**

Run: `cd /Users/yingcong/Code/lark-cli && go build -o lark ./cmd/lark/`
Then: `./lark auth scopes`
Expected: Shows updated scope groups including meetings

- [ ] **Step 3: Commit**

```bash
git add internal/scopes/scopes.go
git commit -m "Add scopes for tasks, bitable, chat, meetings, wiki write"
```

---

## Task 4: Group Chat Operations — API Layer

**Files:**
- Create: `internal/api/chats_write.go`
- Modify: `internal/api/types.go` (add chat types)

- [ ] **Step 1: Add chat types to types.go**

Append to `internal/api/types.go`:

```go
// --- Chat Management Types ---

// CreateChatRequest represents a request to create a group chat
type CreateChatRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	UserIDList  []string `json:"user_id_list,omitempty"`
	ChatMode    string   `json:"chat_mode,omitempty"` // group
	ChatType    string   `json:"chat_type,omitempty"` // private, public
}

// CreateChatResponse is the API response for chat creation
type CreateChatResponse struct {
	BaseResponse
	Data *struct {
		ChatID string `json:"chat_id"`
	} `json:"data"`
}

// UpdateChatRequest represents a request to update a chat
type UpdateChatRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// GetChatResponse is the API response for getting chat info
type GetChatResponse struct {
	BaseResponse
	Data *ChatDetail `json:"data"`
}

// ChatDetail contains detailed chat information
type ChatDetail struct {
	ChatID      string `json:"chat_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
	ChatMode    string `json:"chat_mode"`
	ChatType    string `json:"chat_type"`
	External    bool   `json:"external"`
	ChatStatus  string `json:"chat_status"`
}

// ChatMember represents a chat member
type ChatMember struct {
	MemberIDType string `json:"member_id_type,omitempty"`
	MemberID     string `json:"member_id"`
	Name         string `json:"name,omitempty"`
}

// ChatMembersResponse is the API response for listing chat members
type ChatMembersResponse struct {
	BaseResponse
	Data *struct {
		Items     []ChatMember `json:"items"`
		HasMore   bool         `json:"has_more"`
		PageToken string       `json:"page_token"`
	} `json:"data"`
}

// ModifyChatMembersRequest represents a request to add/remove chat members
type ModifyChatMembersRequest struct {
	IDList []string `json:"id_list"`
}

// PinMessageRequest represents a request to pin a message
type PinMessageRequest struct {
	MessageID string `json:"message_id"`
}

// PinMessageResponse is the API response for pinning a message
type PinMessageResponse struct {
	BaseResponse
	Data *struct {
		Pin *PinnedMessage `json:"pin"`
	} `json:"data"`
}

// PinnedMessage represents a pinned message
type PinnedMessage struct {
	MessageID    string `json:"message_id"`
	ChatID       string `json:"chat_id"`
	OperatorID   string `json:"operator_id"`
	CreateTime   string `json:"create_time"`
}

// ListPinsResponse is the API response for listing pinned messages
type ListPinsResponse struct {
	BaseResponse
	Data *struct {
		Items     []PinnedMessage `json:"items"`
		HasMore   bool            `json:"has_more"`
		PageToken string          `json:"page_token"`
	} `json:"data"`
}

// ChatLinkResponse is the API response for getting a chat link
type ChatLinkResponse struct {
	BaseResponse
	Data *struct {
		ShareLink string `json:"share_link"`
	} `json:"data"`
}
```

- [ ] **Step 2: Create chats_write.go with API methods**

```go
package api

import (
	"fmt"
	"net/url"
)

// CreateChat creates a new group chat
func (c *Client) CreateChat(req *CreateChatRequest, asUser bool) (*CreateChatResponse, error) {
	path := "/im/v1/chats?set_bot_manager=false"
	var resp CreateChatResponse
	if asUser {
		if err := c.Post(path, req, &resp); err != nil {
			return nil, err
		}
	} else {
		if err := c.PostWithTenantToken(path, req, &resp); err != nil {
			return nil, err
		}
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetChat retrieves detailed chat information
func (c *Client) GetChat(chatID string, asUser bool) (*ChatDetail, error) {
	path := fmt.Sprintf("/im/v1/chats/%s", chatID)
	var resp GetChatResponse
	if asUser {
		if err := c.Get(path, &resp); err != nil {
			return nil, err
		}
	} else {
		if err := c.GetWithTenantToken(path, &resp); err != nil {
			return nil, err
		}
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// UpdateChat updates a chat's name or description
func (c *Client) UpdateChat(chatID string, req *UpdateChatRequest, asUser bool) error {
	path := fmt.Sprintf("/im/v1/chats/%s", chatID)
	var resp BaseResponse
	if asUser {
		if err := c.Put(path, req, &resp); err != nil {
			return err
		}
	} else {
		if err := c.PutWithTenantToken(path, req, &resp); err != nil {
			return err
		}
	}
	return resp.Err()
}

// ListChatMembers lists members of a chat
func (c *Client) ListChatMembers(chatID string, pageSize int, pageToken string, asUser bool) ([]ChatMember, bool, string, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	params := url.Values{}
	params.Set("page_size", fmt.Sprintf("%d", pageSize))
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}
	path := fmt.Sprintf("/im/v1/chats/%s/members?%s", chatID, params.Encode())
	var resp ChatMembersResponse
	if asUser {
		if err := c.Get(path, &resp); err != nil {
			return nil, false, "", err
		}
	} else {
		if err := c.GetWithTenantToken(path, &resp); err != nil {
			return nil, false, "", err
		}
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	return resp.Data.Items, resp.Data.HasMore, resp.Data.PageToken, nil
}

// AddChatMembers adds members to a chat
func (c *Client) AddChatMembers(chatID string, memberIDs []string, asUser bool) error {
	path := fmt.Sprintf("/im/v1/chats/%s/members?member_id_type=open_id", chatID)
	req := &ModifyChatMembersRequest{IDList: memberIDs}
	var resp BaseResponse
	if asUser {
		if err := c.Post(path, req, &resp); err != nil {
			return err
		}
	} else {
		if err := c.PostWithTenantToken(path, req, &resp); err != nil {
			return err
		}
	}
	return resp.Err()
}

// RemoveChatMembers removes members from a chat
func (c *Client) RemoveChatMembers(chatID string, memberIDs []string, asUser bool) error {
	path := fmt.Sprintf("/im/v1/chats/%s/members?member_id_type=open_id", chatID)
	req := &ModifyChatMembersRequest{IDList: memberIDs}
	var resp BaseResponse
	if asUser {
		if err := c.DeleteWithBody(path, req, &resp); err != nil {
			return err
		}
	} else {
		// No tenant delete-with-body exists, use raw approach
		if err := c.doRequestWithTenantToken("DELETE", path, req, &resp); err != nil {
			return err
		}
	}
	return resp.Err()
}

// PinMessage pins a message in a chat
func (c *Client) PinMessage(messageID string, asUser bool) (*PinnedMessage, error) {
	path := "/im/v1/pins"
	req := &PinMessageRequest{MessageID: messageID}
	var resp PinMessageResponse
	if asUser {
		if err := c.Post(path, req, &resp); err != nil {
			return nil, err
		}
	} else {
		if err := c.PostWithTenantToken(path, req, &resp); err != nil {
			return nil, err
		}
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	if resp.Data != nil {
		return resp.Data.Pin, nil
	}
	return nil, nil
}

// UnpinMessage unpins a message
func (c *Client) UnpinMessage(messageID string, asUser bool) error {
	path := fmt.Sprintf("/im/v1/pins/%s", messageID)
	var resp BaseResponse
	if asUser {
		if err := c.Delete(path, &resp); err != nil {
			return err
		}
	} else {
		if err := c.DeleteWithTenantToken(path, &resp); err != nil {
			return err
		}
	}
	return resp.Err()
}

// ListPinnedMessages lists pinned messages in a chat
func (c *Client) ListPinnedMessages(chatID string, asUser bool) ([]PinnedMessage, error) {
	path := fmt.Sprintf("/im/v1/pins?chat_id=%s", chatID)
	var resp ListPinsResponse
	if asUser {
		if err := c.Get(path, &resp); err != nil {
			return nil, err
		}
	} else {
		if err := c.GetWithTenantToken(path, &resp); err != nil {
			return nil, err
		}
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Items, nil
}

// GetChatLink gets a shareable link for a chat
func (c *Client) GetChatLink(chatID string, asUser bool) (string, error) {
	path := fmt.Sprintf("/im/v1/chats/%s/link", chatID)
	var resp ChatLinkResponse
	if asUser {
		if err := c.Post(path, nil, &resp); err != nil {
			return "", err
		}
	} else {
		if err := c.PostWithTenantToken(path, nil, &resp); err != nil {
			return "", err
		}
	}
	if err := resp.Err(); err != nil {
		return "", err
	}
	return resp.Data.ShareLink, nil
}
```

- [ ] **Step 3: Build to verify**

Run: `cd /Users/yingcong/Code/lark-cli && go build ./internal/api/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/api/chats_write.go internal/api/types.go
git commit -m "Add chat management API methods (create, update, members, pins)"
```

---

## Task 5: Group Chat Operations — Command Layer

**Files:**
- Modify: `internal/cmd/chat.go`

- [ ] **Step 1: Add all chat subcommands to chat.go**

Add the following variables, commands, and init registrations to `internal/cmd/chat.go`. The existing `chatCmd` and `chatSearchCmd` remain unchanged. Add after the existing code:

```go
var (
	chatCreateName        string
	chatCreateDescription string
	chatCreateMembers     []string
	chatUpdateName        string
	chatUpdateDescription string
	chatMemberAddMembers  []string
	chatMemberRemoveMembers []string
	chatPinMessageID      string
	chatAs                string
)

var chatGetCmd = &cobra.Command{
	Use:   "get <chat-id>",
	Short: "Get chat details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		chat, err := client.GetChat(args[0], asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(chat)
	},
}

var chatCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group chat",
	Run: func(cmd *cobra.Command, args []string) {
		if chatCreateName == "" {
			output.Fatalf("VALIDATION_ERROR", "--name is required")
		}
		asUser := chatAs == "user"
		// Resolve emails to open_ids
		memberIDs := resolveMembers(chatCreateMembers)
		client := api.NewClient()
		req := &api.CreateChatRequest{
			Name:        chatCreateName,
			Description: chatCreateDescription,
			UserIDList:  memberIDs,
		}
		resp, err := client.CreateChat(req, asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success": true,
			"chat_id": resp.Data.ChatID,
		})
	},
}

var chatUpdateCmd = &cobra.Command{
	Use:   "update <chat-id>",
	Short: "Update chat name or description",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if chatUpdateName == "" && chatUpdateDescription == "" {
			output.Fatalf("VALIDATION_ERROR", "--name or --description required")
		}
		asUser := chatAs == "user"
		client := api.NewClient()
		req := &api.UpdateChatRequest{
			Name:        chatUpdateName,
			Description: chatUpdateDescription,
		}
		if err := client.UpdateChat(args[0], req, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "chat_id": args[0]})
	},
}

var chatMembersCmd = &cobra.Command{
	Use:   "members <chat-id>",
	Short: "List chat members",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		var allMembers []api.ChatMember
		var pageToken string
		hasMore := true
		for hasMore {
			members, more, nextToken, err := client.ListChatMembers(args[0], 50, pageToken, asUser)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			allMembers = append(allMembers, members...)
			hasMore = more
			pageToken = nextToken
		}
		output.JSON(map[string]interface{}{
			"chat_id": args[0],
			"members": allMembers,
			"count":   len(allMembers),
		})
	},
}

var chatMemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Add or remove chat members",
}

var chatMemberAddCmd = &cobra.Command{
	Use:   "add <chat-id>",
	Short: "Add members to a chat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(chatMemberAddMembers) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--members is required")
		}
		asUser := chatAs == "user"
		memberIDs := resolveMembers(chatMemberAddMembers)
		client := api.NewClient()
		if err := client.AddChatMembers(args[0], memberIDs, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "chat_id": args[0]})
	},
}

var chatMemberRemoveCmd = &cobra.Command{
	Use:   "remove <chat-id>",
	Short: "Remove members from a chat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(chatMemberRemoveMembers) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--members is required")
		}
		asUser := chatAs == "user"
		memberIDs := resolveMembers(chatMemberRemoveMembers)
		client := api.NewClient()
		if err := client.RemoveChatMembers(args[0], memberIDs, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "chat_id": args[0]})
	},
}

var chatPinCmd = &cobra.Command{
	Use:   "pin <chat-id>",
	Short: "Pin a message in a chat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if chatPinMessageID == "" {
			output.Fatalf("VALIDATION_ERROR", "--message-id is required")
		}
		asUser := chatAs == "user"
		client := api.NewClient()
		pin, err := client.PinMessage(chatPinMessageID, asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "pin": pin})
	},
}

var chatUnpinCmd = &cobra.Command{
	Use:   "unpin <chat-id>",
	Short: "Unpin a message",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if chatPinMessageID == "" {
			output.Fatalf("VALIDATION_ERROR", "--message-id is required")
		}
		asUser := chatAs == "user"
		client := api.NewClient()
		if err := client.UnpinMessage(chatPinMessageID, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true})
	},
}

var chatPinsCmd = &cobra.Command{
	Use:   "pins <chat-id>",
	Short: "List pinned messages",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		pins, err := client.ListPinnedMessages(args[0], asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"chat_id": args[0], "pins": pins, "count": len(pins)})
	},
}

var chatLinkCmd = &cobra.Command{
	Use:   "link <chat-id>",
	Short: "Get shareable chat join link",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		link, err := client.GetChatLink(args[0], asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"chat_id": args[0], "share_link": link})
	},
}

// resolveMembers converts emails to open_ids, passes through ou_ IDs
func resolveMembers(members []string) []string {
	var resolved []string
	client := api.NewClient()
	for _, m := range members {
		if strings.Contains(m, "@") {
			users, err := client.LookupUsers(api.UserLookupOptions{Emails: []string{m}})
			if err == nil && len(users) > 0 && users[0].OpenID != "" {
				resolved = append(resolved, users[0].OpenID)
				continue
			}
		}
		resolved = append(resolved, m)
	}
	return resolved
}
```

Update the existing `init()` function to register all new subcommands and flags:

```go
// Add to existing init():
chatCmd.PersistentFlags().StringVar(&chatAs, "as", "user", "Identity: 'bot' or 'user' (default)")

chatCreateCmd.Flags().StringVar(&chatCreateName, "name", "", "Chat name (required)")
chatCreateCmd.Flags().StringVar(&chatCreateDescription, "description", "", "Chat description")
chatCreateCmd.Flags().StringSliceVar(&chatCreateMembers, "members", nil, "Member IDs or emails (comma-separated)")

chatUpdateCmd.Flags().StringVar(&chatUpdateName, "name", "", "New chat name")
chatUpdateCmd.Flags().StringVar(&chatUpdateDescription, "description", "", "New description")

chatMemberAddCmd.Flags().StringSliceVar(&chatMemberAddMembers, "members", nil, "Member IDs or emails to add")
chatMemberRemoveCmd.Flags().StringSliceVar(&chatMemberRemoveMembers, "members", nil, "Member IDs or emails to remove")

chatPinCmd.Flags().StringVar(&chatPinMessageID, "message-id", "", "Message ID to pin")
chatUnpinCmd.Flags().StringVar(&chatPinMessageID, "message-id", "", "Message ID to unpin")

chatMemberCmd.AddCommand(chatMemberAddCmd)
chatMemberCmd.AddCommand(chatMemberRemoveCmd)

chatCmd.AddCommand(chatGetCmd)
chatCmd.AddCommand(chatCreateCmd)
chatCmd.AddCommand(chatUpdateCmd)
chatCmd.AddCommand(chatMembersCmd)
chatCmd.AddCommand(chatMemberCmd)
chatCmd.AddCommand(chatPinCmd)
chatCmd.AddCommand(chatUnpinCmd)
chatCmd.AddCommand(chatPinsCmd)
chatCmd.AddCommand(chatLinkCmd)
```

- [ ] **Step 2: Add missing import for "strings" if not present**

- [ ] **Step 3: Build and test**

Run: `cd /Users/yingcong/Code/lark-cli && go build -o lark ./cmd/lark/`
Then: `./lark chat --help`
Expected: Shows all new subcommands (create, get, update, members, member, pin, unpin, pins, link)

Test: `./lark chat get oc_eff4687c579d81f50cb3a07452b4bbf8`
Expected: JSON with chat details

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/chat.go
git commit -m "Add group chat management commands (create, update, members, pins)"
```

---

## Task 6: Message Search & Forwarding — API Layer

**Files:**
- Create: `internal/api/search.go`
- Modify: `internal/api/types.go`

- [ ] **Step 1: Add message search types to types.go**

Append to `internal/api/types.go`:

```go
// --- Message Search Types ---

// SearchMessagesResponse is the API response for message search
type SearchMessagesResponse struct {
	BaseResponse
	Data *struct {
		Items     []Message `json:"items"`
		HasMore   bool      `json:"has_more"`
		PageToken string    `json:"page_token"`
	} `json:"data"`
}

// BatchGetMessagesResponse is the API response for batch message fetch
type BatchGetMessagesResponse struct {
	BaseResponse
	Data *struct {
		Items []Message `json:"items"`
	} `json:"data"`
}

// ForwardMessageRequest represents a forward request
type ForwardMessageRequest struct {
	ReceiveID string `json:"receive_id"`
}

// MergeForwardRequest represents a merge-forward request
type MergeForwardRequest struct {
	ReceiveID  string   `json:"receive_id"`
	MessageIDs []string `json:"message_id_list"`
}
```

- [ ] **Step 2: Create search.go with API methods**

```go
package api

import (
	"fmt"
	"net/url"
	"strings"
)

// SearchMessages searches messages across chats (user token only)
func (c *Client) SearchMessages(query string, opts *SearchMessagesOptions) ([]Message, bool, string, error) {
	params := url.Values{}
	params.Set("query", query)
	if opts != nil {
		if opts.ChatID != "" {
			params.Set("chat_id", opts.ChatID)
		}
		if opts.SenderID != "" {
			params.Set("sender_id", opts.SenderID)
		}
		if opts.StartTime != "" {
			params.Set("start_time", opts.StartTime)
		}
		if opts.EndTime != "" {
			params.Set("end_time", opts.EndTime)
		}
		if opts.MsgType != "" {
			params.Set("message_type", opts.MsgType)
		}
		if opts.PageSize > 0 {
			params.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.PageToken != "" {
			params.Set("page_token", opts.PageToken)
		}
	}

	path := "/im/v1/messages/search?" + params.Encode()
	var resp SearchMessagesResponse
	// Search is user-token only
	if err := c.Get(path, &resp); err != nil {
		return nil, false, "", err
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	return resp.Data.Items, resp.Data.HasMore, resp.Data.PageToken, nil
}

// SearchMessagesOptions contains optional parameters for SearchMessages
type SearchMessagesOptions struct {
	ChatID    string
	SenderID  string
	StartTime string // Unix timestamp
	EndTime   string // Unix timestamp
	MsgType   string // text, image, file, etc.
	PageSize  int
	PageToken string
}

// BatchGetMessages fetches multiple messages by ID
func (c *Client) BatchGetMessages(messageIDs []string) ([]Message, error) {
	params := url.Values{}
	for _, id := range messageIDs {
		params.Add("message_ids", id)
	}
	path := "/im/v1/messages/get_batch?" + params.Encode()
	var resp BatchGetMessagesResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Items, nil
}

// ForwardMessage forwards a message to another chat
func (c *Client) ForwardMessage(messageID, receiveIDType, receiveID string, asUser bool) (*SendMessageResponse, error) {
	path := fmt.Sprintf("/im/v1/messages/%s/forward?receive_id_type=%s", messageID, receiveIDType)
	req := &ForwardMessageRequest{ReceiveID: receiveID}
	var resp SendMessageResponse
	if asUser {
		if err := c.Post(path, req, &resp); err != nil {
			return nil, err
		}
	} else {
		if err := c.PostWithTenantToken(path, req, &resp); err != nil {
			return nil, err
		}
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MergeForwardMessages merge-forwards multiple messages to another chat
func (c *Client) MergeForwardMessages(messageIDs []string, receiveIDType, receiveID string, asUser bool) (*SendMessageResponse, error) {
	path := fmt.Sprintf("/im/v1/messages/merge_forward?receive_id_type=%s", receiveIDType)
	req := &MergeForwardRequest{
		ReceiveID:  receiveID,
		MessageIDs: messageIDs,
	}
	var resp SendMessageResponse
	if asUser {
		if err := c.Post(path, req, &resp); err != nil {
			return nil, err
		}
	} else {
		if err := c.PostWithTenantToken(path, req, &resp); err != nil {
			return nil, err
		}
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

Note: `strings` import is used by the existing package; verify it's present.

- [ ] **Step 3: Build to verify**

Run: `cd /Users/yingcong/Code/lark-cli && go build ./internal/api/`

- [ ] **Step 4: Commit**

```bash
git add internal/api/search.go internal/api/types.go
git commit -m "Add message search, batch get, and forward API methods"
```

---

## Task 7: Message Search & Forwarding — Command Layer

**Files:**
- Modify: `internal/cmd/msg.go`

- [ ] **Step 1: Add search, get, forward, merge-forward commands to msg.go**

Add variables and commands. The search command with flags, the get command taking comma-separated IDs, forward and merge-forward commands. Follow the same pattern as existing msg subcommands (use `output.JSON`, `output.Fatal`, etc.).

Key commands:
- `msgSearchCmd` (Use: "search <query>") — flags: `--chat-id`, `--sender`, `--after`, `--before`, `--type`, `--limit`
- `msgGetCmd` (Use: "get <message-ids>") — comma-separated IDs
- `msgForwardCmd` (Use: "forward") — flags: `--message-id`, `--to`, `--as`
- `msgMergeForwardCmd` (Use: "merge-forward") — flags: `--message-ids`, `--to`, `--as`

Register in `init()`:
```go
msgCmd.AddCommand(msgSearchCmd)
msgCmd.AddCommand(msgGetCmd)
msgCmd.AddCommand(msgForwardCmd)
msgCmd.AddCommand(msgMergeForwardCmd)
```

The search command always uses user token. Forward/merge-forward respect `--as` flag.

- [ ] **Step 2: Build and test**

Run: `cd /Users/yingcong/Code/lark-cli && go build -o lark ./cmd/lark/`
Then: `./lark msg search "test" --limit 5`
Expected: JSON with matching messages

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/msg.go
git commit -m "Add message search, batch get, and forward commands"
```

---

## Task 8: Tasks CRUD — API Layer

**Files:**
- Create: `internal/api/tasks_write.go`
- Modify: `internal/api/types.go`

- [ ] **Step 1: Add task types to types.go**

Append to `internal/api/types.go`:

```go
// --- Task Write Types ---

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	Summary  string    `json:"summary"`
	Due      *TaskDue  `json:"due,omitempty"`
	Origin   *TaskOrigin `json:"origin,omitempty"`
}

// TaskDue represents a task due date
type TaskDue struct {
	Timestamp string `json:"timestamp,omitempty"` // Unix seconds
	IsAllDay  bool   `json:"is_all_day,omitempty"`
}

// TaskOrigin represents the source of a task
type TaskOrigin struct {
	PlatformI18nName map[string]string `json:"platform_i18n_name,omitempty"`
}

// UpdateTaskRequest represents a request to update a task
type UpdateTaskRequest struct {
	Task       map[string]interface{} `json:"task"`
	UpdateFields []string             `json:"update_fields"`
}

// CreateTaskResponse is the API response for task creation
type CreateTaskResponse struct {
	BaseResponse
	Data *struct {
		Task *Task `json:"task"`
	} `json:"data"`
}

// TaskCommentRequest represents a request to add a comment
type TaskCommentRequest struct {
	Content string `json:"content"`
}

// CreateSubtaskRequest represents a request to create a subtask
type CreateSubtaskRequest struct {
	Summary string   `json:"summary"`
	Due     *TaskDue `json:"due,omitempty"`
}

// SubtaskListResponse is the API response for listing subtasks
type SubtaskListResponse struct {
	BaseResponse
	Data *struct {
		Items     []Task `json:"items"`
		HasMore   bool   `json:"has_more"`
		PageToken string `json:"page_token"`
	} `json:"data"`
}

// TasklistResponse represents a tasklist
type Tasklist struct {
	GUID    string `json:"guid"`
	Name    string `json:"name"`
	Creator *TaskMember `json:"creator,omitempty"`
}

// TasklistListResponse is the API response for listing tasklists
type TasklistListResponse struct {
	BaseResponse
	Data *struct {
		Items     []Tasklist `json:"items"`
		HasMore   bool       `json:"has_more"`
		PageToken string     `json:"page_token"`
	} `json:"data"`
}

// CreateTasklistRequest represents a request to create a tasklist
type CreateTasklistRequest struct {
	Name string `json:"name"`
}

// CreateTasklistResponse is the API response for tasklist creation
type CreateTasklistResponse struct {
	BaseResponse
	Data *struct {
		Tasklist *Tasklist `json:"tasklist"`
	} `json:"data"`
}

// TaskMember represents a task member
type TaskMember struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"` // user
	Role string `json:"role,omitempty"` // assignee, follower
}

// AddTaskMemberRequest represents adding a member to a task
type AddTaskMemberRequest struct {
	Members []TaskMember `json:"members"`
}

// TaskReminderRequest represents setting a reminder
type TaskReminderRequest struct {
	RelativeFireMinute int `json:"relative_fire_minute"`
}
```

- [ ] **Step 2: Create tasks_write.go with API methods**

```go
package api

import (
	"fmt"
	"net/url"
)

// CreateTask creates a new task
func (c *Client) CreateTask(req *CreateTaskRequest) (*Task, error) {
	path := "/task/v2/tasks"
	var resp CreateTaskResponse
	if err := c.Post(path, req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Task, nil
}

// UpdateTask updates a task
func (c *Client) UpdateTask(taskGUID string, fields map[string]interface{}, updateFields []string) (*Task, error) {
	path := fmt.Sprintf("/task/v2/tasks/%s", taskGUID)
	req := &UpdateTaskRequest{
		Task:         fields,
		UpdateFields: updateFields,
	}
	var resp CreateTaskResponse
	if err := c.Patch(path, req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Task, nil
}

// CompleteTask marks a task as completed
func (c *Client) CompleteTask(taskGUID string) error {
	path := fmt.Sprintf("/task/v2/tasks/%s/complete", taskGUID)
	var resp BaseResponse
	if err := c.Post(path, nil, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// ReopenTask reopens a completed task
func (c *Client) ReopenTask(taskGUID string) error {
	path := fmt.Sprintf("/task/v2/tasks/%s/uncomplete", taskGUID)
	var resp BaseResponse
	if err := c.Post(path, nil, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// AddTaskMembers adds members (assignees/followers) to a task
func (c *Client) AddTaskMembers(taskGUID string, members []TaskMember) error {
	path := fmt.Sprintf("/task/v2/tasks/%s/add_members", taskGUID)
	req := &AddTaskMemberRequest{Members: members}
	var resp BaseResponse
	if err := c.Post(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// RemoveTaskMembers removes members from a task
func (c *Client) RemoveTaskMembers(taskGUID string, members []TaskMember) error {
	path := fmt.Sprintf("/task/v2/tasks/%s/remove_members", taskGUID)
	req := &AddTaskMemberRequest{Members: members}
	var resp BaseResponse
	if err := c.Post(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// CreateSubtask creates a subtask under a parent task
func (c *Client) CreateSubtask(parentGUID string, req *CreateSubtaskRequest) (*Task, error) {
	path := fmt.Sprintf("/task/v2/tasks/%s/subtasks", parentGUID)
	var resp CreateTaskResponse
	if err := c.Post(path, req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Task, nil
}

// ListSubtasks lists subtasks of a parent task
func (c *Client) ListSubtasks(parentGUID string, pageSize int, pageToken string) ([]Task, bool, string, error) {
	params := url.Values{}
	if pageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}
	path := fmt.Sprintf("/task/v2/tasks/%s/subtasks?%s", parentGUID, params.Encode())
	var resp SubtaskListResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, false, "", err
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	return resp.Data.Items, resp.Data.HasMore, resp.Data.PageToken, nil
}

// AddTaskComment adds a comment to a task
func (c *Client) AddTaskComment(taskGUID, content string) error {
	path := fmt.Sprintf("/task/v2/tasks/%s/comments", taskGUID)
	req := &TaskCommentRequest{Content: content}
	var resp BaseResponse
	if err := c.Post(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// AddTaskReminder sets a reminder on a task
func (c *Client) AddTaskReminder(taskGUID string, relativeMinutes int) error {
	path := fmt.Sprintf("/task/v2/tasks/%s/reminders", taskGUID)
	req := &TaskReminderRequest{RelativeFireMinute: relativeMinutes}
	var resp BaseResponse
	if err := c.Post(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// ListTasklists lists all tasklists
func (c *Client) ListTasklists(pageSize int, pageToken string) ([]Tasklist, bool, string, error) {
	params := url.Values{}
	if pageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}
	path := "/task/v2/tasklists?" + params.Encode()
	var resp TasklistListResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, false, "", err
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	return resp.Data.Items, resp.Data.HasMore, resp.Data.PageToken, nil
}

// CreateTasklist creates a new tasklist
func (c *Client) CreateTasklist(name string) (*Tasklist, error) {
	path := "/task/v2/tasklists"
	req := &CreateTasklistRequest{Name: name}
	var resp CreateTasklistResponse
	if err := c.Post(path, req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Tasklist, nil
}

// AddTaskToTasklist adds a task to a tasklist
func (c *Client) AddTaskToTasklist(tasklistGUID, taskGUID string) error {
	path := fmt.Sprintf("/task/v2/tasklists/%s/tasks", tasklistGUID)
	req := map[string]string{"task_guid": taskGUID}
	var resp BaseResponse
	if err := c.Post(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}
```

- [ ] **Step 3: Build to verify**

Run: `cd /Users/yingcong/Code/lark-cli && go build ./internal/api/`

- [ ] **Step 4: Commit**

```bash
git add internal/api/tasks_write.go internal/api/types.go
git commit -m "Add task CRUD API methods (create, update, complete, subtasks, comments)"
```

---

## Task 9: Tasks CRUD — Command Layer

**Files:**
- Modify: `internal/cmd/task.go`

- [ ] **Step 1: Expand task.go with all CRUD subcommands**

Add create, update, complete, reopen, assign, subtask (create/list), comment, remind, tasklist (list/create/add) subcommands. Follow the same cobra pattern as existing commands. Key implementation notes:

- `--due` flag accepts ISO 8601 or date-only, parsed to Unix timestamp using existing `parseTimeArg` from msg.go (extract to shared util or duplicate)
- `--assignee` accepts `ou_xxx` or email (use `resolveMembers` from chat.go — extract to shared util)
- All commands use user token (tasks v2 API is user-token based)

Register in `init()`:
```go
taskCmd.AddCommand(taskCreateCmd)
taskCmd.AddCommand(taskUpdateCmd)
taskCmd.AddCommand(taskCompleteCmd)
taskCmd.AddCommand(taskReopenCmd)
taskCmd.AddCommand(taskAssignCmd)
taskCmd.AddCommand(taskSubtaskCmd)     // parent for subtask create/list
taskCmd.AddCommand(taskCommentCmd)
taskCmd.AddCommand(taskRemindCmd)
taskCmd.AddCommand(taskTasklistCmd)    // parent for tasklist list/create/add
```

- [ ] **Step 2: Build and test**

Run: `cd /Users/yingcong/Code/lark-cli && go build -o lark ./cmd/lark/`
Then: `./lark task --help`
Expected: Shows all new subcommands

Test: `./lark task create --summary "Test task" --due 2026-04-20`
Expected: JSON with created task

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/task.go
git commit -m "Add task CRUD commands (create, update, complete, assign, subtasks, comments)"
```

---

## Task 10: Bitable CRUD — API Layer

**Files:**
- Create: `internal/api/bitable_write.go`
- Modify: `internal/api/types.go`

- [ ] **Step 1: Add bitable write types to types.go**

Append types for: `CreateBitableRequest/Response`, `CreateTableRequest/Response`, `CreateFieldRequest/Response`, `CreateRecordRequest/Response`, `BatchCreateRecordsRequest/Response`, `UpdateRecordRequest`, `SearchRecordsRequest/Response`, `UploadAttachmentResponse`.

- [ ] **Step 2: Create bitable_write.go**

Methods needed:
- `CreateBitable(name string) (string, error)` — returns app_token
- `GetBitable(appToken string) (*BitableApp, error)`
- `CreateTable(appToken, name string) (string, error)` — returns table_id
- `DeleteTable(appToken, tableID string) error`
- `CreateField(appToken, tableID string, req *CreateFieldRequest) (*BitableField, error)`
- `UpdateField(appToken, tableID, fieldID string, req *UpdateFieldRequest) error`
- `DeleteField(appToken, tableID, fieldID string) error`
- `CreateRecord(appToken, tableID string, fields map[string]interface{}) (*BitableRecord, error)`
- `UpdateRecord(appToken, tableID, recordID string, fields map[string]interface{}) error`
- `DeleteRecord(appToken, tableID, recordID string) error`
- `BatchCreateRecords(appToken, tableID string, records []map[string]interface{}) ([]string, error)`
- `BatchDeleteRecords(appToken, tableID string, recordIDs []string) error`
- `SearchRecords(appToken, tableID, filter string, opts *BitableRecordOptions) ([]BitableRecord, bool, string, error)`

All methods use user token via `c.Post`/`c.Get`/etc.

- [ ] **Step 3: Build to verify**

- [ ] **Step 4: Commit**

```bash
git add internal/api/bitable_write.go internal/api/types.go
git commit -m "Add bitable CRUD API methods (tables, fields, records, batch ops)"
```

---

## Task 11: Bitable CRUD — Command Layer

**Files:**
- Modify: `internal/cmd/bitable.go`

- [ ] **Step 1: Add subcommands to bitable.go**

New subcommand groups:
- `bitable create --name "Name"` — create a new bitable
- `bitable get --app-token <token>` — get bitable metadata
- `bitable table create/delete` — table management
- `bitable field create/update/delete` — field management
- `bitable record create/update/delete/batch-create/batch-delete/search/upload` — record CRUD

All use `--app-token` and `--table-id` flags. Token handling: if value looks like a URL, extract the token.

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/bitable.go
git commit -m "Add bitable CRUD commands (tables, fields, records)"
```

---

## Task 12: Wiki Write — API Layer

**Files:**
- Create: `internal/api/wiki.go`
- Modify: `internal/api/types.go`

- [ ] **Step 1: Add wiki write types to types.go**

Types: `WikiSpace`, `WikiSpacesResponse`, `CreateWikiNodeRequest`, `CreateWikiNodeResponse`, `MoveWikiNodeRequest`.

- [ ] **Step 2: Create wiki.go with API methods**

Methods (some may be extractable from existing documents.go — check for overlap):
- `ListWikiSpaces() ([]WikiSpace, error)`
- `GetWikiSpace(spaceID string) (*WikiSpace, error)`
- `CreateWikiNode(spaceID string, req *CreateWikiNodeRequest) (*WikiNode, error)`
- `MoveWikiNode(spaceID, nodeToken, parentToken string) error`
- `DeleteWikiNode(spaceID, nodeToken string) error`

Use user token.

- [ ] **Step 3: Build and commit**

```bash
git add internal/api/wiki.go internal/api/types.go
git commit -m "Add wiki space and node management API methods"
```

---

## Task 13: Wiki Write — Command Layer

**Files:**
- Create: `internal/cmd/wiki.go`
- Modify: `internal/cmd/root.go`

- [ ] **Step 1: Create wiki.go with commands**

Commands:
- `wikiCmd` (Use: "wiki") — parent
- `wikiSpacesCmd` (Use: "spaces") — list spaces
- `wikiSpaceGetCmd` (Use: "space <space-id>") — get space details
- `wikiCreateCmd` (Use: "create") — flags: `--space-id`, `--title`, `--parent`
- `wikiMoveCmd` (Use: "move <node-token>") — flags: `--parent`, `--space-id`
- `wikiDeleteCmd` (Use: "delete <node-token>")
- `wikiGetCmd` (Use: "get <node-token>")
- `wikiChildrenCmd` (Use: "children <node-token>")

`PersistentPreRun: validateScopeGroup("documents")`

- [ ] **Step 2: Register wikiCmd in root.go**

Add `rootCmd.AddCommand(wikiCmd)` in `init()`.

- [ ] **Step 3: Build and test**

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/wiki.go internal/cmd/root.go
git commit -m "Add wiki management commands (spaces, create, move, delete)"
```

---

## Task 14: Meetings — API Layer

**Files:**
- Create: `internal/api/meetings.go`
- Modify: `internal/api/types.go`

- [ ] **Step 1: Add meeting types to types.go**

Types: `Meeting`, `MeetingSearchResponse`, `MeetingNotesResponse`, `MeetingRecordingResponse`.

- [ ] **Step 2: Create meetings.go**

Methods:
- `SearchMeetings(query string, opts *SearchMeetingsOptions) ([]Meeting, bool, string, error)`
- `GetMeeting(meetingID string) (*Meeting, error)`
- `GetMeetingNotes(meetingID string) (*MeetingNotes, error)`
- `GetMeetingRecording(meetingID string) (string, error)` — returns minute_token, then delegates to existing minutes API

Use user token.

- [ ] **Step 3: Build and commit**

```bash
git add internal/api/meetings.go internal/api/types.go
git commit -m "Add video conferencing API methods (search, notes, recordings)"
```

---

## Task 15: Meetings — Command Layer

**Files:**
- Create: `internal/cmd/meetings.go`
- Modify: `internal/cmd/root.go`

- [ ] **Step 1: Create meetings.go**

Commands:
- `meetingsCmd` (Use: "meetings") — parent
- `meetingsSearchCmd` (Use: "search [query]") — flags: `--after`, `--before`, `--organizer`, `--participant`, `--limit`
- `meetingsGetCmd` (Use: "get <meeting-id>")
- `meetingsNotesCmd` (Use: "notes <meeting-id>")
- `meetingsTranscriptCmd` (Use: "transcript <meeting-id>") — flags: `--format` (txt/srt)
- `meetingsRecordingCmd` (Use: "recording <meeting-id>")

`PersistentPreRun: validateScopeGroup("meetings")`

Transcript command: resolve meeting-id to minute_token via meetings API, then delegate to existing `GetMinuteTranscript`.

- [ ] **Step 2: Register meetingsCmd in root.go**

- [ ] **Step 3: Build and test**

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/meetings.go internal/cmd/root.go
git commit -m "Add meetings commands (search, notes, transcript, recording)"
```

---

## Task 16: Real-time Events — API Layer

**Files:**
- Create: `internal/api/events.go`

- [ ] **Step 1: Create events.go with WebSocket subscription**

This is architecturally different from other features — uses WebSocket instead of REST.

```go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yjwong/lark-cli/internal/auth"
)

// EventSubscribeOptions holds subscription filter options
type EventSubscribeOptions struct {
	EventType string         // Filter by event type
	Match     *regexp.Regexp // Filter by regex on payload
	Compact   bool           // Compact output format
}

// SubscribeEvents connects to the Lark event callback endpoint
// and streams events as NDJSON to the provided writer.
// Uses long-polling since Lark doesn't expose a public WebSocket for events.
// The actual implementation uses the callback URL approach:
// 1. Start a local HTTP server to receive event callbacks
// 2. Register the callback URL with the Lark app
// Note: For the initial implementation, we use the long-poll approach via
// the /open-apis/event/v1/outbound/events endpoint.
func (c *Client) SubscribeEvents(writer io.Writer, opts *EventSubscribeOptions) error {
	if err := auth.EnsureValidTenantToken(); err != nil {
		return err
	}

	// Lark event subscription uses a local callback server
	// For now, use the event outbound polling approach
	token := auth.GetTenantTokenStore().GetAccessToken()

	for {
		url := baseURL + "/event/v1/outbound/events"
		req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Reconnect on failure
			time.Sleep(5 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if len(body) == 0 || string(body) == "{}" {
			time.Sleep(2 * time.Second)
			continue
		}

		// Apply filters
		if opts != nil {
			if opts.EventType != "" {
				var event map[string]interface{}
				json.Unmarshal(body, &event)
				header, _ := event["header"].(map[string]interface{})
				eventType, _ := header["event_type"].(string)
				if eventType != opts.EventType {
					continue
				}
			}
			if opts.Match != nil && !opts.Match.Match(body) {
				continue
			}
			if opts.Compact {
				// Compact: single line
				var compact json.RawMessage = body
				body, _ = json.Marshal(compact)
			}
		}

		fmt.Fprintf(writer, "%s\n", body)
	}
}
```

Note: The actual Lark event system uses either webhook callbacks or a WebSocket endpoint. The implementation above is a starting point — the real WebSocket endpoint is `wss://open.larksuite.com/open-apis/event/ws`. This will need refinement based on the actual Lark WebSocket protocol documentation. Use `lark api` as fallback for now.

- [ ] **Step 2: Build and commit**

```bash
git add internal/api/events.go
git commit -m "Add event subscription API (initial implementation)"
```

---

## Task 17: Real-time Events — Command Layer

**Files:**
- Create: `internal/cmd/events.go`
- Modify: `internal/cmd/root.go`

- [ ] **Step 1: Create events.go**

Commands:
- `eventsCmd` (Use: "events") — parent
- `eventsSubscribeCmd` (Use: "subscribe") — flags: `--type`, `--match`, `--output`, `--compact`

The subscribe command runs indefinitely, streaming events to stdout or `--output` file.

- [ ] **Step 2: Register eventsCmd in root.go**

- [ ] **Step 3: Build and test**

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/events.go internal/cmd/root.go
git commit -m "Add events subscribe command for real-time event streaming"
```

---

## Task 18: Update Skill Files

**Files:**
- Modify: `~/.claude/skills/lark-messages/skill.md`
- Modify: `~/.claude/skills/lark-tasks/skill.md`
- Modify: `~/.claude/skills/lark-bitable/skill.md`
- Create: `~/.claude/skills/lark-api/skill.md`
- Create: `~/.claude/skills/lark-meetings/skill.md`
- Create: `~/.claude/skills/lark-events/skill.md`

- [ ] **Step 1: Create lark-api skill**

Create `~/.claude/skills/lark-api/skill.md` with frontmatter, description of `lark api` command, all flags, examples, and discovery via `lark api list`.

- [ ] **Step 2: Update lark-messages skill**

Add sections for: group chat management (`lark chat create/get/update/members/pin`), message search (`lark msg search`), batch get (`lark msg get`), forwarding (`lark msg forward/merge-forward`). Document `--as user` for all new commands.

- [ ] **Step 3: Update lark-tasks skill**

Replace read-only documentation with full CRUD: create, update, complete, reopen, assign, subtask, comment, remind, tasklist management.

- [ ] **Step 4: Update lark-bitable skill**

Add sections for: table create/delete, field create/update/delete, record CRUD, batch operations, search, file uploads.

- [ ] **Step 5: Create lark-meetings skill**

Create `~/.claude/skills/lark-meetings/skill.md` documenting: search, get, notes, transcript, recording commands.

- [ ] **Step 6: Create lark-events skill**

Create `~/.claude/skills/lark-events/skill.md` documenting: subscribe command with filters.

- [ ] **Step 7: Commit all skill updates**

```bash
git add ~/.claude/skills/lark-*/skill.md
git commit -m "Update all lark skill files with new capabilities"
```

---

## Task 19: Final Build & Integration Test

- [ ] **Step 1: Full build**

```bash
cd /Users/yingcong/Code/lark-cli && go build -o lark ./cmd/lark/
```

- [ ] **Step 2: Re-auth with all new scopes**

```bash
./lark auth login --add --scopes messages,tasks,bitable,documents,meetings
```

- [ ] **Step 3: Smoke test each feature**

```bash
# Raw API
./lark api list
./lark api GET /open-apis/im/v1/chats --params '{"page_size":2}' --as user

# Chat ops
./lark chat get oc_eff4687c579d81f50cb3a07452b4bbf8

# Message search
./lark msg search "test" --limit 3

# Tasks
./lark task list
./lark task create --summary "Smoke test" --due 2026-04-20

# Bitable
./lark bitable tables <an-existing-app-token>

# Wiki
./lark wiki spaces

# Meetings
./lark meetings search --limit 3
```

- [ ] **Step 4: Commit version bump**

Update version in `internal/cmd/root.go` or build ldflags, then:
```bash
git commit -m "Lark CLI feature expansion: raw API + 7 ergonomic domains"
```
