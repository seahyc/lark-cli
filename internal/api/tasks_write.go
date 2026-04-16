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
