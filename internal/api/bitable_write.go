package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// CreateBitable creates a new Bitable app and returns the app_token.
func (c *Client) CreateBitable(name, folderToken string) (*BitableApp, error) {
	req := &CreateBitableRequest{Name: name, FolderToken: folderToken}
	var resp CreateBitableResponse
	if err := c.Post("/bitable/v1/apps", req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.App, nil
}

// GetBitable returns metadata for a Bitable app.
func (c *Client) GetBitable(appToken string) (*BitableApp, error) {
	path := fmt.Sprintf("/bitable/v1/apps/%s", url.PathEscape(appToken))
	var resp GetBitableResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.App, nil
}

// CreateTable creates a new table and returns its table_id.
func (c *Client) CreateTable(appToken, name string) (string, error) {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables", url.PathEscape(appToken))
	req := &CreateTableRequest{}
	req.Table.Name = name
	var resp CreateTableResponse
	if err := c.Post(path, req, &resp); err != nil {
		return "", err
	}
	if err := resp.Err(); err != nil {
		return "", err
	}
	return resp.Data.TableID, nil
}

// DeleteTable deletes a table.
func (c *Client) DeleteTable(appToken, tableID string) error {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s",
		url.PathEscape(appToken), url.PathEscape(tableID))
	var resp BaseResponse
	if err := c.Delete(path, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// CreateField adds a field (column) to a table.
func (c *Client) CreateField(appToken, tableID string, req *CreateFieldRequest) (*BitableField, error) {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields",
		url.PathEscape(appToken), url.PathEscape(tableID))
	var resp CreateFieldResponse
	if err := c.Post(path, req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Field, nil
}

// UpdateField updates an existing field.
func (c *Client) UpdateField(appToken, tableID, fieldID string, req *UpdateFieldRequest) error {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields/%s",
		url.PathEscape(appToken), url.PathEscape(tableID), url.PathEscape(fieldID))
	var resp BaseResponse
	if err := c.Put(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// DeleteField deletes a field.
func (c *Client) DeleteField(appToken, tableID, fieldID string) error {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields/%s",
		url.PathEscape(appToken), url.PathEscape(tableID), url.PathEscape(fieldID))
	var resp BaseResponse
	if err := c.Delete(path, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// CreateRecord adds a single record to a table.
func (c *Client) CreateRecord(appToken, tableID string, fields map[string]interface{}) (*BitableRecord, error) {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records",
		url.PathEscape(appToken), url.PathEscape(tableID))
	req := &CreateRecordRequest{Fields: fields}
	var resp CreateRecordResponse
	if err := c.Post(path, req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Record, nil
}

// UpdateRecord updates a record's fields.
func (c *Client) UpdateRecord(appToken, tableID, recordID string, fields map[string]interface{}) error {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s",
		url.PathEscape(appToken), url.PathEscape(tableID), url.PathEscape(recordID))
	req := &UpdateRecordRequest{Fields: fields}
	var resp BaseResponse
	if err := c.Put(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// DeleteRecord deletes a record.
func (c *Client) DeleteRecord(appToken, tableID, recordID string) error {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s",
		url.PathEscape(appToken), url.PathEscape(tableID), url.PathEscape(recordID))
	var resp BaseResponse
	if err := c.Delete(path, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// BatchCreateRecords creates multiple records in one call.
func (c *Client) BatchCreateRecords(appToken, tableID string, records []map[string]interface{}) ([]BitableRecord, error) {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/batch_create",
		url.PathEscape(appToken), url.PathEscape(tableID))
	recs := make([]CreateRecordRequest, len(records))
	for i, f := range records {
		recs[i] = CreateRecordRequest{Fields: f}
	}
	req := &BatchCreateRecordsRequest{Records: recs}
	var resp BatchCreateRecordsResponse
	if err := c.Post(path, req, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Records, nil
}

// BatchDeleteRecords deletes multiple records by id.
func (c *Client) BatchDeleteRecords(appToken, tableID string, recordIDs []string) error {
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/batch_delete",
		url.PathEscape(appToken), url.PathEscape(tableID))
	req := &BatchDeleteRecordsRequest{Records: recordIDs}
	var resp BaseResponse
	if err := c.Post(path, req, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// SearchRecords searches records using filter/sort expressions.
// `filter` is a FormulaBuilder-style string passed through as raw JSON.
func (c *Client) SearchRecords(appToken, tableID string, req *SearchRecordsRequest, pageSize int, pageToken string) ([]BitableRecord, bool, string, error) {
	params := url.Values{}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/search",
		url.PathEscape(appToken), url.PathEscape(tableID))
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	if req == nil {
		req = &SearchRecordsRequest{}
	}
	var resp SearchRecordsResponse
	if err := c.Post(path, req, &resp); err != nil {
		return nil, false, "", err
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	return resp.Data.Items, resp.Data.HasMore, resp.Data.PageToken, nil
}
