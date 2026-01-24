package api

import (
	"fmt"
	"net/url"
)

// GetWikiNode retrieves wiki node information
// nodeToken: the wiki node token from the wiki URL
func (c *Client) GetWikiNode(nodeToken string) (*WikiNode, error) {
	path := fmt.Sprintf("/wiki/v2/spaces/get_node?token=%s",
		url.QueryEscape(nodeToken))

	var resp WikiNodeResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error %d: %s", resp.Code, resp.Msg)
	}

	return resp.Data.Node, nil
}

// GetWikiNodeChildren retrieves the immediate children of a wiki node
// spaceID: the wiki space ID
// parentNodeToken: the parent node token
func (c *Client) GetWikiNodeChildren(spaceID, parentNodeToken string) ([]WikiNode, error) {
	var allItems []WikiNode
	var pageToken string

	for {
		params := url.Values{}
		params.Set("parent_node_token", parentNodeToken)
		params.Set("page_size", "50")
		if pageToken != "" {
			params.Set("page_token", pageToken)
		}

		path := fmt.Sprintf("/wiki/v2/spaces/%s/nodes?%s",
			url.PathEscape(spaceID), params.Encode())

		var resp ListWikiChildrenResponse
		if err := c.Get(path, &resp); err != nil {
			return nil, err
		}

		if resp.Code != 0 {
			return nil, fmt.Errorf("API error %d: %s", resp.Code, resp.Msg)
		}

		allItems = append(allItems, resp.Data.Items...)

		if !resp.Data.HasMore {
			break
		}
		pageToken = resp.Data.PageToken
	}

	return allItems, nil
}

// SearchWikiNodes searches wiki nodes by keyword
// query: search keyword (required)
// spaceID: filter to a specific wiki space (optional)
// nodeID: search within a node and its children - requires spaceID (optional)
func (c *Client) SearchWikiNodes(query, spaceID, nodeID string) ([]WikiSearchItem, error) {
	var allItems []WikiSearchItem
	var pageToken string

	for {
		// Build request body
		body := map[string]interface{}{
			"query":     query,
			"page_size": 50,
		}
		if spaceID != "" {
			body["space_id"] = spaceID
		}
		if nodeID != "" {
			body["node_id"] = nodeID
		}
		if pageToken != "" {
			body["page_token"] = pageToken
		}

		var resp WikiSearchResponse
		if err := c.Post("/wiki/v2/nodes/search", body, &resp); err != nil {
			return nil, err
		}

		if resp.Code != 0 {
			return nil, fmt.Errorf("API error %d: %s", resp.Code, resp.Msg)
		}

		allItems = append(allItems, resp.Data.Items...)

		if !resp.Data.HasMore {
			break
		}
		pageToken = resp.Data.PageToken
	}

	return allItems, nil
}
