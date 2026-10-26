package api

import (
	"fmt"
	"net/url"
	"strings"
)

type TaskSearchFilter struct {
	CreatorIDs  []string `json:"creator_ids,omitempty"`
	AssigneeIDs []string `json:"assignee_ids,omitempty"`
	FollowerIDs []string `json:"follower_ids,omitempty"`
	IsCompleted *bool    `json:"is_completed,omitempty"`
}

type TaskSearchOptions struct {
	Query     string            `json:"query"`
	Filter    *TaskSearchFilter `json:"filter,omitempty"`
	PageToken string            `json:"-"`
}

type TaskSearchResult struct {
	ID          string                 `json:"id"`
	DisplayInfo string                 `json:"display_info"`
	MetaData    map[string]interface{} `json:"meta_data,omitempty"`
}

type TaskSearchPage struct {
	Items     []TaskSearchResult `json:"items"`
	HasMore   bool               `json:"has_more"`
	PageToken string             `json:"page_token,omitempty"`
	Notice    string             `json:"notice,omitempty"`
}

func (c *Client) SearchTasks(opts TaskSearchOptions) (*TaskSearchPage, error) {
	opts.Query = strings.TrimSpace(opts.Query)
	if opts.Query == "" && (opts.Filter == nil || (len(opts.Filter.CreatorIDs) == 0 && len(opts.Filter.AssigneeIDs) == 0 && len(opts.Filter.FollowerIDs) == 0 && opts.Filter.IsCompleted == nil)) {
		return nil, fmt.Errorf("task search requires a query or filter")
	}
	params := url.Values{}
	params.Set("user_id_type", "open_id")
	if opts.PageToken != "" {
		params.Set("page_token", opts.PageToken)
	}
	var response struct {
		BaseResponse
		Data *TaskSearchPage `json:"data"`
	}
	if err := c.Post("/task/v2/tasks/search?"+params.Encode(), opts, &response); err != nil {
		return nil, err
	}
	if err := response.Err(); err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, fmt.Errorf("task search response missing data")
	}
	return response.Data, nil
}

type TasklistMembership struct {
	TasklistGUID string `json:"tasklist_guid"`
	SectionGUID  string `json:"section_guid,omitempty"`
}
