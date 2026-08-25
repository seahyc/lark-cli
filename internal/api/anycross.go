package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultAnyCrossBaseURL = "https://anycross-sg.larksuite.com"

// AnyCrossClient calls the browser-console API used by Lark AnyCross. AnyCross
// does not currently expose these workflow-management endpoints through Lark's
// public OpenAPI, so callers must provide an authenticated console session.
type AnyCrossClient struct {
	baseURL    string
	cookie     string
	csrfToken  string
	httpClient *http.Client
}

// NewAnyCrossClient creates a client from environment configuration.
//
// Required:
//   - ANYCROSS_SESSION_COOKIE: the Cookie header value from an authenticated
//     AnyCross browser session.
//
// POST-based reads (projects and logs) also require ANYCROSS_CSRF_TOKEN.
func NewAnyCrossClient() (*AnyCrossClient, error) {
	base := strings.TrimRight(os.Getenv("ANYCROSS_BASE_URL"), "/")
	if base == "" {
		base = defaultAnyCrossBaseURL
	}
	cookie := strings.TrimSpace(os.Getenv("ANYCROSS_SESSION_COOKIE"))
	if cookie == "" {
		return nil, fmt.Errorf("ANYCROSS_SESSION_COOKIE is not set; copy the Cookie header value from an authenticated AnyCross console request")
	}
	return &AnyCrossClient{
		baseURL:   base,
		cookie:    cookie,
		csrfToken: strings.TrimSpace(os.Getenv("ANYCROSS_CSRF_TOKEN")),
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil
}

func newAnyCrossClientForTest(baseURL, cookie, csrfToken string, httpClient *http.Client) *AnyCrossClient {
	return &AnyCrossClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		cookie:     cookie,
		csrfToken:  csrfToken,
		httpClient: httpClient,
	}
}

type anyCrossEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (c *AnyCrossClient) do(method, path string, body interface{}, result interface{}) error {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode AnyCross request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, c.baseURL+path, requestBody)
	if err != nil {
		return fmt.Errorf("create AnyCross request: %w", err)
	}
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "lark-cli/anycross")
	if body != nil {
		if c.csrfToken == "" {
			return fmt.Errorf("ANYCROSS_CSRF_TOKEN is required for this AnyCross request")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-csrf-token", c.csrfToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("AnyCross request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read AnyCross response: %w", err)
	}
	contentType := resp.Header.Get("Content-Type")
	if resp.Request != nil && strings.Contains(resp.Request.URL.Host, "accounts.larksuite.com") {
		return fmt.Errorf("AnyCross session is not authenticated or has expired")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("AnyCross API %s %s returned HTTP %d: %s", method, path, resp.StatusCode, truncate(string(payload), 500))
	}
	if strings.Contains(contentType, "text/html") {
		return fmt.Errorf("AnyCross returned HTML instead of JSON; the console session may have expired")
	}

	var envelope anyCrossEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode AnyCross response: %w", err)
	}
	if envelope.Code != 0 {
		return fmt.Errorf("AnyCross API error %d: %s", envelope.Code, envelope.Msg)
	}
	if result == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return fmt.Errorf("decode AnyCross response data: %w", err)
	}
	return nil
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

type AnyCrossTenant struct {
	TenantID string `json:"tenantId"`
	Name     string `json:"name"`
	Brand    string `json:"brand"`
}

func (c *AnyCrossClient) CurrentTenant() (*AnyCrossTenant, error) {
	var tenant AnyCrossTenant
	if err := c.do(http.MethodGet, "/webapi/tenant_management/tenants/current", nil, &tenant); err != nil {
		return nil, err
	}
	return &tenant, nil
}

type AnyCrossProject struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenantId"`
	Name        string                 `json:"name"`
	Description string                 `json:"desc"`
	UpdateTime  int64                  `json:"updateTime"`
	Role        string                 `json:"role"`
	ProjectType string                 `json:"projectType"`
	Flow        map[string]interface{} `json:"flow,omitempty"`
}

