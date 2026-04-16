package api

import (
	"fmt"
	"net/url"
)

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

// SearchMessages searches messages across chats (user token only)
func (c *Client) SearchMessages(query string, opts *SearchMessagesOptions) ([]SearchMessageResult, bool, string, error) {
	// Query params for pagination
	qp := url.Values{}
	if opts != nil {
		if opts.PageSize > 0 {
			qp.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.PageToken != "" {
			qp.Set("page_token", opts.PageToken)
		}
	}
	path := "/im/v1/messages/search"
	if encoded := qp.Encode(); encoded != "" {
		path += "?" + encoded
	}

	// Body with filters
	body := map[string]interface{}{"query": query}
	if opts != nil {
		if opts.ChatID != "" {
			body["chat_ids"] = []string{opts.ChatID}
		}
		if opts.SenderID != "" {
			body["from_ids"] = []string{opts.SenderID}
		}
		if opts.StartTime != "" {
			body["start_time"] = opts.StartTime
		}
		if opts.EndTime != "" {
			body["end_time"] = opts.EndTime
		}
		if opts.MsgType != "" {
			body["message_type"] = opts.MsgType
		}
	}

	var resp SearchMessagesResponse
	// Search is user-token only, POST
	if err := c.Post(path, body, &resp); err != nil {
		return nil, false, "", err
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	return resp.Data.Items, resp.Data.HasMore, resp.Data.PageToken, nil
}

// BatchGetMessages fetches multiple messages by ID
func (c *Client) BatchGetMessages(messageIDs []string) ([]Message, error) {
	// Lark has no batch endpoint — fetch each message individually.
	results := make([]Message, 0, len(messageIDs))
	for _, id := range messageIDs {
		var resp BatchGetMessagesResponse
		path := fmt.Sprintf("/im/v1/messages/%s", id)
		if err := c.Get(path, &resp); err != nil {
			return nil, err
		}
		if err := resp.Err(); err != nil {
			return nil, err
		}
		results = append(results, resp.Data.Items...)
	}
	return results, nil
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
