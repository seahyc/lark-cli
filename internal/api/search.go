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