type AnyCrossProjectPage struct {
	Projects []AnyCrossProject `json:"projects"`
	Total    int               `json:"total"`
}

func (c *AnyCrossClient) ListProjects(query string, page, pageSize int) (*AnyCrossProjectPage, error) {
	tenant, err := c.CurrentTenant()
	if err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 40
	}
	body := map[string]interface{}{
		"param": query,
		"filterQuery": map[string]interface{}{
			"projectType":     "normal",
			"tenantId":        tenant.TenantID,
			"ignoreQueryDesc": true,
		},
		"pageQuery": map[string]int{"page": page, "pageSize": pageSize},
		"sortType":  1,
	}
	var result AnyCrossProjectPage
	if err := c.do(http.MethodPost, "/webapi/project/detail/batch", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type AnyCrossResourceNode struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Type               int                    `json:"type"`
	ResourceID         string                 `json:"resourceID"`
	Description        string                 `json:"desc"`
	Status             int                    `json:"status"`
	UpdateTime         int64                  `json:"updateTime"`
	ChildResourceNodes []AnyCrossResourceNode `json:"childResourceNodes,omitempty"`
}

type AnyCrossWorkflowSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Folder      string `json:"folder,omitempty"`
	Status      int    `json:"status"`
	UpdateTime  int64  `json:"update_time"`
}

func (c *AnyCrossClient) ListWorkflows(projectID string) ([]AnyCrossWorkflowSummary, error) {
	var root AnyCrossResourceNode
	path := "/webapi/projects/" + url.PathEscape(projectID) + "/files"
	if err := c.do(http.MethodGet, path, nil, &root); err != nil {
		return nil, err
	}
	var workflows []AnyCrossWorkflowSummary
	flattenWorkflows(root.ChildResourceNodes, "", &workflows)
	return workflows, nil
}

func flattenWorkflows(nodes []AnyCrossResourceNode, folder string, out *[]AnyCrossWorkflowSummary) {
	for _, node := range nodes {
		if node.Type == 2 && node.ResourceID != "" {
			*out = append(*out, AnyCrossWorkflowSummary{
				ID: node.ResourceID, Name: node.Name, Description: node.Description,
				Folder: folder, Status: node.Status, UpdateTime: node.UpdateTime,
			})
		}
		nextFolder := folder
		if node.Type == 3 {
			if nextFolder == "" {
				nextFolder = node.Name
			} else {
				nextFolder += "/" + node.Name
			}
		}
		flattenWorkflows(node.ChildResourceNodes, nextFolder, out)
	}
}

type AnyCrossWorkflow struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	CreatedAt     int64                  `json:"createdAt"`
	UpdatedAt     int64                  `json:"updatedAt"`
	IsPublished   bool                   `json:"isPublished"`
	IsEnabled     bool                   `json:"isEnabled"`
	LatestVersion map[string]interface{} `json:"latestVersion,omitempty"`
	Definition    interface{}            `json:"definition"`
}

type anyCrossWorkflowRaw struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	CreatedAt     int64                  `json:"createdAt"`
	UpdatedAt     int64                  `json:"updatedAt"`
	IsPublished   bool                   `json:"isPublished"`
	IsEnabled     bool                   `json:"isEnabled"`
	LatestVersion map[string]interface{} `json:"latestVersion"`
	Draft         struct {
		Definition string `json:"definition"`
	} `json:"draft"`
}

func (c *AnyCrossClient) getWorkflowRaw(flowID string) (*anyCrossWorkflowRaw, interface{}, error) {
	var raw anyCrossWorkflowRaw
	if err := c.do(http.MethodGet, "/webapi/v2/flows/"+url.PathEscape(flowID), nil, &raw); err != nil {
		return nil, nil, err
	}
	var definition interface{}
	if err := json.Unmarshal([]byte(raw.Draft.Definition), &definition); err != nil {
		return nil, nil, fmt.Errorf("decode AnyCross workflow definition: %w", err)
	}
	return &raw, definition, nil
}

