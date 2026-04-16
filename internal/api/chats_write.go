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

// DeleteChat disbands a group chat (requires owner or admin privileges)
func (c *Client) DeleteChat(chatID string, asUser bool) error {
	path := fmt.Sprintf("/im/v1/chats/%s", chatID)
	var resp BaseResponse
	var err error
	if asUser {
		err = c.Delete(path, &resp)
	} else {
		err = c.DeleteWithTenantToken(path, &resp)
	}
	if err != nil {
		return err
	}
	return resp.Err()
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
