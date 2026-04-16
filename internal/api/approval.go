package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ApprovalTask represents a pending approval task assigned to the current user.
type ApprovalTask struct {
	TaskID            string `json:"task_id"`
	InstanceCode      string `json:"instance_code"`
	ApprovalCode      string `json:"approval_code"`
	ApprovalName      string `json:"approval_name,omitempty"`
	StartTime         string `json:"start_time,omitempty"`
	EndTime           string `json:"end_time,omitempty"`
	Status            string `json:"status,omitempty"`
	Title             string `json:"title,omitempty"`
	Type              string `json:"type,omitempty"`
	Link              *struct {
		PCLink     string `json:"pc_link,omitempty"`
		MobileLink string `json:"mobile_link,omitempty"`
	} `json:"link,omitempty"`
}

// ApprovalInstance is the detail view of an approval instance.
type ApprovalInstance struct {
	ApprovalName  string          `json:"approval_name,omitempty"`
	StartTime     string          `json:"start_time,omitempty"`
	EndTime       string          `json:"end_time,omitempty"`
	UserID        string          `json:"user_id,omitempty"`
	OpenID        string          `json:"open_id,omitempty"`
	Status        string          `json:"status,omitempty"`
	InstanceCode  string          `json:"instance_code,omitempty"`
	Form          string          `json:"form,omitempty"` // Raw JSON string
	Department    string          `json:"department_id,omitempty"`
	SerialNumber  string          `json:"serial_number,omitempty"`
	Timeline      json.RawMessage `json:"timeline,omitempty"`
	TaskList      json.RawMessage `json:"task_list,omitempty"`
	CommentList   json.RawMessage `json:"comment_list,omitempty"`
}

// ListApprovalTasks queries the current user's pending approval tasks.
// Uses tenant token; Lark does not expose a user-token variant for this endpoint.
func (c *Client) ListApprovalTasks(userID string, pageSize int, pageToken string, locale string) ([]ApprovalTask, bool, string, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if locale == "" {
		locale = "en-US"
	}
	body := map[string]interface{}{
		"locale":    locale,
		"page_size": pageSize,
	}
	if userID != "" {
		body["user_id"] = userID
	}
	if pageToken != "" {
		body["page_token"] = pageToken
	}

	var resp struct {
		BaseResponse
		Data struct {
			Count     int            `json:"count"`
			TaskList  []ApprovalTask `json:"task_list"`
			PageToken string         `json:"page_token"`
			HasMore   bool           `json:"has_more"`
		} `json:"data"`
	}

	if err := c.PostWithTenantToken("/approval/v4/tasks/query", body, &resp); err != nil {
		return nil, false, "", err
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	return resp.Data.TaskList, resp.Data.HasMore, resp.Data.PageToken, nil
}

// GetApprovalInstance fetches the detail of an approval instance by its instance_code.
func (c *Client) GetApprovalInstance(instanceCode, locale string) (*ApprovalInstance, error) {
	if locale == "" {
		locale = "en-US"
	}
	params := url.Values{}
	params.Set("locale", locale)
	path := fmt.Sprintf("/approval/v4/instances/%s?%s", instanceCode, params.Encode())

	var resp struct {
		BaseResponse
		Data ApprovalInstance `json:"data"`
	}
	if err := c.GetWithTenantToken(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ApproveApprovalTask approves a pending approval task.
// userID (open_id) identifies the approver; required by Lark.
func (c *Client) ApproveApprovalTask(approvalCode, instanceCode, userID, taskID, comment string) error {
	body := map[string]interface{}{
		"approval_code": approvalCode,
		"instance_code": instanceCode,
		"user_id":       userID,
		"task_id":       taskID,
	}
	if comment != "" {
		body["comment"] = comment
	}
	var resp BaseResponse
	if err := c.PostWithTenantToken("/approval/v4/instances/approve", body, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// RejectApprovalTask rejects a pending approval task.
func (c *Client) RejectApprovalTask(approvalCode, instanceCode, userID, taskID, comment string) error {
	body := map[string]interface{}{
		"approval_code": approvalCode,
		"instance_code": instanceCode,
		"user_id":       userID,
		"task_id":       taskID,
	}
	if comment != "" {
		body["comment"] = comment
	}
	var resp BaseResponse
	if err := c.PostWithTenantToken("/approval/v4/instances/reject", body, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// TransferApprovalTask transfers a pending approval task to another user.
func (c *Client) TransferApprovalTask(approvalCode, instanceCode, userID, taskID, transferUserID, comment string) error {
	body := map[string]interface{}{
		"approval_code":    approvalCode,
		"instance_code":    instanceCode,
		"user_id":          userID,
		"task_id":          taskID,
		"transfer_user_id": transferUserID,
	}
	if comment != "" {
		body["comment"] = comment
	}
	var resp BaseResponse
	if err := c.PostWithTenantToken("/approval/v4/instances/transfer", body, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// CancelApprovalInstance cancels an approval instance started by the user.
func (c *Client) CancelApprovalInstance(approvalCode, instanceCode, userID string) error {
	body := map[string]interface{}{
		"approval_code": approvalCode,
		"instance_code": instanceCode,
		"user_id":       userID,
	}
	var resp BaseResponse
	if err := c.PostWithTenantToken("/approval/v4/instances/cancel", body, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// CCApprovalInstance cc's an approval instance to additional users.
func (c *Client) CCApprovalInstance(approvalCode, instanceCode, userID string, ccUserIDs []string, comment string) error {
	body := map[string]interface{}{
		"approval_code":  approvalCode,
		"instance_code":  instanceCode,
		"user_id":        userID,
		"cc_user_ids":    ccUserIDs,
	}
	if comment != "" {
		body["comment"] = comment
	}
	var resp BaseResponse
	if err := c.PostWithTenantToken("/approval/v4/instances/cc", body, &resp); err != nil {
		return err
	}
	return resp.Err()
}