func (c *AnyCrossClient) GetWorkflow(flowID string) (*AnyCrossWorkflow, error) {
	raw, definition, err := c.getWorkflowRaw(flowID)
	if err != nil {
		return nil, err
	}
	return &AnyCrossWorkflow{
		ID: raw.ID, Name: raw.Name, Description: raw.Description,
		CreatedAt: raw.CreatedAt, UpdatedAt: raw.UpdatedAt,
		IsPublished: raw.IsPublished, IsEnabled: raw.IsEnabled,
		LatestVersion: raw.LatestVersion, Definition: RedactAnyCrossSecrets(definition),
	}, nil
}

type AnyCrossCreateWorkflow struct {
	ProjectID        string
	Name             string
	Description      string
	TriggerVersionID string
	FolderID         string
}

// CreateWorkflowDraft creates an unpublished workflow. It never publishes,
// enables, or runs the workflow.
func (c *AnyCrossClient) CreateWorkflowDraft(opts AnyCrossCreateWorkflow) (string, error) {
	if strings.TrimSpace(opts.ProjectID) == "" || strings.TrimSpace(opts.Name) == "" || strings.TrimSpace(opts.TriggerVersionID) == "" {
		return "", fmt.Errorf("project ID, name, and trigger connector version ID are required")
	}
	body := map[string]interface{}{
		"name":        opts.Name,
		"desc":        opts.Description,
		"connectorID": opts.TriggerVersionID,
	}
	if opts.FolderID != "" {
		body["createTo"] = opts.FolderID
	}
	var result struct {
		FlowID string `json:"flowID"`
	}
	path := "/webapi/projects/" + url.PathEscape(opts.ProjectID) + "/flows"
	if err := c.do(http.MethodPost, path, body, &result); err != nil {
		return "", err
	}
	if result.FlowID == "" {
		return "", fmt.Errorf("AnyCross created a workflow but returned no flow ID")
	}
	return result.FlowID, nil
}

type AnyCrossJSONPatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type AnyCrossWorkflowPatchPlan struct {
	FlowID           string      `json:"flow_id"`
	CurrentUpdatedAt int64       `json:"current_updated_at"`
	OperationCount   int         `json:"operation_count"`
	Definition       interface{} `json:"definition"`
	Applied          bool        `json:"applied"`
}

// PatchWorkflowDraft applies a constrained RFC 6902-style patch to the live
// draft in memory, preserving credentials that are never emitted by the CLI.
// Only add, replace, and remove are supported. The expected update timestamp
// prevents overwriting a collaborator's newer draft.
func (c *AnyCrossClient) PatchWorkflowDraft(flowID string, expectedUpdatedAt int64, operations []AnyCrossJSONPatchOperation, apply bool) (*AnyCrossWorkflowPatchPlan, error) {
	raw, definition, err := c.getWorkflowRaw(flowID)
	if err != nil {
		return nil, err
	}
	if expectedUpdatedAt <= 0 {
		return nil, fmt.Errorf("expected updated_at is required")
	}
	if raw.UpdatedAt != expectedUpdatedAt {
		return nil, fmt.Errorf("workflow changed: expected updated_at %d, current value is %d", expectedUpdatedAt, raw.UpdatedAt)
	}
	if len(operations) == 0 {
		return nil, fmt.Errorf("at least one patch operation is required")
	}
	patched, err := applyAnyCrossJSONPatch(definition, operations)
	if err != nil {
		return nil, err
	}
	root, ok := patched.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("workflow definition must remain an object")
	}
	steps, stepsOK := root["steps"].([]interface{})
	structure, structureOK := root["structure"].([]interface{})
	if !stepsOK || !structureOK {
		return nil, fmt.Errorf("workflow definition must contain steps and structure arrays")
	}
	plan := &AnyCrossWorkflowPatchPlan{
		FlowID: flowID, CurrentUpdatedAt: raw.UpdatedAt,
		OperationCount: len(operations), Definition: RedactAnyCrossSecrets(patched),
		Applied: false,
	}
	if !apply {
		return plan, nil
	}
	body := map[string]interface{}{
		"flowId":    flowID,
		"steps":     mustJSONString(steps),
		"structure": mustJSONString(structure),
	}
	if err := c.do(http.MethodPut, "/webapi/projects/-/flows/"+url.PathEscape(flowID)+"/draft", body, nil); err != nil {
		return nil, err
	}
	plan.Applied = true
	return plan, nil
}

func mustJSONString(value interface{}) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func applyAnyCrossJSONPatch(document interface{}, operations []AnyCrossJSONPatchOperation) (interface{}, error) {
	// Round-trip to detach the patched value from the response object.
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	for index, operation := range operations {
		if operation.Op != "add" && operation.Op != "replace" && operation.Op != "remove" {
			return nil, fmt.Errorf("patch %d: unsupported operation %q", index, operation.Op)
		}
		segments, err := parseAnyCrossJSONPointer(operation.Path)
		if err != nil {
			return nil, fmt.Errorf("patch %d: %w", index, err)
		}
		if len(segments) < 2 || (segments[0] != "steps" && segments[0] != "structure") {
			return nil, fmt.Errorf("patch %d: path must target /steps or /structure", index)
		}
		for _, segment := range segments {
			if isSensitiveAnyCrossKey(segment) {
				return nil, fmt.Errorf("patch %d: sensitive path segment %q is not allowed", index, segment)
			}
		}
		if err := mutateAnyCrossJSON(&result, segments, operation); err != nil {
			return nil, fmt.Errorf("patch %d: %w", index, err)
		}
	}
	return result, nil
}

func parseAnyCrossJSONPointer(pointer string) ([]string, error) {
	if pointer == "" || !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("path must be a JSON pointer beginning with /")
	}
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	return parts, nil
}

func mutateAnyCrossJSON(root *interface{}, segments []string, operation AnyCrossJSONPatchOperation) error {
	if len(segments) == 0 {
		return fmt.Errorf("root replacement is not allowed")
	}
	current := root
	for _, segment := range segments[:len(segments)-1] {
		switch node := (*current).(type) {
		case map[string]interface{}:
			next, ok := node[segment]
			if !ok {
				return fmt.Errorf("path segment %q does not exist", segment)
			}
			current = &next
			// Map values are not addressable; write the possibly changed child
			// back while unwinding by delegating the final mutation recursively.
			return mutateAnyCrossMapChild(node, segment, segments[1:], operation)
		case []interface{}:
			return mutateAnyCrossSliceChild(node, segment, segments[1:], operation)
		default:
			return fmt.Errorf("path traverses a non-container at %q", segment)
		}
	}
	return mutateAnyCrossLeaf(root, segments[0], operation)
}

func mutateAnyCrossMapChild(parent map[string]interface{}, key string, remaining []string, operation AnyCrossJSONPatchOperation) error {
	child := parent[key]
	if err := mutateAnyCrossJSON(&child, remaining, operation); err != nil {
		return err
	}
	parent[key] = child
	return nil
}

func mutateAnyCrossSliceChild(parent []interface{}, indexText string, remaining []string, operation AnyCrossJSONPatchOperation) error {
	index, err := parseAnyCrossArrayIndex(indexText, len(parent), false)
	if err != nil {
		return err
	}
	child := parent[index]
	if err := mutateAnyCrossJSON(&child, remaining, operation); err != nil {
		return err
	}
	parent[index] = child
	return nil
}

func mutateAnyCrossLeaf(container *interface{}, key string, operation AnyCrossJSONPatchOperation) error {
	switch node := (*container).(type) {
	case map[string]interface{}:
		_, exists := node[key]
		if operation.Op == "replace" && !exists {
			return fmt.Errorf("replace target %q does not exist", key)
		}
		if operation.Op == "remove" {
			if !exists {
				return fmt.Errorf("remove target %q does not exist", key)
			}
			delete(node, key)
			return nil
		}
		node[key] = operation.Value
		return nil
	case []interface{}:
		allowAppend := operation.Op == "add"
		index, err := parseAnyCrossArrayIndex(key, len(node), allowAppend)
		if err != nil {
			return err
		}
		switch operation.Op {
		case "add":
			if index == len(node) {
				node = append(node, operation.Value)
			} else {
				node = append(node[:index], append([]interface{}{operation.Value}, node[index:]...)...)
			}
		case "replace":
			node[index] = operation.Value
		case "remove":
			node = append(node[:index], node[index+1:]...)
		}
		*container = node
		return nil
	default:
		return fmt.Errorf("patch target is not a container")
	}
}

func parseAnyCrossArrayIndex(value string, length int, allowAppend bool) (int, error) {
	if value == "-" && allowAppend {
		return length, nil
	}
	var index int
	if _, err := fmt.Sscanf(value, "%d", &index); err != nil || index < 0 {
		return 0, fmt.Errorf("invalid array index %q", value)
	}
	limit := length - 1
	if allowAppend {
		limit = length
	}
	if index > limit {
		return 0, fmt.Errorf("array index %d is out of bounds", index)
	}
	return index, nil
}

type AnyCrossLogSearch struct {
	AfterMS    int64
	BeforeMS   int64
	ProjectIDs []string
	FlowIDs    []string
	Statuses   []string
	Page       int
	PageSize   int
	Debug      bool
}

func (c *AnyCrossClient) SearchLogs(opts AnyCrossLogSearch) (map[string]interface{}, error) {
	now := time.Now()
	if opts.BeforeMS == 0 {
		opts.BeforeMS = now.UnixMilli()
	}
	if opts.AfterMS == 0 {
		opts.AfterMS = now.Add(-24 * time.Hour).UnixMilli()
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 || opts.PageSize > 100 {
		opts.PageSize = 20
	}
	body := map[string]interface{}{
		"timeRange":   map[string]int64{"left": opts.AfterMS, "right": opts.BeforeMS},
		"projectList": opts.ProjectIDs, "flowList": opts.FlowIDs,
		"flowSnapshotList": []string{}, "status": opts.Statuses,
		"connectorList": []string{}, "contentList": []string{},
		"asc": false, "orCondition": false,
		"pageSize": opts.PageSize, "page": opts.Page, "debug": opts.Debug,
	}
	var result map[string]interface{}
	if err := c.do(http.MethodPost, "/webapi/flow_log/search_flow_log", body, &result); err != nil {
		return nil, err
	}
	return RedactAnyCrossSecrets(result).(map[string]interface{}), nil
}

func (c *AnyCrossClient) GetLog(flowID, instanceID string, debug bool) (map[string]interface{}, error) {
	baseBody := map[string]interface{}{
		"flowID": flowID, "flowInstanceID": instanceID, "debug": debug,
	}
	var summary map[string]interface{}
	if err := c.do(http.MethodPost, "/webapi/flow_log/get_flow_log", baseBody, &summary); err != nil {
		return nil, err
	}
	nodeBody := map[string]interface{}{
		"flowID": flowID, "flowInstanceID": instanceID, "debug": debug,
		"parentNodeInstanceID": "0", "pageSize": 100, "page": 1,
	}
	var nodes map[string]interface{}
	if err := c.do(http.MethodPost, "/webapi/flow_log/node_tree_log", nodeBody, &nodes); err != nil {
		return nil, err
	}
	return RedactAnyCrossSecrets(map[string]interface{}{
		"summary": summary,
		"nodes":   nodes,
	}).(map[string]interface{}), nil
}

// RedactAnyCrossSecrets removes credentials from workflow definitions and log
// payloads before they reach stdout or an agent context.
func RedactAnyCrossSecrets(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			if isSensitiveAnyCrossKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = RedactAnyCrossSecrets(child)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, child := range typed {
			out[i] = RedactAnyCrossSecrets(child)
		}
		return out
	default:
		return value
	}
}

func isSensitiveAnyCrossKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, marker := range []string{
		"password", "passwd", "secret", "token", "cookie", "credential",
		"authorization", "apikey", "accesskey", "privatekey",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
